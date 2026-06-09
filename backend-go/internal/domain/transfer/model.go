// Package transfer defines V4 Transfer & Payment Link domain models (§4-6).
package transfer

import "time"

// ── Transfer Order (§5.2) ───────────────────────

type TransferStatus string

const (
	StatusCreated              TransferStatus = "created"
	StatusQuoted               TransferStatus = "quoted"
	StatusConfirmed            TransferStatus = "confirmed"
	StatusRiskChecking         TransferStatus = "risk_checking"
	StatusProcessing           TransferStatus = "processing"
	StatusSucceeded            TransferStatus = "succeeded"
	StatusFailed               TransferStatus = "failed"
	StatusRejected             TransferStatus = "rejected"
	StatusReversed             TransferStatus = "reversed"
	StatusFailedBalance        TransferStatus = "failed_insufficient_balance"
)

var ValidTransitions = map[TransferStatus][]TransferStatus{
	StatusCreated:       {StatusQuoted, StatusRejected},
	StatusQuoted:        {StatusConfirmed, StatusRejected},
	StatusConfirmed:     {StatusRiskChecking},
	StatusRiskChecking:  {StatusProcessing, StatusRejected},
	StatusProcessing:    {StatusSucceeded, StatusFailed, StatusReversed},
	StatusFailed:        {},
	StatusRejected:      {},
	StatusReversed:      {},
	StatusSucceeded:     {},
	StatusFailedBalance: {},
}

func CanTransition(from, to TransferStatus) bool {
	for _, t := range ValidTransitions[from] {
		if t == to { return true }
	}
	return false
}

type TransferOrder struct {
	ID               int64          `json:"-"`
	TransferID       string         `json:"transfer_id"`
	PayerUserID      string         `json:"payer_user_id"`
	PayerAccountID   string         `json:"payer_account_id"`
	ReceiverUserID   string         `json:"receiver_user_id"`
	ReceiverAccountID string        `json:"receiver_account_id"`
	SourceCurrency   string         `json:"source_currency"`
	TargetCurrency   string         `json:"target_currency"`
	SourceAmount     int64          `json:"source_amount"`
	TargetAmount     int64          `json:"target_amount"`
	FeeAmount        int64          `json:"fee_amount"`
	FXRate           string         `json:"fx_rate,omitempty"`
	QuoteID          string         `json:"quote_id,omitempty"`
	PaymentLinkID    string         `json:"payment_link_id,omitempty"`
	Status           TransferStatus `json:"status"`
	Remark           string         `json:"remark,omitempty"`
	IdempotencyKey   string         `json:"idempotency_key"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
}

// ── Transfer Contact (§4.4) ─────────────────────

type TransferContact struct {
	ID                    int64     `json:"-"`
	ContactID             string    `json:"contact_id"`
	OwnerUserID           string    `json:"owner_user_id"`
	TargetUserID          string    `json:"target_user_id"`
	TargetAccountID       string    `json:"target_account_id"`
	TargetDisplayName     string    `json:"target_display_name,omitempty"`
	TargetAspiraID        string    `json:"target_aspira_id,omitempty"`
	TargetAccountNoMasked string    `json:"target_account_no_masked,omitempty"`
	TargetCurrency        string    `json:"target_currency,omitempty"`
	LastTransferAt        time.Time `json:"last_transfer_at"`
	TransferCount         int       `json:"transfer_count"`
	TotalAmount           int64     `json:"total_amount"`
	Status                string    `json:"status"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// ── Payment Link (§6.2) ─────────────────────────

type PaymentLinkStatus string

const (
	LinkPending   PaymentLinkStatus = "pending"
	LinkViewed    PaymentLinkStatus = "viewed"
	LinkQuoted    PaymentLinkStatus = "quoted"
	LinkProcessing PaymentLinkStatus = "processing"
	LinkPaid      PaymentLinkStatus = "paid"
	LinkExpired   PaymentLinkStatus = "expired"
	LinkCancelled PaymentLinkStatus = "cancelled"
	LinkFailed    PaymentLinkStatus = "failed"
)

type PaymentLink struct {
	ID                  int64             `json:"-"`
	PaymentLinkID       string            `json:"payment_link_id"`
	LinkTokenHash       string            `json:"-"`
	LinkTokenPrefix     string            `json:"link_token_prefix"`
	LinkToken           string            `json:"link_token,omitempty"` // Only returned on creation
	CreatorUserID       string            `json:"creator_user_id"`
	ReceiverAccountID   string            `json:"receiver_account_id"`
	Amount              int64             `json:"amount"`
	Currency            string            `json:"currency"`
	Title               string            `json:"title,omitempty"`
	Description         string            `json:"description,omitempty"`
	ExpireAt            time.Time         `json:"expire_at"`
	MaxPayCount         int               `json:"max_pay_count"`
	PaidCount           int               `json:"paid_count"`
	Status              PaymentLinkStatus `json:"status"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	PaidAt              *time.Time        `json:"paid_at,omitempty"`
	CancelledAt         *time.Time        `json:"cancelled_at,omitempty"`
}

// ── API Request/Response types ──────────────────

type ResolveRecipientRequest struct {
	RecipientType  string `json:"recipient_type"`  // aspira_id, phone, email, account_no
	RecipientValue string `json:"recipient_value"`
	Currency       string `json:"currency"`
}

type ResolveRecipientResponse struct {
	RecipientUserID    string `json:"recipient_user_id"`
	RecipientAccountID string `json:"recipient_account_id"`
	DisplayName        string `json:"display_name"`
	AspiraID           string `json:"aspira_id,omitempty"`
	AccountNoMasked    string `json:"account_no_masked,omitempty"`
	Currency           string `json:"currency"`
	Status             string `json:"status"`
}

type TransferQuoteRequest struct {
	SourceAccountID string `json:"source_account_id"`
	TargetAccountID string `json:"target_account_id"`
	SourceCurrency  string `json:"source_currency"`
	TargetCurrency  string `json:"target_currency"`
	Amount          int64  `json:"amount"`
	Remark          string `json:"remark,omitempty"`
}

type TransferQuoteResponse struct {
	QuoteID            string `json:"quote_id"`
	Amount             int64  `json:"amount"`
	SourceCurrency     string `json:"source_currency"`
	TargetCurrency     string `json:"target_currency"`
	FXRate             string `json:"fx_rate"`
	Fee                int64  `json:"fee"`
	TotalDebitAmount   int64  `json:"total_debit_amount"`
	TargetReceiveAmount int64 `json:"target_receive_amount"`
	QuoteExpireAt      int64  `json:"quote_expire_at"`
}

type TransferConfirmRequest struct {
	QuoteID string `json:"quote_id"`
	PIN     string `json:"pin,omitempty"`
}

type CreatePaymentLinkRequest struct {
	ReceiverAccountID string `json:"receiver_account_id"`
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
	Title             string `json:"title,omitempty"`
	Description       string `json:"description,omitempty"`
	ExpireMinutes     int    `json:"expire_minutes"`
}
