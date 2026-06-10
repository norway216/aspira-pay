// Package service implements V5 Transaction Orchestrator (§12.1).
// Manages transaction lifecycle: validate→reserve→commit→compensate.
// Ensures data consistency across payment flows.
package service

import (
	"fmt"
	"log"
	"time"

	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/idgen"
)

// TransactionOrchestrator provides unified transaction lifecycle management (§12.1-12.3).
// Applicable to: Transfer, Payment Link Pay, Card Payment, Checkout Payment.
type TransactionOrchestrator struct {
	db       *repository.DB
	activity *ActivityService
}

func NewTransactionOrchestrator(db *repository.DB, activity *ActivityService) *TransactionOrchestrator {
	return &TransactionOrchestrator{db: db, activity: activity}
}

// ReserveFunds moves available→frozen for a pending transaction (§12.3).
// Returns the reservation ID for later commit or release.
func (o *TransactionOrchestrator) ReserveFunds(accountID string, amount int64, currency string) (string, error) {
	if err := o.db.FreezeFunds(accountID, amount); err != nil {
		return "", fmt.Errorf("reserve: %w", err)
	}
	reservationID := "res_" + idgen.CardID()
	log.Printf("[Orchestrator] Reserved %d %s from %s (res=%s)", amount, currency, accountID, reservationID)
	return reservationID, nil
}

// CommitFunds finalizes a reservation: frozen→settled (§12.3).
func (o *TransactionOrchestrator) CommitFunds(accountID string, amount int64) error {
	if err := o.db.DebitAccount(accountID, amount); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ReleaseFunds cancels a reservation: frozen→available (§12.3).
func (o *TransactionOrchestrator) ReleaseFunds(accountID string, amount int64) error {
	if err := o.db.UnfreezeFunds(accountID, amount); err != nil {
		return fmt.Errorf("release: %w", err)
	}
	return nil
}

// ExecuteTransaction runs the full reserve→execute→commit cycle (§12.2).
// If any step fails, reserved funds are released (compensated).
func (o *TransactionOrchestrator) ExecuteTransaction(req OrchestratorRequest) (*OrchestratorResult, error) {
	txID := req.TransactionID
	if txID == "" { txID = "tx_" + idgen.PaymentID() }

	// Step 1: Reserve funds from source account
	reservationID, err := o.ReserveFunds(req.SourceAccountID, req.TotalDebit, req.Currency)
	if err != nil {
		return o.failResult(txID, "reserve_failed", err)
	}

	// Step 2: Execute business logic (credit receiver, charge fee, write ledger)
	if err := req.Execute(); err != nil {
		// Compensate: release reserved funds
		o.ReleaseFunds(req.SourceAccountID, req.TotalDebit)
		log.Printf("[Orchestrator] Compensated: released %d from %s", req.TotalDebit, req.SourceAccountID)
		return o.failResult(txID, "execute_failed", err)
	}

	// Step 3: Commit funds (frozen→settled)
	if err := o.CommitFunds(req.SourceAccountID, req.TotalDebit); err != nil {
		// Critical: reserve succeeded, execute succeeded, but commit failed
		// In production: retry commit, then manual intervention
		log.Printf("[Orchestrator] CRITICAL: commit failed for %s: %v", txID, err)
		return o.failResult(txID, "commit_failed", err)
	}

	// Step 4: Record activity feed (§14.5)
	o.activity.RecordActivity(req.UserID, req.ActivityType, req.RefType, txID, req.ActivityTitle, req.ActivitySubtitle, req.TotalDebit, req.Currency, "succeeded")

	// Step 5: Take balance snapshot for recovery (§13.4)
	o.db.TakeBalanceSnapshot(req.SourceAccountID, req.Currency)

	now := time.Now()
	log.Printf("[Orchestrator] Transaction %s completed: %d %s", txID, req.TotalDebit, req.Currency)
	return &OrchestratorResult{
		TransactionID: txID,
		Status:        "succeeded",
		ReservationID: reservationID,
		CompletedAt:   &now,
	}, nil
}

func (o *TransactionOrchestrator) failResult(txID, status string, err error) (*OrchestratorResult, error) {
	return &OrchestratorResult{TransactionID: txID, Status: status}, fmt.Errorf("%s: %w", status, err)
}

// ── Request/Result types ─────────────────────────

type OrchestratorRequest struct {
	TransactionID    string
	UserID           string
	SourceAccountID  string
	TotalDebit       int64
	Currency         string
	Execute          func() error
	ActivityType     string
	ActivityTitle    string
	ActivitySubtitle string
	RefType          string
}

type OrchestratorResult struct {
	TransactionID string
	Status        string
	ReservationID string
	CompletedAt   *time.Time
}

// ── Activity Feed Service (§14.5) ────────────────

type ActivityService struct {
	db *repository.DB
}

func NewActivityService(db *repository.DB) *ActivityService {
	return &ActivityService{db: db}
}

func (s *ActivityService) RecordActivity(userID, activityType, refType, refID, title, subtitle string, amount int64, currency, status string) {
	s.db.InsertActivity(idgen.EventID(), userID, activityType, refType, refID, title, subtitle, amount, currency, status)
}

func (s *ActivityService) GetUserFeed(userID string, page, pageSize int) ([]map[string]interface{}, int64, error) {
	return s.db.ListUserActivity(userID, page, pageSize)
}
