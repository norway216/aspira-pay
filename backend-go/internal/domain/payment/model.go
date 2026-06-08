// Package payment defines the Payment Order domain model and state machine.
// Architecture doc V3.0 §5.2.3-5.2.4.
package payment

import "time"

// PaymentStatus represents the state of a payment order.
type PaymentStatus string

// V3.0 Payment State Machine (§5.2.3)
const (
	// Normal flow
	PaymentCreated             PaymentStatus = "CREATED"
	PaymentQuoteLocked         PaymentStatus = "QUOTE_LOCKED"
	PaymentCompliancePrechecked PaymentStatus = "COMPLIANCE_PRECHECKED"
	PaymentPending             PaymentStatus = "PAYMENT_PENDING"
	PaymentExecuting           PaymentStatus = "PAYMENT_EXECUTING"
	PaymentConfirmed           PaymentStatus = "PAYMENT_CONFIRMED"
	PaymentSettlementProofed   PaymentStatus = "SETTLEMENT_PROOFED"
	PaymentReconciled          PaymentStatus = "RECONCILED"
	PaymentClosed              PaymentStatus = "CLOSED"

	// Abnormal / terminal states (§5.2.3)
	PaymentRiskRejected  PaymentStatus = "RISK_REJECTED"
	PaymentFailed        PaymentStatus = "PAYMENT_FAILED"
	PaymentRefundPending PaymentStatus = "REFUND_PENDING"
	PaymentRefunded      PaymentStatus = "REFUNDED"
	PaymentDisputed      PaymentStatus = "DISPUTED"
	PaymentFrozen        PaymentStatus = "FROZEN"
	PaymentManualReview  PaymentStatus = "MANUAL_REVIEW"
	PaymentCancelled     PaymentStatus = "CANCELLED"

	// Legacy aliases for backward compatibility
	PaymentRejected  PaymentStatus = "REJECTED"  // → RISK_REJECTED
	PaymentCompleted PaymentStatus = "COMPLETED"  // → CLOSED
)

// IsTerminal returns true if the status is a terminal state.
func (s PaymentStatus) IsTerminal() bool {
	switch s {
	case PaymentClosed, PaymentRiskRejected, PaymentRejected,
		PaymentFailed, PaymentRefunded, PaymentCancelled:
		return true
	}
	return false
}

// IsSuccessful returns true if the payment reached a successful terminal state.
func (s PaymentStatus) IsSuccessful() bool {
	return s == PaymentClosed || s == PaymentCompleted
}

// IsPending returns true if the payment is still in progress.
func (s PaymentStatus) IsPending() bool {
	return !s.IsTerminal()
}

// ValidTransitions defines the V3.0 payment state machine transitions (§5.2.4).
// All non-terminal states can transition to PaymentFailed (failure can happen at any step).
var ValidTransitions = map[PaymentStatus][]PaymentStatus{
	// Normal flow
	PaymentCreated:              {PaymentQuoteLocked, PaymentCancelled, PaymentRiskRejected, PaymentFailed},
	PaymentQuoteLocked:          {PaymentCompliancePrechecked, PaymentRiskRejected, PaymentFailed},
	PaymentCompliancePrechecked: {PaymentPending, PaymentManualReview, PaymentFailed},
	PaymentPending:              {PaymentExecuting, PaymentCancelled, PaymentFailed},
	PaymentExecuting:            {PaymentConfirmed, PaymentFailed},
	PaymentConfirmed:            {PaymentSettlementProofed, PaymentFailed},
	PaymentSettlementProofed:    {PaymentReconciled, PaymentDisputed, PaymentFailed},
	PaymentReconciled:           {PaymentClosed, PaymentDisputed, PaymentFailed},

	// Abnormal transitions
	PaymentClosed:       {PaymentRefundPending, PaymentDisputed},
	PaymentRefundPending: {PaymentRefunded, PaymentDisputed},
	PaymentManualReview:  {PaymentPending, PaymentRiskRejected, PaymentFrozen, PaymentFailed},
	PaymentDisputed:      {PaymentReconciled, PaymentClosed},

	// Terminal states
	PaymentFailed:       {},
	PaymentRiskRejected: {},
	PaymentRejected:     {},
	PaymentRefunded:     {},
	PaymentCancelled:    {},
	PaymentFrozen:       {},
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

// Order represents a payment order in the system (§6.2).
type Order struct {
	ID               int64         `json:"-"`
	OrderID          string        `json:"order_id"`
	MerchantID       string        `json:"merchant_id"`
	CustomerIDHash   string        `json:"customer_id_hash,omitempty"`
	PaymentID        string        `json:"payment_id"`
	RequestID        string        `json:"request_id"`
	SenderUserID     string        `json:"sender_user_id"`
	ReceiverUserID   string        `json:"receiver_user_id"`
	SourceCurrency   string        `json:"source_currency"`
	TargetCurrency   string        `json:"target_currency"`
	SourceAmount     int64         `json:"source_amount"`
	TargetAmount     int64         `json:"target_amount"`
	FeeAmount        int64         `json:"fee_amount"`
	FXRate           string        `json:"fx_rate"`
	Status           PaymentStatus `json:"status"`
	RiskScore        int           `json:"risk_score"`
	RiskReasons      string        `json:"risk_reasons,omitempty"`
	QuoteID          string        `json:"quote_id,omitempty"`
	ChannelID        string        `json:"channel_id,omitempty"`   // §5.8: payment channel
	ChainTxID        string        `json:"chain_tx_id,omitempty"`
	Purpose          string        `json:"purpose,omitempty"`
	CountryFrom      string        `json:"country_from,omitempty"`
	CountryTo        string        `json:"country_to,omitempty"`
	ReceiptHash      string        `json:"receipt_hash,omitempty"` // §5.11.3: payment receipt hash
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
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
	PaymentID    string        `json:"payment_id"`
	Status       PaymentStatus `json:"status"`
	SourceAmount int64         `json:"source_amount"`
	TargetAmount int64         `json:"target_amount"`
	FeeAmount    int64         `json:"fee_amount"`
	FXRate       string        `json:"fx_rate"`
	QuoteID      string        `json:"quote_id,omitempty"`
	CreatedAt    int64         `json:"created_at"`
}

// ListQuery filters for listing payments.
type ListQuery struct {
	Status   string `form:"status"`
	SenderID string `form:"sender_id"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Cursor   string `form:"cursor"`
}
