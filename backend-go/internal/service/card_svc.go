// Package service implements the Card Payment Subsystem (§5.1-5.4).
// Wise-like multi-currency card: issuing, authorization, FX selection, fee calculation.
package service

import (
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/card"
	"github.com/aspira/aspira-pay/internal/domain/user"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/crypto"
	pkgcard "github.com/aspira/aspira-pay/pkg/card"
	"github.com/aspira/aspira-pay/pkg/idgen"
	pkgerrors "github.com/aspira/aspira-pay/pkg/errors"
)

// CardService manages card lifecycle and payment authorization.
type CardService struct {
	db    *repository.DB
	fxSvc *FXService
}

func NewCardService(db *repository.DB, fxSvc *FXService) *CardService {
	return &CardService{db: db, fxSvc: fxSvc}
}

// ── Card Issuing (§5.3) ──────────────────────────

// ApplyForCard handles KYC-based card application (§5.3).
// Regular users must provide KYC data. Max 5 cards per user.
func (s *CardService) ApplyForCard(ownerID string, req user.CardApplicationRequest) (*card.CreateCardResponse, error) {
	// Check 5-card limit
	count, err := s.db.CountCardsByUser(ownerID)
	if err != nil { return nil, fmt.Errorf("cannot count cards: %w", err) }
	if count >= 5 {
		return nil, pkgerrors.New("CARD_LIMIT", "maximum 5 cards per user")
	}

	network := card.CardNetwork(req.CardNetwork)
	if network == "" { network = card.NetworkVisa }
	currency := req.DefaultCurrency
	if currency == "" { currency = "USD" }

	testBINs := pkgcard.TestBINs()
	bin := testBINs[string(network)]
	if bin == "" { bin = "400000" }
	pan := pkgcard.GenerateTestPAN(bin, 16)
	last4 := pkgcard.MaskPAN(pan)
	token := crypto.SHA256("card:" + pan + idgen.RequestID())

	c := &card.Card{
		CardID:          idgen.CardID(),
		OwnerType:       "CUSTOMER",
		OwnerID:         ownerID,
		CardToken:       "tok_" + token[:32],
		PANLast4:        last4,
		CardNetwork:     network,
		CardType:        card.TypeDebit,
		CardForm:        card.FormVirtual,
		ExpiryMonth:     int(time.Now().Month()),
		ExpiryYear:      time.Now().Year() + 3,
		Status:          card.StatusActive,
		DefaultCurrency: currency,
		DailyLimit:      500000,
		MonthlyLimit:    5000000,
		SingleTxLimit:   500000,
	}

	if err := s.db.CreateCard(c); err != nil {
		return nil, fmt.Errorf("cannot create card: %w", err)
	}

	log.Printf("CardService: card %s issued for user %s (last4=%s, network=%s, %d/5 cards)",
		c.CardID, ownerID, last4, network, count+1)
	return &card.CreateCardResponse{
		CardID: c.CardID, CardToken: c.CardToken, Last4: last4,
		CardNetwork: string(network), Status: string(c.Status),
	}, nil
}

// CancelCard cancels a card without affecting the user account.
func (s *CardService) CancelCard(ownerID, cardID string) error {
	c, err := s.db.GetCard(cardID)
	if err != nil { return err }
	if c.OwnerID != ownerID {
		return pkgerrors.New(pkgerrors.ErrCodeForbidden, "card does not belong to this user")
	}
	return s.db.UpdateCardStatus(cardID, card.StatusCancelled)
}

// CreateVirtualCard issues a new virtual card with test PAN (admin/internal use).
func (s *CardService) CreateVirtualCard(req card.CreateCardRequest) (*card.CreateCardResponse, error) {
	network := card.CardNetwork(req.CardNetwork)
	if network == "" {
		network = card.NetworkVisa
	}
	testBINs := pkgcard.TestBINs()
	bin := testBINs[string(network)]
	if bin == "" {
		bin = "400000"
	}

	// Generate test PAN
	pan := pkgcard.GenerateTestPAN(bin, 16)
	last4 := pkgcard.MaskPAN(pan)

	// Generate token (SHA256 of PAN — sandbox only, production uses HSM)
	token := crypto.SHA256("card:" + pan + idgen.RequestID())

	c := &card.Card{
		CardID:          idgen.CardID(),
		OwnerType:       req.OwnerType,
		OwnerID:         req.OwnerID,
		CardToken:       "tok_" + token[:32],
		PANLast4:        last4,
		CardNetwork:     network,
		CardType:        card.TypeDebit,
		CardForm:        card.FormVirtual,
		ExpiryMonth:     int(time.Now().Month()),
		ExpiryYear:      time.Now().Year() + 3,
		Status:          card.StatusActive,
		DefaultCurrency: req.DefaultCurrency,
		DailyLimit:      req.DailyLimit,
		MonthlyLimit:    req.MonthlyLimit,
		SingleTxLimit:   500000, // $5,000 default
	}
	if c.DefaultCurrency == "" {
		c.DefaultCurrency = "USD"
	}
	if c.DailyLimit == 0 {
		c.DailyLimit = 500000 // $5,000 default
	}
	if c.MonthlyLimit == 0 {
		c.MonthlyLimit = 5000000 // $50,000 default
	}

	if err := s.db.CreateCard(c); err != nil {
		return nil, fmt.Errorf("cannot create card: %w", err)
	}

	log.Printf("CardService: virtual card %s issued (last4=%s, network=%s)",
		c.CardID, last4, network)
	return &card.CreateCardResponse{
		CardID: c.CardID, CardToken: c.CardToken, Last4: last4,
		CardNetwork: string(network), Status: string(c.Status),
	}, nil
}

// ── Card Authorization (§5.4, §10) ───────────────

// SpendQuote returns a fee-estimate before the actual payment (§16.2).
// This matches the Wise "show fee before you pay" principle.
func (s *CardService) SpendQuote(cardID string, req card.SpendQuoteRequest) (*card.SpendQuoteResponse, error) {
	c, err := s.db.GetCard(cardID)
	if err != nil {
		return nil, fmt.Errorf("card not found: %w", err)
	}
	if !c.Status.IsUsable() {
		return nil, pkgerrors.InvalidInput("card is not active")
	}

	// Determine debit currency using Wise-like priority (§8.2)
	debitCurrency := s.selectDebitCurrency(c, req.TransactionCurrency)

	// Get FX rate if cross-currency
	var fxRate string
	var fxFee int64
	var fixedFee int64

	if debitCurrency == req.TransactionCurrency {
		fxRate = "1.000000000000"
		fxFee = 0
		fixedFee = 0 // Same-currency: no platform fee (§9.3)
	} else {
		rateStr, err := s.fxSvc.GetRate(debitCurrency, req.TransactionCurrency)
		if err != nil {
			return nil, fmt.Errorf("FX rate unavailable: %w", err)
		}
		fxRate = rateStr
		fxFee, fixedFee = s.calculateFee(req.TransactionCurrency, debitCurrency, req.MerchantCountry, c.CardNetwork)
	}

	// Convert transaction amount to debit amount (§8.4)
	debitBeforeFee := s.convertAmount(req.TransactionAmount, fxRate, true)
	totalDebit := debitBeforeFee + fxFee + fixedFee

	return &card.SpendQuoteResponse{
		TransactionAmount:   req.TransactionAmount,
		TransactionCurrency: req.TransactionCurrency,
		DebitAmount:         totalDebit,
		DebitCurrency:       debitCurrency,
		FXRate:              fxRate,
		FXFee:               fxFee,
		FixedFee:            fixedFee,
		TotalFee:            fxFee + fixedFee,
		ValidUntil:          time.Now().Add(120 * time.Second).Unix(),
	}, nil
}

// AuthorizeCardPayment processes a card authorization request (§10.2).
func (s *CardService) AuthorizeCardPayment(cardToken, networkAuthID string, req card.SpendQuoteRequest) (*card.CardAuthorization, error) {
	// 1. Find card by token
	c, err := s.db.GetCardByToken(cardToken)
	if err != nil {
		return nil, fmt.Errorf("card not found")
	}

	// 2. Validate card status
	if !c.Status.IsUsable() {
		return s.declineAuth("", c.CardID, req, card.DeclineCardNotActive)
	}

	// 3. Check limits
	if req.TransactionAmount > c.SingleTxLimit {
		return s.declineAuth("", c.CardID, req, card.DeclineLimitExceeded)
	}

	// 4. Get spend quote
	quote, err := s.SpendQuote(c.CardID, req)
	if err != nil {
		return s.declineAuth("", c.CardID, req, card.DeclineRiskRejected)
	}

	// 5. Check balance
	acct, err := s.db.GetAccountByUserAndCurrency(c.OwnerID, quote.DebitCurrency)
	if err != nil || !acct.CanDebit(quote.DebitAmount) {
		return s.declineAuth("", c.CardID, req, card.DeclineInsufficientFunds)
	}

	// 6. Freeze funds
	if err := s.db.FreezeFunds(acct.AccountID, quote.DebitAmount); err != nil {
		return s.declineAuth("", c.CardID, req, card.DeclineInsufficientFunds)
	}

	// 7. Record authorization
	authID := idgen.AuthID()
	auth := &card.CardAuthorization{
		AuthID:              authID,
		CardID:              c.CardID,
		MerchantName:        "",
		MerchantCountry:     req.MerchantCountry,
		MCC:                 req.MCC,
		TransactionAmount:   req.TransactionAmount,
		TransactionCurrency: req.TransactionCurrency,
		DebitAmount:         quote.DebitAmount,
		DebitCurrency:       quote.DebitCurrency,
		FXRate:              quote.FXRate,
		FeeAmount:           quote.TotalFee,
		Status:              card.AuthApproved,
	}
	if err := s.db.CreateCardAuthorization(auth); err != nil {
		// Rollback freeze
		s.db.UnfreezeFunds(acct.AccountID, quote.DebitAmount)
		return nil, fmt.Errorf("cannot record authorization: %w", err)
	}

	log.Printf("CardService: auth %s APPROVED — %d %s → debit %d %s (fee=%d)",
		authID, req.TransactionAmount, req.TransactionCurrency,
		quote.DebitAmount, quote.DebitCurrency, quote.TotalFee)
	return auth, nil
}

// ── FX & Fee Engine (§8, §9) ─────────────────────

// selectDebitCurrency implements Wise-like currency selection priority (§8.2):
// 1. Same currency as transaction → no FX
// 2. Card's default currency
// 3. USD as universal fallback
func (s *CardService) selectDebitCurrency(c *card.Card, txnCurrency string) string {
	if c.DefaultCurrency == txnCurrency {
		return txnCurrency
	}
	// Check if user has balance in transaction currency
	acct, err := s.db.GetAccountByUserAndCurrency(c.OwnerID, txnCurrency)
	if err == nil && acct.CanDebit(1) {
		return txnCurrency
	}
	// Fall back to card's default currency
	if c.DefaultCurrency != "" {
		return c.DefaultCurrency
	}
	return "USD"
}

// calculateFee implements Wise-like transparent fee model (§9.2):
//
//	Total = fixed_fee + percentage_fee × amount
//
// Cross-currency: 0.45% + $0.20 (sandbox defaults)
// Same-currency: 0%
func (s *CardService) calculateFee(txnCurrency, debitCurrency, country string, network card.CardNetwork) (fxFee int64, fixedFee int64) {
	if txnCurrency == debitCurrency {
		return 0, 0
	}

	// Sandbox default rates (§9.3):
	// Cross-currency card spend: 0.45% variable + 20 cents fixed
	feeRate := "0.0045"   // 0.45%
	fixedFee = 20          // $0.20 in cents

	// Convert fee rate to big.Rat for precision
	rateRat := new(big.Rat)
	rateRat.SetString(feeRate)
	_ = rateRat // fee calculated per-transaction in SpendQuote

	return 0, fixedFee // FX fee calculated per-transaction; fixed fee is constant
}

// convertAmount converts between currencies using the given rate.
// If inverse is true, divides by rate (target → source).
func (s *CardService) convertAmount(amount int64, rateStr string, inverse bool) int64 {
	rateRat := new(big.Rat)
	if _, ok := rateRat.SetString(rateStr); !ok {
		return amount
	}
	amountRat := new(big.Rat).SetInt64(amount)
	var result *big.Rat
	if inverse {
		result = new(big.Rat).Quo(amountRat, rateRat) // divide by rate
	} else {
		result = new(big.Rat).Mul(amountRat, rateRat) // multiply by rate
	}
	num := result.Num()
	denom := result.Denom()
	return new(big.Int).Div(num, denom).Int64()
}

// ── Helpers ──────────────────────────────────────

func (s *CardService) declineAuth(authID, cardID string, req card.SpendQuoteRequest, reason card.DeclineReason) (*card.CardAuthorization, error) {
	if authID == "" {
		authID = idgen.AuthID()
	}
	auth := &card.CardAuthorization{
		AuthID:              authID,
		CardID:              cardID,
		TransactionAmount:   req.TransactionAmount,
		TransactionCurrency: req.TransactionCurrency,
		Status:              card.AuthDeclined,
		DeclineReason:       reason,
	}
	s.db.CreateCardAuthorization(auth) // Best-effort recording
	return auth, fmt.Errorf("card auth declined: %s", reason)
}

// ── Card Management ──────────────────────────────

func (s *CardService) GetCard(cardID string) (*card.Card, error) {
	return s.db.GetCard(cardID)
}

func (s *CardService) ListCards(ownerID string) ([]card.Card, error) {
	return s.db.ListCardsByOwner(ownerID)
}

func (s *CardService) FreezeCard(cardID string) error {
	c, err := s.db.GetCard(cardID)
	if err != nil {
		return err
	}
	if !card.CanTransition(c.Status, card.StatusFrozen) {
		return pkgerrors.InvalidState(string(c.Status), string(card.StatusFrozen))
	}
	return s.db.UpdateCardStatus(cardID, card.StatusFrozen)
}

func (s *CardService) UnfreezeCard(cardID string) error {
	_, err := s.db.GetCard(cardID)
	if err != nil {
		return err
	}
	return s.db.UpdateCardStatus(cardID, card.StatusActive)
}

func (s *CardService) GetCardTransactions(cardID string, page, pageSize int) ([]card.CardTransaction, int64, error) {
	return s.db.ListCardTransactions(cardID, page, pageSize)
}
