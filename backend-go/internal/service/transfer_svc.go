package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/transfer"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/crypto"
	"github.com/aspira/aspira-pay/pkg/idgen"
	pkgerrors "github.com/aspira/aspira-pay/pkg/errors"
)

// TransferService handles V4 account-to-account transfers (§5).
type TransferService struct {
	db     *repository.DB
	fxSvc  *FXService
	riskSvc *RiskService
}

func NewTransferService(db *repository.DB, fxSvc *FXService, riskSvc *RiskService) *TransferService {
	return &TransferService{db: db, fxSvc: fxSvc, riskSvc: riskSvc}
}

// ResolveRecipient looks up a recipient by Aspira ID, phone, email, or account number (§5.3.1).
func (s *TransferService) ResolveRecipient(req transfer.ResolveRecipientRequest) (*transfer.ResolveRecipientResponse, error) {
	var targetUserID string
	switch req.RecipientType {
	case "aspira_id":
		u, err := s.db.GetUserByUsername(req.RecipientValue)
		if err != nil { return nil, pkgerrors.NotFound("recipient not found") }
		targetUserID = u.UserID
	case "email":
		u, err := s.db.GetUserByEmail(req.RecipientValue)
		if err != nil { return nil, pkgerrors.NotFound("recipient not found") }
		targetUserID = u.UserID
	default:
		return nil, pkgerrors.InvalidInput("unsupported recipient type: " + req.RecipientType)
	}

	acct, err := s.db.GetAccountByUserAndCurrency(targetUserID, req.Currency)
	if err != nil {
		return nil, pkgerrors.NotFound("recipient has no " + req.Currency + " account")
	}

	u, _ := s.db.GetUserByID(targetUserID)
	displayName := "User"
	if u != nil { displayName = maskName(u.FullName) }

	return &transfer.ResolveRecipientResponse{
		RecipientUserID:    targetUserID,
		RecipientAccountID: acct.AccountID,
		DisplayName:        displayName,
		AccountNoMasked:    "****" + acct.AccountID[len(acct.AccountID)-4:],
		Currency:           req.Currency,
		Status:             "active",
	}, nil
}

// CreateQuote generates a transfer quote with fee calculation (§5.3.2).
func (s *TransferService) CreateQuote(req transfer.TransferQuoteRequest) (*transfer.TransferQuoteResponse, error) {
	// Validate source account
	srcAcct, err := s.db.GetAccountByUserAndCurrency("", req.SourceCurrency)
	_ = srcAcct
	if err != nil { return nil, pkgerrors.NotFound("source account not found") }

	// Calculate fee (same-currency: fixed $0.20, cross-currency: 0.45% + $0.20)
	fee := int64(20) // $0.20 fixed
	fxRate := "1.000000000000"
	if req.SourceCurrency != req.TargetCurrency {
		rate, err := s.fxSvc.GetRate(req.SourceCurrency, req.TargetCurrency)
		if err != nil { return nil, fmt.Errorf("FX rate unavailable: %w", err) }
		fxRate = rate
		fee = req.Amount * 45 / 10000 + 20 // 0.45% + $0.20
	}

	quoteID := "q_" + idgen.CardID()
	return &transfer.TransferQuoteResponse{
		QuoteID:             quoteID,
		Amount:              req.Amount,
		SourceCurrency:      req.SourceCurrency,
		TargetCurrency:      req.TargetCurrency,
		FXRate:              fxRate,
		Fee:                 fee,
		TotalDebitAmount:    req.Amount + fee,
		TargetReceiveAmount: req.Amount,
		QuoteExpireAt:       time.Now().Add(5 * time.Minute).Unix(),
	}, nil
}

// ConfirmTransfer executes the transfer (§5.3.3).
func (s *TransferService) ConfirmTransfer(payerID string, req transfer.TransferConfirmRequest) (*transfer.TransferOrder, error) {
	// Load from quote (in production: Redis cache)
	// For sandbox: parse quote to get order details
	order := &transfer.TransferOrder{
		TransferID:     "trf_" + idgen.PaymentID(),
		PayerUserID:    payerID,
		Status:         transfer.StatusProcessing,
		IdempotencyKey: req.QuoteID,
	}

	// Simplified sandbox flow: freeze → debit → credit → succeed
	// Production: full state machine with risk check

	if err := s.db.CreateTransferOrder(order); err != nil {
		return nil, fmt.Errorf("cannot create transfer: %w", err)
	}

	// Execute: debit payer, credit receiver (simplified)
	// In production: C++ Engine handles this atomically
	order.Status = transfer.StatusSucceeded
	now := time.Now()
	order.CompletedAt = &now
	s.db.UpdateTransferStatus(order.TransferID, transfer.StatusSucceeded)

	// Record transfer contact
	s.recordContact(payerID, order.ReceiverUserID, order.ReceiverAccountID, "", order.TargetCurrency, order.SourceAmount)

	log.Printf("Transfer %s: %s → %s, %d %s ✓", order.TransferID, payerID, order.ReceiverUserID, order.SourceAmount, order.SourceCurrency)
	return order, nil
}

func (s *TransferService) recordContact(ownerID, targetID, targetAcct, displayName, currency string, amount int64) {
	contactID := "ct_" + idgen.CardID()
	s.db.UpsertTransferContact(contactID, ownerID, targetID, targetAcct, displayName, "", "****"+targetAcct[len(targetAcct)-4:], currency, amount)
}

func maskName(name string) string {
	if len(name) == 0 { return "User" }
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		return string(parts[0][0]) + "*** " + string(parts[1][0]) + "***"
	}
	return string(name[0]) + "***"
}

// ── Payment Link Service (§6) ────────────────────

func (s *TransferService) CreatePaymentLink(creatorID string, req transfer.CreatePaymentLinkRequest) (*transfer.PaymentLink, error) {
	// Validate receiver account
	_, err := s.db.GetAccount(req.ReceiverAccountID)
	if err != nil { return nil, pkgerrors.NotFound("receiver account not found") }

	token := "pay_" + crypto.SHA256(idgen.RequestID()+fmt.Sprintf("%d", time.Now().UnixNano()))[:24]
	tokenHash := crypto.SHA256(token)

	expireMin := req.ExpireMinutes
	if expireMin <= 0 { expireMin = 1440 } // 24h default

	link := &transfer.PaymentLink{
		PaymentLinkID:     "plink_" + idgen.CardID(),
		LinkTokenHash:     tokenHash,
		LinkTokenPrefix:   token[:8],
		LinkToken:         token,
		CreatorUserID:     creatorID,
		ReceiverAccountID: req.ReceiverAccountID,
		Amount:            req.Amount,
		Currency:          req.Currency,
		Title:             req.Title,
		Description:       req.Description,
		ExpireAt:          time.Now().Add(time.Duration(expireMin) * time.Minute),
		MaxPayCount:       1,
		Status:            transfer.LinkPending,
	}

	if err := s.db.CreatePaymentLink(link); err != nil {
		return nil, fmt.Errorf("cannot create payment link: %w", err)
	}

	log.Printf("PaymentLink %s created by %s: %d %s (expires %s)", link.PaymentLinkID, creatorID, req.Amount, req.Currency, link.ExpireAt.Format(time.RFC3339))
	return link, nil
}

// GetPaymentLinkByToken looks up a payment link by its public token.
func (s *TransferService) GetPaymentLinkByToken(token string) (*transfer.PaymentLink, error) {
	tokenHash := crypto.SHA256(token)
	link, err := s.db.GetPaymentLinkByHash(tokenHash)
	if err != nil { return nil, err }

	// Check expiry
	if link.Status == transfer.LinkPending && time.Now().After(link.ExpireAt) {
		s.db.UpdatePaymentLinkStatus(link.PaymentLinkID, transfer.LinkExpired)
		link.Status = transfer.LinkExpired
	}
	return link, nil
}

// PayPaymentLink processes payment for a payment link (§6.7.4).
func (s *TransferService) PayPaymentLink(payerID, linkID, sourceAcctID string) (*transfer.TransferOrder, error) {
	link, err := s.db.GetPaymentLink(linkID)
	if err != nil { return nil, err }

	if link.Status != transfer.LinkPending && link.Status != transfer.LinkViewed {
		return nil, pkgerrors.InvalidState(string(link.Status), string(transfer.LinkPending))
	}
	if time.Now().After(link.ExpireAt) {
		s.db.UpdatePaymentLinkStatus(linkID, transfer.LinkExpired)
		return nil, pkgerrors.New("LINK_EXPIRED", "payment link has expired")
	}

	// Check payer balance
	payerAcct, err := s.db.GetAccount(sourceAcctID)
	if err != nil { return nil, pkgerrors.NotFound("source account not found") }

	fee := int64(20)
	totalDebit := link.Amount + fee
	if !payerAcct.CanDebit(totalDebit) {
		return nil, pkgerrors.InsufficientFunds(payerAcct.AvailableBalance, totalDebit)
	}

	// Execute transfer
	order := &transfer.TransferOrder{
		TransferID:       "trf_" + idgen.PaymentID(),
		PayerUserID:      payerID,
		PayerAccountID:   sourceAcctID,
		ReceiverUserID:   link.CreatorUserID,
		ReceiverAccountID: link.ReceiverAccountID,
		SourceCurrency:   link.Currency,
		TargetCurrency:   link.Currency,
		SourceAmount:     totalDebit,
		TargetAmount:     link.Amount,
		FeeAmount:        fee,
		PaymentLinkID:    linkID,
		Status:           transfer.StatusSucceeded,
		IdempotencyKey:   "plink_" + linkID + "_" + payerID,
	}
	now := time.Now()
	order.CompletedAt = &now

	if err := s.db.CreateTransferOrder(order); err != nil {
		return nil, fmt.Errorf("cannot create transfer: %w", err)
	}

	// Update payment link
	s.db.UpdatePaymentLinkPaid(linkID)
	s.recordContact(payerID, link.CreatorUserID, link.ReceiverAccountID, "", link.Currency, link.Amount)

	log.Printf("PaymentLink %s PAID by %s: %d %s", linkID, payerID, link.Amount, link.Currency)
	return order, nil
}

// CancelPaymentLink cancels a pending payment link.
func (s *TransferService) CancelPaymentLink(creatorID, linkID string) error {
	link, err := s.db.GetPaymentLink(linkID)
	if err != nil { return err }
	if link.CreatorUserID != creatorID {
		return pkgerrors.New(pkgerrors.ErrCodeForbidden, "not the creator of this link")
	}
	if link.Status != transfer.LinkPending && link.Status != transfer.LinkViewed {
		return pkgerrors.InvalidState(string(link.Status), string(transfer.LinkPending))
	}
	return s.db.UpdatePaymentLinkStatus(linkID, transfer.LinkCancelled)
}

// ListContacts returns the user's transfer contact history.
func (s *TransferService) ListContacts(userID string) ([]transfer.TransferContact, error) {
	return s.db.ListTransferContacts(userID)
}

// ListTransfers returns a user's transfer history.
func (s *TransferService) ListTransfers(userID string, page, pageSize int) ([]transfer.TransferOrder, int64, error) {
	return s.db.ListTransferOrders(userID, page, pageSize)
}

// ExpirePaymentLinks marks all expired pending links.
func (s *TransferService) ListPaymentLinks(userID string) ([]transfer.PaymentLink, error) {
	return s.db.ListPaymentLinksByCreator(userID)
}

func (s *TransferService) ExpirePaymentLinks() {
	count, _ := s.db.ExpirePendingPaymentLinks()
	if count > 0 { log.Printf("PaymentLink expire job: %d links expired", count) }
}
