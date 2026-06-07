// Package chain defines the Blockchain Audit Layer domain models.
// Architecture doc §5-7, §11: Hash chain, Merkle batch proofs, audit signatures.
package chain

import "time"

// ──────────────────────────────────────────────
// Chain Status Constants (§10.2)
// ──────────────────────────────────────────────

type BatchStatus string

const (
	BatchPending           BatchStatus = "PENDING"
	BatchBuilding           BatchStatus = "BUILDING"
	BatchSubmitting         BatchStatus = "SUBMITTING"
	BatchConfirmed          BatchStatus = "CONFIRMED"
	BatchFailedRetryable    BatchStatus = "FAILED_RETRYABLE"
	BatchFailedManualReview BatchStatus = "FAILED_MANUAL_REVIEW"
)

// IsTerminal returns true if the batch has reached a final state.
func (s BatchStatus) IsTerminal() bool {
	switch s {
	case BatchConfirmed, BatchFailedManualReview:
		return true
	}
	return false
}

// ──────────────────────────────────────────────
// Chain Block (§7.2)
// ──────────────────────────────────────────────

type ChainBlock struct {
	ID               int64     `json:"-"`
	BlockHeight      int64     `json:"block_height"`
	BlockHash        string    `json:"block_hash"`
	PrevHash         string    `json:"prev_hash"`
	MerkleRoot       string    `json:"merkle_root"`
	EventCount       int       `json:"event_count"`
	BatchID          string    `json:"batch_id,omitempty"`
	StartSequenceID  int64     `json:"start_sequence_id"`
	EndSequenceID    int64     `json:"end_sequence_id"`
	AuditSignature   string    `json:"audit_signature,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────
// Chain Event (§7.3)
// ──────────────────────────────────────────────

type ChainEvent struct {
	ID          int64     `json:"-"`
	EventID     string    `json:"event_id"`
	BlockHeight int64     `json:"block_height"`
	PaymentID   string    `json:"payment_id"`
	EventType   string    `json:"event_type"`
	PayloadHash string    `json:"payload_hash"`
	BatchID     string    `json:"batch_id,omitempty"`
	MerkleProof []string  `json:"merkle_proof,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────
// Chain Batch (§14.1)
// ──────────────────────────────────────────────

type ChainBatch struct {
	ID              int64       `json:"-"`
	BatchID         string      `json:"batch_id"`
	MerkleRoot      string      `json:"merkle_root"`
	LedgerRootHash  string      `json:"ledger_root_hash,omitempty"`
	EventCount      int         `json:"event_count"`
	StartSequenceID int64       `json:"start_sequence_id"`
	EndSequenceID   int64       `json:"end_sequence_id"`
	Status          BatchStatus `json:"status"`
	AuditSignature  string      `json:"audit_signature,omitempty"`
	BlockHeight     *int64      `json:"block_height,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	SubmittedAt     *time.Time  `json:"submitted_at,omitempty"`
	ConfirmedAt     *time.Time  `json:"confirmed_at,omitempty"`
}

// ──────────────────────────────────────────────
// Submit Log (§14.2)
// ──────────────────────────────────────────────

type ChainSubmitLog struct {
	ID          int64     `json:"-"`
	BatchID     string    `json:"batch_id"`
	ChainType   string    `json:"chain_type"`
	ChainTxID   string    `json:"chain_tx_id,omitempty"`
	Status      string    `json:"status"`
	ErrorMessage string   `json:"error_message,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────
// Retry Queue (§14.3)
// ──────────────────────────────────────────────

type ChainRetryEntry struct {
	ID          int64     `json:"-"`
	BatchID     string    `json:"batch_id"`
	RetryCount  int       `json:"retry_count"`
	MaxRetries  int       `json:"max_retries"`
	NextRetryAt time.Time `json:"next_retry_at"`
	LastError   string    `json:"last_error,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ──────────────────────────────────────────────
// Audit Record (§6.3)
// ──────────────────────────────────────────────

type ChainAuditRecord struct {
	BatchID         string `json:"batch_id"`
	MerkleRoot      string `json:"merkle_root"`
	LedgerRootHash  string `json:"ledger_root_hash"`
	EventCount      int    `json:"event_count"`
	StartSequenceID uint64 `json:"start_sequence_id"`
	EndSequenceID   uint64 `json:"end_sequence_id"`
	PreviousHash    string `json:"previous_hash"`
	Timestamp       int64  `json:"timestamp"`
	AuditSignature  string `json:"audit_signature"`
}

// ──────────────────────────────────────────────
// Audit Verification (§15)
// ──────────────────────────────────────────────

type AuditVerification struct {
	PaymentID    string   `json:"payment_id"`
	EventHash    string   `json:"event_hash"`
	BatchID      string   `json:"batch_id"`
	MerkleRoot   string   `json:"merkle_root"`
	MerkleProof  []string `json:"merkle_proof"`
	LeafIndex    int      `json:"leaf_index"`
	BlockHash    string   `json:"block_hash"`
	BlockHeight  int64    `json:"block_height"`
	ChainTxID    string   `json:"chain_tx_id,omitempty"`
	AuditSig     string   `json:"audit_signature,omitempty"`
	Verified     bool     `json:"verified"`
}

// ──────────────────────────────────────────────
// Event Type Constants
// ──────────────────────────────────────────────

const (
	EventPaymentCreated      = "PAYMENT_CREATED"
	EventRiskApproved        = "RISK_APPROVED"
	EventFundsFrozen         = "FUNDS_FROZEN"
	EventEngineExecuted      = "ENGINE_EXECUTED"
	EventSettlementCompleted = "SETTLEMENT_COMPLETED"
	EventPaymentCompleted    = "PAYMENT_COMPLETED"
	EventPaymentRejected     = "PAYMENT_REJECTED"
	EventPaymentRefunded     = "PAYMENT_REFUNDED"
)

// ──────────────────────────────────────────────
// Audit Trail (§15)
// ──────────────────────────────────────────────

type AuditTrail struct {
	PaymentID string       `json:"payment_id"`
	Blocks    []ChainBlock `json:"blocks"`
	Events    []ChainEvent `json:"events"`
	Verified  bool         `json:"verified"`
}

// ──────────────────────────────────────────────
// Chain Metrics Snapshot (§16)
// ──────────────────────────────────────────────

type ChainMetrics struct {
	EventsReceived   int64 `json:"events_received_total"`
	BatchesCreated   int64 `json:"batches_created_total"`
	BatchesConfirmed int64 `json:"batches_confirmed_total"`
	BatchesFailed    int64 `json:"batches_failed_total"`
	RetryQueueDepth  int64 `json:"retry_queue_depth"`
	PendingBatches   int64 `json:"pending_batches"`
}
