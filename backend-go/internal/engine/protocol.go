// Package engine provides the protocol definitions for C++ engine communication.
package engine

// CommandType represents the type of engine command.
type CommandType string

const (
	CmdFreezeFunds     CommandType = "FREEZE_FUNDS"
	CmdExecutePayment  CommandType = "EXECUTE_PAYMENT"
	CmdReleaseFunds    CommandType = "RELEASE_FUNDS"
	CmdRefundPayment   CommandType = "REFUND_PAYMENT"
	CmdSettlementBatch CommandType = "SETTLEMENT_BATCH"
)

// EngineResult represents the outcome of an engine operation.
type EngineResult string

const (
	ResultAccepted         EngineResult = "ACCEPTED"
	ResultRejected         EngineResult = "REJECTED"
	ResultExecuted         EngineResult = "EXECUTED"
	ResultDuplicate        EngineResult = "DUPLICATED"
	ResultInsufficientFunds EngineResult = "INSUFFICIENT_FUNDS"
)

// Command is a payment command sent to the C++ engine.
// Matches the C++ PaymentCommand struct from the architecture doc.
type Command struct {
	SequenceID     uint64      `json:"sequence_id"`
	RequestID      string      `json:"request_id"`
	PaymentID      string      `json:"payment_id"`
	CommandType    CommandType `json:"command_type"`
	FromAccount    string      `json:"from_account"`
	ToAccount      string      `json:"to_account"`
	SourceCurrency string      `json:"source_currency"`
	TargetCurrency string      `json:"target_currency"`
	SourceAmount   int64       `json:"source_amount"`
	TargetAmount   int64       `json:"target_amount"`
	FeeAmount      int64       `json:"fee_amount"`
	Timestamp      int64       `json:"timestamp"`
}

// Response is the engine's response to a command.
type Response struct {
	SequenceID uint64       `json:"sequence_id"`
	EventID    string       `json:"event_id"`
	PaymentID  string       `json:"payment_id"`
	Result     EngineResult `json:"result"`
	Message    string       `json:"message,omitempty"`
	Timestamp  int64        `json:"timestamp"`
}

// AccountBalance mirrors the C++ AccountBalance struct.
type AccountBalance struct {
	Available int64 `json:"available"`
	Frozen    int64 `json:"frozen"`
	Settled   int64 `json:"settled"`
}

// Event mirrors the C++ EngineEvent struct.
type Event struct {
	SequenceID uint64 `json:"sequence_id"`
	EventID    string `json:"event_id"`
	PaymentID  string `json:"payment_id"`
	EventType  string `json:"event_type"`
	Result     string `json:"result"`
	Timestamp  int64  `json:"timestamp"`
}
