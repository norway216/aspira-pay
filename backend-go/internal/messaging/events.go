// Package messaging provides event type definitions for the event-driven architecture.
package messaging

// Event types matching the architecture document §8.2 Topic Design.
const (
	TopicPaymentCreated      = "payment.created"
	TopicPaymentKYCChecked   = "payment.kyc_checked"
	TopicPaymentRiskChecked  = "payment.risk_checked"
	TopicPaymentQuoteLocked  = "payment.quote_locked"
	TopicEngineCommand       = "engine.command"
	TopicEngineExecuted      = "engine.executed"
	TopicEngineRejected      = "engine.rejected"
	TopicSettlementCreated   = "settlement.created"
	TopicSettlementCompleted = "settlement.completed"
	TopicChainRecorded       = "chain.recorded"
	TopicPaymentCompleted    = "payment.completed"
	TopicPaymentFailed       = "payment.failed"
	TopicAuditEvent          = "audit.event"
)

// Event is the standard event envelope (architecture doc §8.3).
type Event struct {
	EventID     string `json:"event_id"`
	EventType   string `json:"event_type"`
	AggregateID string `json:"aggregate_id"`
	PaymentID   string `json:"payment_id"`
	SequenceID  uint64 `json:"sequence_id"`
	PayloadHash string `json:"payload_hash"`
	CreatedAt   int64  `json:"created_at"`
}

// PaymentCreatedPayload is sent when a payment is created.
type PaymentCreatedPayload struct {
	PaymentID      string `json:"payment_id"`
	SenderUserID   string `json:"sender_user_id"`
	ReceiverUserID string `json:"receiver_user_id"`
	SourceCurrency string `json:"source_currency"`
	TargetCurrency string `json:"target_currency"`
	SourceAmount   int64  `json:"source_amount"`
	TargetAmount   int64  `json:"target_amount"`
	FeeAmount      int64  `json:"fee_amount"`
}

// EngineExecutedPayload is sent when the C++ engine completes execution.
type EngineExecutedPayload struct {
	PaymentID    string `json:"payment_id"`
	Result       string `json:"result"`
	FromAccount  string `json:"from_account"`
	ToAccount    string `json:"to_account"`
	SourceAmount int64  `json:"source_amount"`
	TargetAmount int64  `json:"target_amount"`
	FeeAmount    int64  `json:"fee_amount"`
}

// SettlementCompletedPayload is sent when settlement is done.
type SettlementCompletedPayload struct {
	PaymentID    string `json:"payment_id"`
	BatchID      string `json:"batch_id"`
	EntryCount   int    `json:"entry_count"`
	LedgerRootHash string `json:"ledger_root_hash"`
}
