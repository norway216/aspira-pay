package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/fx"
	"github.com/aspira/aspira-pay/internal/domain/ledger"
	"github.com/aspira/aspira-pay/internal/domain/payment"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/idgen"
	pkgerrors "github.com/aspira/aspira-pay/pkg/errors"
)

// PaymentService is the core business orchestrator for payment processing.
// It coordinates the full payment lifecycle: KYC → Risk → FX → Engine → Settlement → Chain.
type PaymentService struct {
	db      *repository.DB
	kycSvc  *KYCService
	riskSvc *RiskService
	fxSvc   *FXService
	settlementSvc *SettlementService
	chainSvc *ChainService
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(
	db *repository.DB,
	kycSvc *KYCService,
	riskSvc *RiskService,
	fxSvc *FXService,
	settlementSvc *SettlementService,
	chainSvc *ChainService,
) *PaymentService {
	return &PaymentService{
		db:            db,
		kycSvc:        kycSvc,
		riskSvc:       riskSvc,
		fxSvc:         fxSvc,
		settlementSvc: settlementSvc,
		chainSvc:      chainSvc,
	}
}

// CreatePayment initiates the full payment flow.
// This is the main entry point for the cross-border payment process.
func (s *PaymentService) CreatePayment(req payment.CreateRequest) (*payment.CreateResponse, error) {
	// Step 1: KYC Check
	sender, err := s.db.GetUserByID(req.SenderUserID)
	if err != nil {
		return nil, pkgerrors.NotFound("sender not found")
	}
	if !sender.CanTransact() {
		return nil, pkgerrors.New(pkgerrors.ErrCodeKYCPending, "sender cannot transact — KYC not completed or account frozen")
	}

	kycProfile, err := s.kycSvc.GetKYCStatus(req.SenderUserID)
	if err != nil || !kycProfile.IsApproved() {
		return nil, pkgerrors.New(pkgerrors.ErrCodeKYCPending, "sender KYC not approved")
	}

	// Step 2: Risk Assessment
	riskResult, err := s.riskSvc.AssessPayment(req)
	if err != nil {
		return nil, fmt.Errorf("risk assessment failed: %w", err)
	}

	if riskResult.Decision == "REJECT" {
		return nil, pkgerrors.New(pkgerrors.ErrCodeRiskRejected,
			fmt.Sprintf("transaction rejected by risk engine: %v", riskResult.Reasons))
	}

	if riskResult.Decision == "MANUAL_REVIEW" {
		// In Sandbox, auto-pass manual review for testing; in production, queue for review
		log.Printf("Payment flagged for manual review: score=%d reasons=%v", riskResult.Score, riskResult.Reasons)
	}

	// Step 3: FX Quote
	quote, err := s.fxSvc.GetQuote(fx.QuoteRequest{
		SourceCurrency: req.SourceCurrency,
		TargetCurrency: req.TargetCurrency,
		SourceAmount:   req.SourceAmount,
	})
	if err != nil {
		return nil, fmt.Errorf("FX quote failed: %w", err)
	}

	// Step 4: Check sender account balance
	senderAccount, err := s.db.GetAccountByUserAndCurrency(req.SenderUserID, req.SourceCurrency)
	if err != nil {
		return nil, pkgerrors.InsufficientFunds(0, req.SourceAmount+quote.FeeAmount)
	}

	totalRequired := req.SourceAmount + quote.FeeAmount
	if !senderAccount.CanDebit(totalRequired) {
		return nil, pkgerrors.InsufficientFunds(senderAccount.AvailableBalance, totalRequired)
	}

	// Step 5: Create payment order
	paymentID := idgen.PaymentID()
	order := &payment.Order{
		PaymentID:      paymentID,
		RequestID:      idgen.RequestID(),
		SenderUserID:   req.SenderUserID,
		ReceiverUserID: req.ReceiverUserID,
		SourceCurrency: req.SourceCurrency,
		TargetCurrency: req.TargetCurrency,
		SourceAmount:   req.SourceAmount,
		TargetAmount:   quote.TargetAmount,
		FeeAmount:      quote.FeeAmount,
		FXRate:         quote.Rate,
		Status:         payment.PaymentCreated,
		RiskScore:      riskResult.Score,
		QuoteID:        quote.QuoteID,
		Purpose:        req.Purpose,
		CountryFrom:    req.CountryFrom,
		CountryTo:      req.CountryTo,
	}

	if err := s.db.CreatePaymentOrder(order); err != nil {
		return nil, fmt.Errorf("cannot create payment order: %w", err)
	}

	// Step 6: Process payment through state machine
	go s.processPaymentAsync(order.PaymentID)

	return &payment.CreateResponse{
		PaymentID:    order.PaymentID,
		Status:       order.Status,
		SourceAmount: order.SourceAmount,
		TargetAmount: order.TargetAmount,
		FeeAmount:    order.FeeAmount,
		FXRate:       order.FXRate,
		QuoteID:      order.QuoteID,
		CreatedAt:    order.CreatedAt.Unix(),
	}, nil
}

// processPaymentAsync drives the payment through its state machine asynchronously.
func (s *PaymentService) processPaymentAsync(paymentID string) {
	states := []payment.PaymentStatus{
		payment.PaymentKYCChecked,
		payment.PaymentRiskChecked,
		payment.PaymentQuoteLocked,
		payment.PaymentFundsFreezeRequested,
		payment.PaymentSubmittedToEngine,
		payment.PaymentEngineExecuted,
		payment.PaymentSettlementPending,
		payment.PaymentSettled,
		payment.PaymentChainConfirmed,
		payment.PaymentCompleted,
	}

	for _, targetStatus := range states {
		time.Sleep(100 * time.Millisecond) // Simulate processing time

		if err := s.db.UpdatePaymentStatus(paymentID, targetStatus); err != nil {
			log.Printf("Payment %s: cannot transition to %s: %v", paymentID, targetStatus, err)
			// Try to transition to FAILED
			if targetStatus != payment.PaymentCreated {
				s.db.UpdatePaymentStatus(paymentID, payment.PaymentFailed)
			}
			return
		}

		// Perform side effects at key states
		switch targetStatus {
		case payment.PaymentFundsFreezeRequested:
			if err := s.executeFreeze(paymentID); err != nil {
				log.Printf("Payment %s: freeze failed: %v", paymentID, err)
				s.db.UpdatePaymentStatus(paymentID, payment.PaymentFailed)
				return
			}

		case payment.PaymentEngineExecuted:
			if err := s.executeEngineOp(paymentID); err != nil {
				log.Printf("Payment %s: engine execution failed: %v", paymentID, err)
				s.db.UpdatePaymentStatus(paymentID, payment.PaymentFailed)
				// Attempt refund of frozen funds
				s.executeRelease(paymentID)
				return
			}

		case payment.PaymentSettled:
			if err := s.settlementSvc.SettlePayment(paymentID); err != nil {
				log.Printf("Payment %s: settlement failed: %v", paymentID, err)
				s.db.UpdatePaymentStatus(paymentID, payment.PaymentFailed)
				return
			}

		case payment.PaymentChainConfirmed:
			if err := s.chainSvc.RecordPaymentOnChain(paymentID); err != nil {
				log.Printf("Payment %s: chain recording failed (non-fatal): %v", paymentID, err)
				// Non-fatal — continue to COMPLETED anyway
			}
		}
	}

	log.Printf("Payment %s completed successfully", paymentID)
}

// executeFreeze freezes funds in the sender's account.
func (s *PaymentService) executeFreeze(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return err
	}

	totalRequired := order.SourceAmount + order.FeeAmount
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return err
	}

	if !senderAccount.CanDebit(totalRequired) {
		return pkgerrors.InsufficientFunds(senderAccount.AvailableBalance, totalRequired)
	}

	return s.db.FreezeFunds(senderAccount.AccountID, totalRequired)
}

// executeEngineOp simulates the C++ engine execution.
func (s *PaymentService) executeEngineOp(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return err
	}

	// In Sandbox without C++ engine, perform direct account operations
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return err
	}

	totalRequired := order.SourceAmount + order.FeeAmount

	// Debit sender (frozen → settled)
	if err := s.db.DebitAccount(senderAccount.AccountID, totalRequired); err != nil {
		return fmt.Errorf("debit sender failed: %w", err)
	}

	// Credit receiver (target currency)
	receiverAccount, err := s.db.GetAccountByUserAndCurrency(order.ReceiverUserID, order.TargetCurrency)
	if err != nil {
		// Auto-create receiver account if not exists
		receiverAccount = &ledger.Account{
			AccountID: idgen.AccountID(),
			UserID:    order.ReceiverUserID,
			Currency:  order.TargetCurrency,
		}
		if err := s.db.CreateAccount(receiverAccount); err != nil {
			return fmt.Errorf("cannot create receiver account: %w", err)
		}
	}

	if err := s.db.CreditAccount(receiverAccount.AccountID, order.TargetAmount); err != nil {
		return fmt.Errorf("credit receiver failed: %w", err)
	}

	// Credit fee to platform
	feeAccountID := "sys_fee_income_" + strings.ToLower(order.SourceCurrency)
	if err := s.db.CreditAccount(feeAccountID, order.FeeAmount); err != nil {
		return fmt.Errorf("credit fee failed: %w", err)
	}

	return nil
}

// executeRelease releases frozen funds (compensation on failure).
func (s *PaymentService) executeRelease(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return err
	}

	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return err
	}

	totalRequired := order.SourceAmount + order.FeeAmount
	return s.db.UnfreezeFunds(senderAccount.AccountID, totalRequired)
}

// GetPayment retrieves a payment by ID.
func (s *PaymentService) GetPayment(paymentID string) (*payment.Order, error) {
	return s.db.GetPaymentOrder(paymentID)
}

// ListPayments returns a filtered list of payments.
func (s *PaymentService) ListPayments(q payment.ListQuery) ([]payment.Order, int64, error) {
	return s.db.ListPaymentOrders(q)
}

// GetPaymentByRequestID checks idempotency by request_id.
func (s *PaymentService) GetPaymentByRequestID(requestID string) (*payment.Order, error) {
	return s.db.GetPaymentByRequestID(requestID)
}

// RefundPayment processes a full refund.
func (s *PaymentService) RefundPayment(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return err
	}

	if !payment.CanTransition(order.Status, payment.PaymentRefunded) {
		if order.Status != payment.PaymentCompleted {
			return pkgerrors.InvalidState(string(order.Status), string(payment.PaymentCompleted))
		}
	}

	// Reverse the transaction
	senderAccount, _ := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	receiverAccount, _ := s.db.GetAccountByUserAndCurrency(order.ReceiverUserID, order.TargetCurrency)
	feeAccountID := "sys_fee_income_" + strings.ToLower(order.SourceCurrency)

	if senderAccount != nil {
		// Refund source amount + fee to sender
		s.db.AddAvailableBalance(senderAccount.AccountID, order.SourceAmount+order.FeeAmount)
	}
	if receiverAccount != nil {
		s.db.DebitAccount(receiverAccount.AccountID, order.TargetAmount) // Simplified
	}
	// Debit fee account
	s.db.DebitAccount(feeAccountID, order.FeeAmount)

	return s.db.UpdatePaymentStatus(paymentID, payment.PaymentRefunded)
}
