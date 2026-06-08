// Package settlement defines the V3.0 Settlement domain (§5.6).
package settlement

import "time"

// BatchStatus represents the lifecycle of a settlement batch (§5.6.4).
type BatchStatus string

const (
	BatchCreated          BatchStatus = "BATCH_CREATED"
	BatchCalculated       BatchStatus = "BATCH_CALCULATED"
	BatchApproved         BatchStatus = "BATCH_APPROVED"
	SettlementInstructed  BatchStatus = "SETTLEMENT_INSTRUCTED"
	SettlementProcessing  BatchStatus = "SETTLEMENT_PROCESSING"
	SettlementConfirmed   BatchStatus = "SETTLEMENT_CONFIRMED"
	BatchReconciled       BatchStatus = "RECONCILED"
	BatchClosed           BatchStatus = "BATCH_CLOSED"
	BatchFailed           BatchStatus = "BATCH_FAILED"

	// Legacy aliases
	BatchOpen    BatchStatus = "OPEN"    // → BATCH_CREATED
	BatchSettled BatchStatus = "SETTLED" // → SETTLEMENT_CONFIRMED
)

func (s BatchStatus) IsTerminal() bool {
	switch s {
	case BatchClosed, BatchFailed:
		return true
	}
	return false
}

// Batch represents a settlement batch.
type Batch struct {
	ID             int64       `json:"-"`
	BatchID        string      `json:"batch_id"`
	Currency       string      `json:"currency"`
	TotalDebit     int64       `json:"total_debit"`
	TotalCredit    int64       `json:"total_credit"`
	EntryCount     int         `json:"entry_count"`
	Status         BatchStatus `json:"status"`
	LedgerRootHash string      `json:"ledger_root_hash,omitempty"`
	ChainTxID      string      `json:"chain_tx_id,omitempty"`
	ApprovedBy     string      `json:"approved_by,omitempty"`
	SettlementDate string      `json:"settlement_date,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

func (b *Batch) IsBalanced() bool {
	return b.TotalDebit == b.TotalCredit
}

// SettlementDetail is an individual entry in a settlement batch.
type SettlementDetail struct {
	ID        int64     `json:"-"`
	DetailID  string    `json:"detail_id"`
	BatchID   string    `json:"batch_id"`
	OrderID   string    `json:"order_id"`
	PaymentID string    `json:"payment_id"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	FeeAmount int64     `json:"fee_amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ReconciliationResult represents the outcome of a reconciliation run (§5.7).
type ReconciliationResult struct {
	Matched    int64 `json:"matched"`
	Mismatched int64 `json:"mismatched"`
	Pending    int64 `json:"pending"`
	Errors     int64 `json:"errors"`
}
