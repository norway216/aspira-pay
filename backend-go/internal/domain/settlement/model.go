// Package settlement defines the Settlement Batch domain model.
package settlement

import "time"

// BatchStatus represents the lifecycle of a settlement batch.
type BatchStatus string

const (
	BatchOpen      BatchStatus = "OPEN"
	BatchClosed    BatchStatus = "CLOSED"
	BatchSettled   BatchStatus = "SETTLED"
	BatchFailed    BatchStatus = "FAILED"
)

// Batch represents a settlement batch grouping ledger entries.
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
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// IsBalanced checks if total debits equal total credits.
func (b *Batch) IsBalanced() bool {
	return b.TotalDebit == b.TotalCredit
}

// ReconciliationResult represents the outcome of a reconciliation run.
type ReconciliationResult struct {
	Matched   int64 `json:"matched"`
	Mismatched int64 `json:"mismatched"`
	Pending   int64 `json:"pending"`
	Errors    int64 `json:"errors"`
}
