// Package payment defines the Payment Order domain model and state machine.
package payment

import "time"

// PaymentStatus represents the state of a payment order.
type PaymentStatus string

// Full payment state machine as defined in the architecture document.
const (
	PaymentCreated              PaymentStatus = "CREATED"
	PaymentKYCChecked           PaymentStatus = "KYC_CHECKED"
	PaymentRiskChecked          PaymentStatus = "RISK_CHECKED"
	PaymentQuoteLocked          PaymentStatus = "QUOTE_LOCKED"
	PaymentFundsFreezeRequested PaymentStatus = "FUNDS_FREEZE_REQUESTED"
	PaymentSubmittedToEngine    PaymentStatus = "SUBMITTED_TO_ENGINE"
	PaymentEngineExecuted       PaymentStatus = "ENGINE_EXECUTED"
	PaymentSettlementPending    PaymentStatus = "SETTLEMENT_PENDING"
	PaymentSettled              PaymentStatus = "SETTLED"
	PaymentChainConfirmed       PaymentStatus = "CHAIN_CONFIRMED"
	PaymentCompleted            PaymentStatus = "COMPLETED"

	// Abnormal/terminal states
	PaymentRejected     PaymentStatus = "REJECTED"
	PaymentFailed       PaymentStatus = "FAILED"
	PaymentCancelled    PaymentStatus = "CANCELLED"
	PaymentRefunded     PaymentStatus = "REFUNDED"
	PaymentManualReview PaymentStatus = "MANUAL_REVIEW"
)

// IsTerminal returns true if the status is a terminal state.
func (s PaymentStatus) IsTerminal() bool {
	switch s {
	case PaymentCompleted, PaymentRejected, PaymentFailed, PaymentCancelled, PaymentRefunded:
		return true
	}
	return false
}

// IsSuccessful returns true if the payment reached COMPLETED.
func (s PaymentStatus) IsSuccessful() bool {
	return s == PaymentCompleted
}

// ValidTransitions defines the payment state machine transitions.
var ValidTransitions = map[PaymentStatus][]PaymentStatus{
	PaymentCreated:              {PaymentKYCChecked, PaymentRejected, PaymentCancelled},
	PaymentKYCChecked:           {PaymentRiskChecked, PaymentRejected},
	PaymentRiskChecked:          {PaymentQuoteLocked, PaymentRejected, PaymentManualReview},
	PaymentQuoteLocked:          {PaymentFundsFreezeRequested, PaymentRejected, PaymentCancelled},
	PaymentFundsFreezeRequested: {PaymentSubmittedToEngine, PaymentFailed},
	PaymentSubmittedToEngine:    {PaymentEngineExecuted, PaymentFailed},
	PaymentEngineExecuted:       {PaymentSettlementPending, PaymentFailed},
	PaymentSettlementPending:    {PaymentSettled, PaymentFailed},
	PaymentSettled:              {PaymentChainConfirmed, PaymentFailed},
	PaymentChainConfirmed:       {PaymentCompleted, PaymentFailed},
	PaymentCompleted:            {PaymentRefunded},
	PaymentManualReview:         {PaymentRiskChecked, PaymentRejected},
	PaymentFailed:               {}, // Terminal
	PaymentRejected:             {}, // Terminal
	PaymentCancelled:            {}, // Terminal
	PaymentRefunded:             {}, // Terminal
}

// CanTransition checks if transitioning from one status to another is valid.
func CanTransition(from, to PaymentStatus) bool {
	allowed, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// Order represents a payment order in the system.
type Order struct {
	ID             int64         `json:"-"`
	PaymentID      string        `json:"payment_id"`
	RequestID      string        `json:"request_id"`
	SenderUserID   string        `json:"sender_user_id"`
	ReceiverUserID string        `json:"receiver_user_id"`
	SourceCurrency string        `json:"source_currency"`
	TargetCurrency string        `json:"target_currency"`
	SourceAmount   int64         `json:"source_amount"`
	TargetAmount   int64         `json:"target_amount"`
	FeeAmount      int64         `json:"fee_amount"`
	FXRate         string        `json:"fx_rate"`
	Status         PaymentStatus `json:"status"`
	RiskScore      int           `json:"risk_score"`
	RiskReasons    string        `json:"risk_reasons,omitempty"`
	QuoteID        string        `json:"quote_id,omitempty"`
	ChainTxID      string        `json:"chain_tx_id,omitempty"`
	Purpose        string        `json:"purpose,omitempty"`
	CountryFrom    string        `json:"country_from,omitempty"`
	CountryTo      string        `json:"country_to,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// CreateRequest is the API input for creating a payment.
type CreateRequest struct {
	SenderUserID   string `json:"sender_user_id" binding:"required"`
	ReceiverUserID string `json:"receiver_user_id" binding:"required"`
	SourceCurrency string `json:"source_currency" binding:"required"`
	TargetCurrency string `json:"target_currency" binding:"required"`
	SourceAmount   int64  `json:"source_amount" binding:"required"`
	Purpose        string `json:"purpose"`
	CountryFrom    string `json:"country_from"`
	CountryTo      string `json:"country_to"`
}

// CreateResponse is the API output for payment creation.
type CreateResponse struct {
	PaymentID      string        `json:"payment_id"`
	Status         PaymentStatus `json:"status"`
	SourceAmount   int64         `json:"source_amount"`
	TargetAmount   int64         `json:"target_amount"`
	FeeAmount      int64         `json:"fee_amount"`
	FXRate         string        `json:"fx_rate"`
	QuoteID        string        `json:"quote_id,omitempty"`
	CreatedAt      int64         `json:"created_at"`
}

// ListQuery filters for listing payments.
type ListQuery struct {
	Status   string `form:"status"`
	SenderID string `form:"sender_id"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}
