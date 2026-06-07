// Package chain defines the Blockchain Audit Layer domain models.
package chain

import "time"

// ChainBlock represents a block in the hash chain.
type ChainBlock struct {
	ID          int64     `json:"-"`
	BlockHeight int64     `json:"block_height"`
	BlockHash   string    `json:"block_hash"`
	PrevHash    string    `json:"prev_hash"`
	MerkleRoot  string    `json:"merkle_root"`
	EventCount  int       `json:"event_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// ChainEvent represents a single event recorded on the chain.
type ChainEvent struct {
	ID          int64     `json:"-"`
	EventID     string    `json:"event_id"`
	BlockHeight int64     `json:"block_height"`
	PaymentID   string    `json:"payment_id"`
	EventType   string    `json:"event_type"`
	PayloadHash string    `json:"payload_hash"`
	CreatedAt   time.Time `json:"created_at"`
}

// Event types for on-chain recording (matching architecture doc §10.3).
const (
	EventPaymentCreated     = "PAYMENT_CREATED"
	EventRiskApproved       = "RISK_APPROVED"
	EventFundsFrozen        = "FUNDS_FROZEN"
	EventEngineExecuted     = "ENGINE_EXECUTED"
	EventSettlementCompleted = "SETTLEMENT_COMPLETED"
	EventPaymentCompleted   = "PAYMENT_COMPLETED"
	EventPaymentRejected    = "PAYMENT_REJECTED"
	EventPaymentRefunded    = "PAYMENT_REFUNDED"
)

// ChainPaymentRecord is the on-chain record structure (architecture doc §4.9.1).
// Only hashes are recorded, never sensitive plaintext data.
type ChainPaymentRecord struct {
	ChainTxID         string `json:"chain_tx_id"`
	PaymentIDHash     string `json:"payment_id_hash"`
	SenderHash        string `json:"sender_hash"`
	ReceiverHash      string `json:"receiver_hash"`
	AmountHash        string `json:"amount_hash"`
	CurrencyPair      string `json:"currency_pair"`
	Status            string `json:"status"`
	SettlementBatchID string `json:"settlement_batch_id"`
	LedgerRootHash    string `json:"ledger_root_hash"`
	Timestamp         int64  `json:"timestamp"`
	Signature         string `json:"signature"`
}

// AuditTrail is a complete audit trail for a payment.
type AuditTrail struct {
	PaymentID string       `json:"payment_id"`
	Blocks    []ChainBlock `json:"blocks"`
	Events    []ChainEvent `json:"events"`
	Verified  bool         `json:"verified"`
}

// BatchSubmission represents a batch of events to submit on-chain.
type BatchSubmission struct {
	BatchID        string   `json:"batch_id"`
	EventHashes    []string `json:"event_hashes"`
	MerkleRoot     string   `json:"merkle_root"`
	PrevBlockHash  string   `json:"prev_block_hash"`
	BlockHeight    int64    `json:"block_height"`
}
