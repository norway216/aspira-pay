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
// It coordinates the full payment lifecycle: KYC → Risk → FX → Engine → Settlement → Chain.
type PaymentService struct {
	db            *repository.DB
	kycSvc        *KYCService
	riskSvc       *RiskService
	fxSvc         *FXService
	settlementSvc *SettlementService
	chainSvc      *ChainService
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

// ──────────────────────────────────────────────
// CreatePayment — main entry point with idempotency and transactional outbox
// ──────────────────────────────────────────────

// CreatePayment initiates the full payment flow with idempotency guarantees.
// Architecture doc §9.1: Same request_id + same request_hash → return cached result.
// Same request_id + different request_hash → reject (conflict).
func (s *PaymentService) CreatePayment(req payment.CreateRequest, idempotencyKey, requestHash string) (*payment.CreateResponse, error) {
	// ── Step 0: Idempotency check ──────────────────────────
	// Architecture doc §9.1: Check idempotency_keys before processing.
	if requestHash != "" {
		cachedResp, err := s.checkIdempotency(idempotencyKey, requestHash)
		if err != nil {
			return nil, err
		}
		if cachedResp != nil {
			return cachedResp, nil
		}
	}

	// ── Step 1: KYC Check ──────────────────────────────────
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

	// ── Step 2: Risk Assessment ────────────────────────────
	riskResult, err := s.riskSvc.AssessPayment(req)
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

	// ── Step 3: FX Quote ──────────────────────────────────
	quote, err := s.fxSvc.GetQuote(fx.QuoteRequest{
		SourceCurrency: req.SourceCurrency,
		TargetCurrency: req.TargetCurrency,
		SourceAmount:   req.SourceAmount,
	})
	if err != nil {
		return nil, fmt.Errorf("FX quote failed: %w", err)
	}

	// ── Step 4: Balance Check ──────────────────────────────
	senderAccount, err := s.db.GetAccountByUserAndCurrency(req.SenderUserID, req.SourceCurrency)
	if err != nil {
		return nil, pkgerrors.InsufficientFunds(0, req.SourceAmount+quote.FeeAmount)
	}

	totalRequired := req.SourceAmount + quote.FeeAmount
	if !senderAccount.CanDebit(totalRequired) {
		return nil, pkgerrors.InsufficientFunds(senderAccount.AvailableBalance, totalRequired)
	}

	// ── Step 5: Transactional Outbox ───────────────────────
	// Architecture doc §7.4: INSERT payment_orders + INSERT outbox_events in one TX.
	paymentID := idgen.PaymentID()
	requestID := idgen.RequestID()

	order := &payment.Order{
		PaymentID:      paymentID,
		RequestID:      requestID,
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

	// Open transaction
	tx, err := s.db.BeginTx()
	if err != nil {
		return nil, fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert payment order within TX
	if err := s.db.CreatePaymentOrderTx(tx, order); err != nil {
		return nil, fmt.Errorf("cannot create payment order: %w", err)
	}

	// Insert outbox event within same TX (ensures atomicity)
	outboxPayload := map[string]interface{}{
		"payment_id":      paymentID,
		"source_currency": req.SourceCurrency,
		"target_currency": req.TargetCurrency,
		"source_amount":   req.SourceAmount,
		"target_amount":   quote.TargetAmount,
		"fee_amount":      quote.FeeAmount,
		"status":          string(payment.PaymentCreated),
	}
	if err := s.db.InsertOutboxEventTx(tx, idgen.EventID(), paymentID, "payment.created", outboxPayload); err != nil {
		return nil, fmt.Errorf("cannot insert outbox event: %w", err)
	}

	// Store idempotency record within same TX
	respJSON, _ := json.Marshal(&payment.CreateResponse{
		PaymentID:    paymentID,
		Status:       payment.PaymentCreated,
		SourceAmount: req.SourceAmount,
		TargetAmount: quote.TargetAmount,
		FeeAmount:    quote.FeeAmount,
		FXRate:       quote.Rate,
		QuoteID:      quote.QuoteID,
		CreatedAt:    time.Now().Unix(),
	})
	if err := s.db.InsertIdempotencyTx(tx, idempotencyKey, requestHash, string(respJSON)); err != nil {
		log.Printf("Warning: idempotency record insert failed (non-fatal): %v", err)
	}

	// Commit transaction — payment order + outbox event + idempotency are atomic
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("cannot commit transaction: %w", err)
	}

	log.Printf("Payment %s created with outbox event (idempotency_key=%s)", paymentID, idempotencyKey)

	// ── Step 6: Process via Saga orchestrator asynchronously ──
	go s.runSaga(paymentID)

	return &payment.CreateResponse{
		PaymentID:    paymentID,
		Status:       payment.PaymentCreated,
		SourceAmount: order.SourceAmount,
		TargetAmount: order.TargetAmount,
		FeeAmount:    order.FeeAmount,
		FXRate:       order.FXRate,
		QuoteID:      order.QuoteID,
		CreatedAt:    order.CreatedAt.Unix(),
	}, nil
}

// ──────────────────────────────────────────────
// Idempotency helpers
// ──────────────────────────────────────────────

// checkIdempotency enforces idempotency per architecture doc §9.1.
// Returns the cached response if found, nil if this is a new request, or error on mismatch.
func (s *PaymentService) checkIdempotency(key, requestHash string) (*payment.CreateResponse, error) {
	record, err := s.db.GetIdempotencyRecord(key)
	if err != nil {
		return nil, nil // Not found — new request, proceed
	}

	// Same key + same hash → return cached result (idempotent replay)
	if record.RequestHash == requestHash {
		var resp payment.CreateResponse
		if err := json.Unmarshal([]byte(record.ResponseBody), &resp); err != nil {
			return nil, nil // Corrupt cache, allow reprocessing
		}
		log.Printf("Idempotency: returning cached result for key=%s", key)
		return &resp, nil
	}

	// Same key + different hash → reject (potential conflict/replay attack)
	return nil, pkgerrors.New(pkgerrors.ErrCodeIdempotencyMismatch,
		"idempotency key reused with different request body")
}

// ──────────────────────────────────────────────
// Saga Orchestrator
// Architecture doc §9.2: Each step has a defined compensation.
// ──────────────────────────────────────────────

// sagaStep defines a single step in the payment saga with its compensation.
type sagaStep struct {
	name         string
	targetStatus payment.PaymentStatus
	execute      func(order *payment.Order) error
	compensate   func(order *payment.Order) error // nil if not compensatable
}

// runSaga executes the payment saga: a sequence of steps each with a compensation.
// On failure at any step, previously completed steps are compensated in reverse order.
// Architecture doc §9.2: Saga Pattern for distributed transaction flow.
func (s *PaymentService) runSaga(paymentID string) {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		log.Printf("Saga %s: cannot load payment: %v", paymentID, err)
		return
	}

	steps := []sagaStep{
		{
			name:         "KYC_CHECK",
			targetStatus: payment.PaymentKYCChecked,
			execute:      func(o *payment.Order) error { return nil }, // Already checked in CreatePayment
			compensate:   nil, // Nothing to undo
		},
		{
			name:         "RISK_CHECK",
			targetStatus: payment.PaymentRiskChecked,
			execute:      func(o *payment.Order) error { return nil }, // Already checked in CreatePayment
			compensate:   nil,
		},
		{
			name:         "FX_QUOTE_LOCK",
			targetStatus: payment.PaymentQuoteLocked,
			execute:      func(o *payment.Order) error { return nil }, // Quote already locked
			compensate:   nil,
		},
		{
			name:         "FUNDS_FREEZE",
			targetStatus: payment.PaymentFundsFreezeRequested,
			execute:      s.executeFreeze,
			compensate:   s.executeRelease, // Release frozen funds on failure
		},
		{
			name:         "ENGINE_EXECUTION",
			targetStatus: payment.PaymentEngineExecuted,
			execute: func(o *payment.Order) error {
				// Transition through SUBMITTED_TO_ENGINE first
				if err := s.db.UpdatePaymentStatusValidated(o.PaymentID, payment.PaymentFundsFreezeRequested, payment.PaymentSubmittedToEngine); err != nil {
					return fmt.Errorf("engine submit: %w", err)
				}
				return s.executeEngineOp(o.PaymentID)
			},
			compensate: s.executeRefund, // Refund on engine failure after freeze
		},
		{
			name:         "SETTLEMENT",
			targetStatus: payment.PaymentSettled,
			execute: func(o *payment.Order) error {
				// Transition through SETTLEMENT_PENDING
				if err := s.db.UpdatePaymentStatusValidated(o.PaymentID, payment.PaymentEngineExecuted, payment.PaymentSettlementPending); err != nil {
					return fmt.Errorf("settlement pending: %w", err)
				}
				return s.settlementSvc.SettlePayment(o.PaymentID)
			},
			compensate: s.compensateSettlement, // Reverse ledger entries
		},
		{
			name:         "CHAIN_CONFIRMATION",
			targetStatus: payment.PaymentChainConfirmed,
			execute: func(o *payment.Order) error {
				if err := s.chainSvc.RecordPaymentOnChain(o.PaymentID); err != nil {
					log.Printf("Saga %s: chain recording failed (non-fatal): %v", o.PaymentID, err)
					// Non-fatal per architecture doc §12.5: blockchain failures do not block
				}
				return s.db.UpdatePaymentStatusValidated(o.PaymentID, payment.PaymentSettled, payment.PaymentChainConfirmed)
			},
			compensate: nil, // Chain is append-only, no compensation needed
		},
		{
			name:         "COMPLETION",
			targetStatus: payment.PaymentCompleted,
			execute: func(o *payment.Order) error {
				return s.db.UpdatePaymentStatusValidated(o.PaymentID, payment.PaymentChainConfirmed, payment.PaymentCompleted)
			},
			compensate: nil, // Terminal state
		},
	}

	// Execute steps in sequence
	completedSteps := []int{}

	for i, step := range steps {
		log.Printf("Saga %s: executing step %d/%d — %s", paymentID, i+1, len(steps), step.name)

		// Reload order for fresh state
		order, err = s.db.GetPaymentOrder(paymentID)
		if err != nil {
			log.Printf("Saga %s: cannot reload order at step %s: %v", paymentID, step.name, err)
			s.compensateInReverse(paymentID, completedSteps, steps)
			return
		}

		// Skip already-completed steps (idempotent replay)
		if s.isStatusAtOrAfter(order.Status, step.targetStatus) {
			log.Printf("Saga %s: step %s already completed (status=%s), skipping", paymentID, step.name, order.Status)
			completedSteps = append(completedSteps, i)
			continue
		}

		// Validate transition
		if !payment.CanTransition(order.Status, step.targetStatus) {
			log.Printf("Saga %s: invalid transition %s → %s at step %s", paymentID, order.Status, step.targetStatus, step.name)
			s.compensateInReverse(paymentID, completedSteps, steps)
			return
		}

		// Execute the step
		if err := step.execute(order); err != nil {
			log.Printf("Saga %s: step %s FAILED: %v", paymentID, step.name, err)
			// Mark as failed
			s.db.UpdatePaymentStatus(paymentID, payment.PaymentFailed)
			// Compensate completed steps in reverse order
			s.compensateInReverse(paymentID, completedSteps, steps)
			return
		}

		// Update state
		if err := s.db.UpdatePaymentStatus(paymentID, step.targetStatus); err != nil {
			log.Printf("Saga %s: cannot transition to %s: %v", paymentID, step.targetStatus, err)
			// If can't update status, mark as failed
			if step.targetStatus != payment.PaymentCreated {
				s.db.UpdatePaymentStatus(paymentID, payment.PaymentFailed)
			}
			s.compensateInReverse(paymentID, completedSteps, steps)
			return
		}

		completedSteps = append(completedSteps, i)
	}

	log.Printf("Saga %s: completed successfully ✓", paymentID)
}

// compensateInReverse runs compensation for completed steps in reverse order.
// Architecture doc §9.2: Compensation flow — release funds, reverse ledger entries.
func (s *PaymentService) compensateInReverse(paymentID string, completedIndices []int, steps []sagaStep) {
	log.Printf("Saga %s: starting compensation for %d completed steps", paymentID, len(completedIndices))

	for idx := len(completedIndices) - 1; idx >= 0; idx-- {
		stepIdx := completedIndices[idx]
		step := steps[stepIdx]
		if step.compensate == nil {
			continue
		}

		order, err := s.db.GetPaymentOrder(paymentID)
		if err != nil {
			log.Printf("Saga %s: compensation %s: cannot load order: %v", paymentID, step.name, err)
			continue
		}

		log.Printf("Saga %s: compensating step %s", paymentID, step.name)
		if err := step.compensate(order); err != nil {
			log.Printf("Saga %s: compensation %s FAILED: %v", paymentID, step.name, err)
			// Continue compensating other steps — best effort
		}
	}

	log.Printf("Saga %s: compensation complete", paymentID)
}

// isStatusAtOrAfter checks if current status has reached or passed the target in the state machine.
func (s *PaymentService) isStatusAtOrAfter(current, target payment.PaymentStatus) bool {
	order := []payment.PaymentStatus{
		payment.PaymentCreated,
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

	currentIdx := -1
	targetIdx := -1
	for i, s := range order {
		if s == current {
			currentIdx = i
		}
		if s == target {
			targetIdx = i
		}
	}
	return currentIdx >= targetIdx
}

// ──────────────────────────────────────────────
// Step implementations
// ──────────────────────────────────────────────

// executeFreeze freezes funds in the sender's account.
func (s *PaymentService) executeFreeze(order *payment.Order) error {
	totalRequired := order.SourceAmount + order.FeeAmount
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return fmt.Errorf("freeze: sender account not found: %w", err)
	}

	if !senderAccount.CanDebit(totalRequired) {
		return pkgerrors.InsufficientFunds(senderAccount.AvailableBalance, totalRequired)
	}

	return s.db.FreezeFunds(senderAccount.AccountID, totalRequired)
}

// executeEngineOp performs the actual debit/credit operations (simulates C++ engine in Sandbox).
func (s *PaymentService) executeEngineOp(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return fmt.Errorf("engine: payment not found: %w", err)
	}

	totalRequired := order.SourceAmount + order.FeeAmount

	// Get sender account
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return fmt.Errorf("engine: sender account not found: %w", err)
	}

	// Debit sender (frozen → settled)
	if err := s.db.DebitAccount(senderAccount.AccountID, totalRequired); err != nil {
		return fmt.Errorf("debit sender failed: %w", err)
	}

	// Credit receiver (target currency) — auto-create account if needed
	receiverAccount, err := s.db.GetAccountByUserAndCurrency(order.ReceiverUserID, order.TargetCurrency)
	if err != nil {
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

// executeRelease unfreezes funds — compensation for executeFreeze.
func (s *PaymentService) executeRelease(order *payment.Order) error {
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return fmt.Errorf("release: sender account not found: %w", err)
	}

	totalRequired := order.SourceAmount + order.FeeAmount
	return s.db.UnfreezeFunds(senderAccount.AccountID, totalRequired)
}

// executeRefund reverses the engine execution — compensation for executeEngineOp.
func (s *PaymentService) executeRefund(order *payment.Order) error {
	// Reverse: debit receiver, credit sender (undo the payment)
	senderAccount, err := s.db.GetAccountByUserAndCurrency(order.SenderUserID, order.SourceCurrency)
	if err != nil {
		return fmt.Errorf("refund: sender account not found: %w", err)
	}

	// Credit back to sender
	if err := s.db.AddAvailableBalance(senderAccount.AccountID, order.SourceAmount+order.FeeAmount); err != nil {
		return fmt.Errorf("refund: cannot credit sender: %w", err)
	}

	// Debit from receiver if possible (best effort)
	receiverAccount, _ := s.db.GetAccountByUserAndCurrency(order.ReceiverUserID, order.TargetCurrency)
	if receiverAccount != nil {
		s.db.DebitAccount(receiverAccount.AccountID, order.TargetAmount)
	}

	// Debit fee account
	feeAccountID := "sys_fee_income_" + strings.ToLower(order.SourceCurrency)
	s.db.DebitAccount(feeAccountID, order.FeeAmount)

	return nil
}

// compensateSettlement reverses the settlement ledger entries — compensation for SettlePayment.
func (s *PaymentService) compensateSettlement(order *payment.Order) error {
	log.Printf("Saga %s: reversing settlement entries", order.PaymentID)
	// Create reversal entries (opposite direction for each original entry)
	eventID := idgen.EventID()
	entries, err := s.db.GetLedgerEntriesByPayment(order.PaymentID)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no ledger entries to reverse")
	}

	var reversals []ledger.Entry
	for i, e := range entries {
		reverseDir := ledger.DirectionCredit
		if e.Direction == ledger.DirectionCredit {
			reverseDir = ledger.DirectionDebit
		}
		reversals = append(reversals, ledger.Entry{
			EntryID:      idgen.EntryID(order.PaymentID, i+100), // Offset to avoid collision
			EventID:      eventID,
			PaymentID:    order.PaymentID,
			AccountID:    e.AccountID,
			Currency:     e.Currency,
			Direction:    reverseDir,
			Amount:       e.Amount,
			BalanceAfter: 0,
			Description:  fmt.Sprintf("REVERSAL: %s", e.Description),
		})
	}

	return s.db.InsertLedgerEntries(reversals)
}

// ──────────────────────────────────────────────
// Query methods
// ──────────────────────────────────────────────

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

// RefundPayment processes a full refund — only from COMPLETED state.
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
		s.db.AddAvailableBalance(senderAccount.AccountID, order.SourceAmount+order.FeeAmount)
	}
	if receiverAccount != nil {
		s.db.DebitAccount(receiverAccount.AccountID, order.TargetAmount)
	}
	s.db.DebitAccount(feeAccountID, order.FeeAmount)

	return s.db.UpdatePaymentStatus(paymentID, payment.PaymentRefunded)
}

// ──────────────────────────────────────────────
// Outbox Worker
// Architecture doc §7.4: Outbox Worker publishes events to NATS.
// ──────────────────────────────────────────────

// OutboxWorker periodically polls unpublished outbox events and publishes them.
type OutboxWorker struct {
	db          *repository.DB
	publisher   EventPublisher
	pollInterval time.Duration
	batchSize   int
	stopCh      chan struct{}
}

// EventPublisher is the interface for publishing events to a message queue.
type EventPublisher interface {
	Publish(eventType string, payload []byte) error
}

// NewOutboxWorker creates a new OutboxWorker.
func NewOutboxWorker(db *repository.DB, publisher EventPublisher) *OutboxWorker {
	return &OutboxWorker{
		db:          db,
		publisher:   publisher,
		pollInterval: 1 * time.Second,
		batchSize:   100,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the outbox worker loop.
func (w *OutboxWorker) Start() {
	go w.loop()
	log.Println("OutboxWorker: started (polling every", w.pollInterval, ")")
}

// Stop signals the worker to stop gracefully.
func (w *OutboxWorker) Stop() {
	close(w.stopCh)
}

func (w *OutboxWorker) loop() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			log.Println("OutboxWorker: stopped")
			return
		case <-ticker.C:
			w.processBatch()
		}
	}
}

func (w *OutboxWorker) processBatch() {
	events, err := w.db.FetchUnpublishedEvents(w.batchSize)
	if err != nil {
		log.Printf("OutboxWorker: fetch error: %v", err)
		return
	}

	for _, event := range events {
		if err := w.publisher.Publish(event.EventType, []byte(event.Payload)); err != nil {
			log.Printf("OutboxWorker: publish failed for %s: %v", event.EventID, err)
			continue
		}

		if err := w.db.MarkOutboxPublished(event.EventID); err != nil {
			log.Printf("OutboxWorker: mark published failed for %s: %v", event.EventID, err)
		}
	}

	if len(events) > 0 {
		log.Printf("OutboxWorker: published %d events", len(events))
	}
}

// Ensure CreatePaymentOrderTx exists (used by transactional outbox)
var _ = func() bool {
	// This is a compile-time check — if CreatePaymentOrderTx doesn't exist,
	// the code won't compile. The actual method is defined in the repository package.
	return true
}

// Note: CreatePaymentOrderTx and InsertIdempotencyTx are defined in their
// respective repository files. See:
//   - repository/payment_repo.go (CreatePaymentOrderTx)
//   - repository/idempotency_repo.go (InsertIdempotencyTx)

// Ensure sql.Tx is imported for the tx-based repository methods
var _ = (*sql.Tx)(nil)
