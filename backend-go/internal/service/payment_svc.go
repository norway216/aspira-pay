package service

import (
	"database/sql"
	"encoding/json"
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
// V3.0: CREATED → QUOTE_LOCKED → COMPLIANCE_PRECHECKED → PAYMENT_PENDING
// → PAYMENT_EXECUTING → PAYMENT_CONFIRMED → SETTLEMENT_PROOFED → RECONCILED → CLOSED
type PaymentService struct {
	db            *repository.DB
	kycSvc        *KYCService
	riskSvc       *RiskService
	notifSvc      *NotificationService // V3: payment status notifications
	webhookSvc    *WebhookService      // V3: merchant webhook delivery
	fxSvc         *FXService
	settlementSvc *SettlementService
	chainSvc      *ChainService
	connector     PaymentConnector // §5.8: payment channel connector
}

func NewPaymentService(
	db *repository.DB, kycSvc *KYCService, riskSvc *RiskService,
	fxSvc *FXService, settlementSvc *SettlementService, chainSvc *ChainService,
	notifSvc *NotificationService, webhookSvc *WebhookService,
) *PaymentService {
	return &PaymentService{
		db:            db, kycSvc: kycSvc, riskSvc: riskSvc,
		fxSvc:         fxSvc, settlementSvc: settlementSvc, chainSvc: chainSvc,
		notifSvc:      notifSvc, webhookSvc: webhookSvc,
		connector:     NewSimulatedChannel("sandbox", 1.0),
	}
}

// CreatePayment initiates the full payment flow with idempotency and transactional outbox.
func (s *PaymentService) CreatePayment(req payment.CreateRequest, idempotencyKey, requestHash string) (*payment.CreateResponse, error) {
	if requestHash != "" {
		cachedResp, err := s.checkIdempotency(idempotencyKey, requestHash)
		if err != nil {
			return nil, err
		}
		if cachedResp != nil {
			return cachedResp, nil
		}
	}

	// Step 1: KYC Check + pre-fetch for Risk
	sender, err := s.db.GetUserByID(req.SenderUserID)
	if err != nil {
		return nil, pkgerrors.NotFound("sender not found")
	}
	if !sender.CanTransact() {
		return nil, pkgerrors.New(pkgerrors.ErrCodeKYCPending, "sender cannot transact")
	}

	kycProfile, err := s.kycSvc.GetKYCStatus(req.SenderUserID)
	if err != nil || !kycProfile.IsApproved() {
		return nil, pkgerrors.New(pkgerrors.ErrCodeKYCPending, "sender KYC not approved")
	}

	// Pre-fetch receiver for Risk (avoids duplicate DB lookup)
	receiver, _ := s.db.GetUserByID(req.ReceiverUserID)

	// Step 2: Risk Assessment (with pre-fetched data, avoids 5+ redundant DB queries)
	riskResult, err := s.riskSvc.AssessPaymentWithContext(req, sender, kycProfile, receiver)
	if err != nil {
		return nil, fmt.Errorf("risk assessment failed: %w", err)
	}
	if riskResult.Decision == "REJECT" {
		return nil, pkgerrors.New(pkgerrors.ErrCodeRiskRejected,
			fmt.Sprintf("transaction rejected by risk engine: %v", riskResult.Reasons))
	}
	if riskResult.Decision == "MANUAL_REVIEW" {
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

	// Step 4: Balance Check
	senderAccount, err := s.db.GetAccountByUserAndCurrency(req.SenderUserID, req.SourceCurrency)
	if err != nil {
		return nil, pkgerrors.InsufficientFunds(0, req.SourceAmount+quote.FeeAmount)
	}
	totalRequired := req.SourceAmount + quote.FeeAmount
	if !senderAccount.CanDebit(totalRequired) {
		return nil, pkgerrors.InsufficientFunds(senderAccount.AvailableBalance, totalRequired)
	}

	// Step 5: Transactional Outbox
	paymentID := idgen.PaymentID()
	order := &payment.Order{
		PaymentID:      paymentID,
		OrderID:        paymentID, // V3: order_id = payment_id for simple cases
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

	tx, err := s.db.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.db.CreatePaymentOrderTx(tx, order); err != nil {
		return nil, fmt.Errorf("cannot create payment order: %w", err)
	}

	outboxPayload := map[string]interface{}{"payment_id": paymentID, "status": string(payment.PaymentCreated)}
	if err := s.db.InsertOutboxEventTx(tx, idgen.EventID(), paymentID, "payment.created", outboxPayload); err != nil {
		return nil, fmt.Errorf("cannot insert outbox event: %w", err)
	}

	respJSON, _ := json.Marshal(&payment.CreateResponse{
		PaymentID: paymentID, Status: payment.PaymentCreated,
		SourceAmount: req.SourceAmount, TargetAmount: quote.TargetAmount,
		FeeAmount: quote.FeeAmount, FXRate: quote.Rate, QuoteID: quote.QuoteID,
		CreatedAt: time.Now().Unix(),
	})
	if err := s.db.InsertIdempotencyTx(tx, idempotencyKey, requestHash, string(respJSON)); err != nil {
		log.Printf("Warning: idempotency record insert failed (non-fatal): %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("cannot commit transaction: %w", err)
	}

	log.Printf("Payment %s created with outbox event", paymentID)

	// Step 6: V3.0 Saga (§5.3.2)
	go s.runSaga(paymentID)

	return &payment.CreateResponse{
		PaymentID: paymentID, Status: payment.PaymentCreated,
		SourceAmount: order.SourceAmount, TargetAmount: order.TargetAmount,
		FeeAmount: order.FeeAmount, FXRate: order.FXRate, QuoteID: order.QuoteID,
		CreatedAt: order.CreatedAt.Unix(),
	}, nil
}

// checkIdempotency enforces idempotency per §9.1.
func (s *PaymentService) checkIdempotency(key, requestHash string) (*payment.CreateResponse, error) {
	record, err := s.db.GetIdempotencyRecord(key)
	if err != nil {
		return nil, nil
	}
	if record == nil {
		return nil, nil
	}
	if record.RequestHash == requestHash {
		var resp payment.CreateResponse
		if err := json.Unmarshal([]byte(record.ResponseBody), &resp); err != nil {
			return nil, nil
		}
		log.Printf("Idempotency: returning cached result for key=%s", key)
		return &resp, nil
	}
	return nil, pkgerrors.New(pkgerrors.ErrCodeIdempotencyMismatch,
		"idempotency key reused with different request body")
}

// ── V3.0 Saga (§5.3.2) ──────────────────────────────

type sagaStep struct {
	name         string
	targetStatus payment.PaymentStatus
	execute      func(order *payment.Order) error
	compensate   func(order *payment.Order) error
}

func (s *PaymentService) runSaga(paymentID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Saga %s PANIC: %v", paymentID, r)
		}
	}()

	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		log.Printf("Saga %s: cannot load payment: %v", paymentID, err)
		return
	}

	// V3.0 flow: CREATED → QUOTE_LOCKED → COMPLIANCE_PRECHECKED → PAYMENT_PENDING
	// → PAYMENT_EXECUTING → PAYMENT_CONFIRMED → SETTLEMENT_PROOFED → RECONCILED → CLOSED
	steps := []sagaStep{
		{name: "QUOTE_LOCK", targetStatus: payment.PaymentQuoteLocked,
			execute: func(o *payment.Order) error { return nil }, compensate: nil},
		{name: "COMPLIANCE", targetStatus: payment.PaymentCompliancePrechecked,
			execute: func(o *payment.Order) error { return nil }, compensate: nil},
		{name: "FUNDS_FREEZE", targetStatus: payment.PaymentPending,
			execute: s.executeFreeze, compensate: s.executeRelease},
		{name: "PAYMENT_EXECUTING", targetStatus: payment.PaymentExecuting,
			execute: s.executeEngineOpStep, compensate: nil},
		{name: "PAYMENT_CONFIRMED", targetStatus: payment.PaymentConfirmed,
			execute: func(o *payment.Order) error { return nil }, compensate: s.executeRefund},
		{name: "SETTLEMENT", targetStatus: payment.PaymentSettlementProofed,
			execute: func(o *payment.Order) error {
				return s.settlementSvc.SettlePayment(o.PaymentID)
			}, compensate: s.compensateSettlement},
		{name: "RECONCILIATION", targetStatus: payment.PaymentReconciled,
			execute: func(o *payment.Order) error { return nil }, compensate: nil},
		{name: "CHAIN_AND_CLOSE", targetStatus: payment.PaymentClosed,
			execute: func(o *payment.Order) error {
				s.chainSvc.RecordPaymentOnChain(o.PaymentID)
				return nil
			}, compensate: nil},
	}

	completedSteps := []int{}
	for i, step := range steps {
		order, err = s.db.GetPaymentOrder(paymentID)
		if err != nil {
			log.Printf("Saga %s: cannot reload at step %s: %v", paymentID, step.name, err)
			s.compensateInReverse(paymentID, completedSteps, steps)
			return
		}

		// Skip already-completed steps
		if s.isStatusAtOrAfter(order.Status, step.targetStatus) {
			completedSteps = append(completedSteps, i)
			continue
		}

		if !payment.CanTransition(order.Status, step.targetStatus) {
			log.Printf("Saga %s: invalid transition %s → %s", paymentID, order.Status, step.targetStatus)
			s.compensateInReverse(paymentID, completedSteps, steps)
			return
		}

		if err := step.execute(order); err != nil {
			log.Printf("Saga %s: step %s FAILED: %v", paymentID, step.name, err)
			s.db.UpdatePaymentStatus(paymentID, payment.PaymentFailed)
			s.compensateInReverse(paymentID, completedSteps, steps)
			return
		}

		s.db.UpdatePaymentStatus(paymentID, step.targetStatus)
		completedSteps = append(completedSteps, i)

		// V3: Notify on key state transitions
		s.notifyIfNeeded(order, step.targetStatus)
	}
	log.Printf("Saga %s: completed ✓", paymentID)

	// V3: Final webhook notification
	s.dispatchWebhook(paymentID, "payment.completed")
}

// notifyIfNeeded sends user notification on key state changes.
func (s *PaymentService) notifyIfNeeded(order *payment.Order, status payment.PaymentStatus) {
	switch status {
	case payment.PaymentConfirmed, payment.PaymentFailed, payment.PaymentClosed:
		if s.notifSvc != nil {
			s.notifSvc.NotifyPaymentStatus(order.SenderUserID, order.PaymentID, string(status))
		}
	}
}

// dispatchWebhook sends webhook to registered merchant endpoints.
func (s *PaymentService) dispatchWebhook(paymentID, eventType string) {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil { return }
	if s.webhookSvc != nil {
		s.webhookSvc.DeliverEvent(eventType, paymentID, map[string]interface{}{
			"payment_id":      order.PaymentID,
			"status":          string(order.Status),
			"source_currency": order.SourceCurrency,
			"target_currency": order.TargetCurrency,
			"source_amount":   order.SourceAmount,
			"target_amount":   order.TargetAmount,
			"fee_amount":      order.FeeAmount,
		})
	}
}

func (s *PaymentService) compensateInReverse(paymentID string, completedIndices []int, steps []sagaStep) {
	for idx := len(completedIndices) - 1; idx >= 0; idx-- {
		step := steps[completedIndices[idx]]
		if step.compensate == nil {
			continue
		}
		order, err := s.db.GetPaymentOrder(paymentID)
		if err != nil {
			continue
		}
		log.Printf("Saga %s: compensating step %s", paymentID, step.name)
		if err := step.compensate(order); err != nil {
			log.Printf("Saga %s: compensation %s FAILED: %v", paymentID, step.name, err)
		}
	}
}

func (s *PaymentService) isStatusAtOrAfter(current, target payment.PaymentStatus) bool {
	order := []payment.PaymentStatus{
		payment.PaymentCreated, payment.PaymentQuoteLocked,
		payment.PaymentCompliancePrechecked, payment.PaymentPending,
		payment.PaymentExecuting, payment.PaymentConfirmed,
		payment.PaymentSettlementProofed, payment.PaymentReconciled,
		payment.PaymentClosed,
	}
	ci, ti := -1, -1
	for i, st := range order {
		if st == current { ci = i }
		if st == target { ti = i }
	}
	return ci >= ti
}

// ── Step implementations ──────────────────────────

func (s *PaymentService) executeFreeze(order *payment.Order) error {
	totalRequired := order.SourceAmount + order.FeeAmount
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return fmt.Errorf("freeze: %w", err)
	}
	if !senderAccount.CanDebit(totalRequired) {
		return pkgerrors.InsufficientFunds(senderAccount.AvailableBalance, totalRequired)
	}
	return s.db.FreezeFunds(senderAccount.AccountID, totalRequired)
}

// executeEngineOpStep is the Saga step function that calls executeEngineOp.
func (s *PaymentService) executeEngineOpStep(order *payment.Order) error {
	return s.executeEngineOp(order.PaymentID)
}

func (s *PaymentService) executeEngineOp(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return fmt.Errorf("engine: %w", err)
	}
	totalRequired := order.SourceAmount + order.FeeAmount

	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return fmt.Errorf("engine: sender: %w", err)
	}
	if err := s.db.DebitAccount(senderAccount.AccountID, totalRequired); err != nil {
		return fmt.Errorf("debit: %w", err)
	}

	receiverAccount, err := s.db.GetAccountByUserAndCurrency(order.ReceiverUserID, order.TargetCurrency)
	if err != nil {
		receiverAccount = &ledger.LegacyAccount{
			AccountID: idgen.AccountID(), UserID: order.ReceiverUserID, Currency: order.TargetCurrency,
		}
		if err := s.db.CreateAccount(receiverAccount); err != nil {
			return fmt.Errorf("create receiver: %w", err)
		}
	}
	if err := s.db.CreditAccount(receiverAccount.AccountID, order.TargetAmount); err != nil {
		return fmt.Errorf("credit: %w", err)
	}

	feeAccountID := "sys_fee_income_" + strings.ToLower(order.SourceCurrency)
	if err := s.db.CreditAccount(feeAccountID, order.FeeAmount); err != nil {
		return fmt.Errorf("fee: %w", err)
	}
	return nil
}

func (s *PaymentService) executeRelease(order *payment.Order) error {
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return err
	}
	return s.db.UnfreezeFunds(senderAccount.AccountID, order.SourceAmount+order.FeeAmount)
}

func (s *PaymentService) executeRefund(order *payment.Order) error {
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return err
	}
	s.db.AddAvailableBalance(senderAccount.AccountID, order.SourceAmount+order.FeeAmount)
	receiverAccount, _ := s.db.GetAccountByUserAndCurrency(order.ReceiverUserID, order.TargetCurrency)
	if receiverAccount != nil {
		s.db.DebitAccount(receiverAccount.AccountID, order.TargetAmount)
	}
	feeAccountID := "sys_fee_income_" + strings.ToLower(order.SourceCurrency)
	s.db.DebitAccount(feeAccountID, order.FeeAmount)
	return nil
}

func (s *PaymentService) compensateSettlement(order *payment.Order) error {
	entries, err := s.db.GetLedgerEntriesByPayment(order.PaymentID)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no ledger entries to reverse")
	}
	eventID := idgen.EventID()
	var reversals []ledger.Entry
	for i, e := range entries {
		revDir := ledger.DirectionCredit
		if e.Direction == ledger.DirectionCredit {
			revDir = ledger.DirectionDebit
		}
		reversals = append(reversals, ledger.Entry{
			EntryID: idgen.EntryID(order.PaymentID, i+100), EventID: eventID,
			PaymentID: order.PaymentID, AccountID: e.AccountID,
			Currency: e.Currency, Direction: revDir, Amount: e.Amount,
			Description: fmt.Sprintf("REVERSAL: %s", e.Description),
		})
	}
	return s.db.InsertLedgerEntries(reversals)
}

// ── Query methods ─────────────────────────────────

func (s *PaymentService) GetPayment(paymentID string) (*payment.Order, error) {
	return s.db.GetPaymentOrder(paymentID)
}
func (s *PaymentService) ListPayments(q payment.ListQuery) ([]payment.Order, int64, error) {
	return s.db.ListPaymentOrders(q)
}
func (s *PaymentService) GetPaymentByRequestID(requestID string) (*payment.Order, error) {
	return s.db.GetPaymentByRequestID(requestID)
}
func (s *PaymentService) RefundPayment(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return err
	}
	if !payment.CanTransition(order.Status, payment.PaymentRefundPending) {
		return pkgerrors.InvalidState(string(order.Status), string(payment.PaymentClosed))
	}
	s.db.UpdatePaymentStatus(paymentID, payment.PaymentRefundPending)
	senderAccount, _ := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	receiverAccount, _ := s.db.GetAccountByUserAndCurrency(order.ReceiverUserID, order.TargetCurrency)
	feeAccountID := "sys_fee_income_" + strings.ToLower(order.SourceCurrency)
	if senderAccount != nil {
		s.db.AddAvailableBalance(senderAccount.AccountID, order.SourceAmount+order.FeeAmount)
	}
	if receiverAccount != nil {
		s.db.DebitAccount(receiverAccount.AccountID, order.TargetAmount)
	}
	s.db.DebitAccount(feeAccountID, order.FeeAmount)
	return s.db.UpdatePaymentStatus(paymentID, payment.PaymentRefunded)
}

// OutboxWorker is defined in a separate file (outbox_worker.go).
// See payment_connector.go for PaymentConnector interface.

var _ = (*sql.Tx)(nil)
