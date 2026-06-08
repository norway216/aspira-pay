// Package card defines the Card Payment Subsystem domain models.
// Architecture doc §5, §15: Card, Authorization, Transaction, FX, Fee.
package card

import "time"

// ── Card Network ─────────────────────────────────

type CardNetwork string

const (
	NetworkVisa       CardNetwork = "VISA"
	NetworkMastercard CardNetwork = "MASTERCARD"
	NetworkUnionPay   CardNetwork = "UNIONPAY"
)

// ── Card Type / Form ────────────────────────────

type CardType  string
type CardForm  string

const (
	TypeDebit  CardType = "DEBIT"
	TypeCredit CardType = "CREDIT"
	TypePrepaid CardType = "PREPAID"

	FormVirtual  CardForm = "VIRTUAL"
	FormPhysical CardForm = "PHYSICAL"
)

// ── Card Status ──────────────────────────────────

type CardStatus string

const (
	StatusIssuing   CardStatus = "ISSUING"
	StatusActive    CardStatus = "ACTIVE"
	StatusFrozen    CardStatus = "FROZEN"
	StatusLost      CardStatus = "LOST"
	StatusCancelled CardStatus = "CANCELLED"
	StatusExpired   CardStatus = "EXPIRED"
)

func (s CardStatus) IsUsable() bool {
	return s == StatusActive
}

var ValidCardTransitions = map[CardStatus][]CardStatus{
	StatusIssuing:   {StatusActive, StatusCancelled},
	StatusActive:    {StatusFrozen, StatusLost, StatusCancelled},
	StatusFrozen:    {StatusActive, StatusCancelled},
	StatusLost:      {StatusCancelled},
	StatusCancelled: {},
	StatusExpired:   {},
}

func CanTransition(from, to CardStatus) bool {
	for _, t := range ValidCardTransitions[from] {
		if t == to { return true }
	}
	return false
}

// ── Card (§15.1) ─────────────────────────────────

type Card struct {
	ID              int64     `json:"-"`
	CardID          string    `json:"card_id"`
	OwnerType       string    `json:"owner_type"`
	OwnerID         string    `json:"owner_id"`
	CardToken       string    `json:"card_token"`
	PANLast4        string    `json:"pan_last4"`
	CardNetwork     CardNetwork `json:"card_network"`
	CardType        CardType  `json:"card_type"`
	CardForm        CardForm  `json:"card_form"`
	ExpiryMonth     int       `json:"expiry_month"`
	ExpiryYear      int       `json:"expiry_year"`
	Status          CardStatus `json:"status"`
	DefaultCurrency string    `json:"default_currency,omitempty"`
	DailyLimit      int64     `json:"daily_limit"`
	MonthlyLimit    int64     `json:"monthly_limit"`
	SingleTxLimit   int64     `json:"single_tx_limit"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ── Authorization (§10.1, §15.2) ─────────────────

type AuthStatus string

const (
	AuthReceived     AuthStatus = "AUTH_RECEIVED"
	AuthValidating   AuthStatus = "AUTH_VALIDATING"
	AuthRiskChecking AuthStatus = "AUTH_RISK_CHECKING"
	AuthFXCalc       AuthStatus = "AUTH_FX_CALCULATING"
	AuthFundCheck    AuthStatus = "AUTH_FUND_CHECKING"
	AuthApproved     AuthStatus = "AUTH_APPROVED"
	AuthDeclined     AuthStatus = "AUTH_DECLINED"
	AuthReversed     AuthStatus = "AUTH_REVERSED"
	AuthExpired      AuthStatus = "AUTH_EXPIRED"
)

type DeclineReason string

const (
	DeclineCardNotActive     DeclineReason = "CARD_NOT_ACTIVE"
	DeclineCardFrozen        DeclineReason = "CARD_FROZEN"
	DeclineInsufficientFunds DeclineReason = "INSUFFICIENT_FUNDS"
	DeclineLimitExceeded     DeclineReason = "LIMIT_EXCEEDED"
	DeclineMCCBlocked        DeclineReason = "MCC_BLOCKED"
	DeclineCountryBlocked    DeclineReason = "COUNTRY_BLOCKED"
	DeclineRiskRejected      DeclineReason = "RISK_REJECTED"
	DeclineDuplicate         DeclineReason = "DUPLICATE_AUTH"
)

type CardAuthorization struct {
	ID                 int64         `json:"-"`
	AuthID             string        `json:"auth_id"`
	CardID             string        `json:"card_id"`
	MerchantName       string        `json:"merchant_name,omitempty"`
	MerchantCountry    string        `json:"merchant_country,omitempty"`
	MCC                string        `json:"merchant_category_code,omitempty"`
	TransactionAmount  int64         `json:"transaction_amount"`
	TransactionCurrency string      `json:"transaction_currency"`
	DebitAmount        int64         `json:"debit_amount"`
	DebitCurrency      string        `json:"debit_currency"`
	FXRate             string        `json:"fx_rate,omitempty"`
	FeeAmount          int64         `json:"fee_amount"`
	Status             AuthStatus    `json:"status"`
	DeclineReason      DeclineReason `json:"decline_reason,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// ── Card Transaction (§15.3) ─────────────────────

type CardTxStatus string

const (
	TxPending    CardTxStatus = "PENDING"
	TxClearing   CardTxStatus = "CLEARING"
	TxSettled    CardTxStatus = "SETTLED"
	TxRefunded   CardTxStatus = "REFUNDED"
	TxDisputed   CardTxStatus = "DISPUTED"
	TxReversed   CardTxStatus = "REVERSED"
)

type CardTransaction struct {
	ID                  int64      `json:"-"`
	TxID                string     `json:"tx_id"`
	AuthID              string     `json:"auth_id,omitempty"`
	CardID              string     `json:"card_id"`
	TransactionAmount   int64      `json:"transaction_amount"`
	TransactionCurrency string     `json:"transaction_currency"`
	DebitAmount         int64      `json:"debit_amount"`
	DebitCurrency       string     `json:"debit_currency"`
	FXRate              string     `json:"fx_rate,omitempty"`
	FeeAmount           int64      `json:"fee_amount"`
	Status              CardTxStatus `json:"status"`
	SettlementDate      *time.Time `json:"settlement_date,omitempty"`
	ReceiptHash         string     `json:"receipt_hash,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// ── FX Quote (§15.4) ─────────────────────────────

type FXQuote struct {
	ID             int64     `json:"-"`
	QuoteID        string    `json:"quote_id"`
	SourceCurrency string    `json:"source_currency"`
	TargetCurrency string    `json:"target_currency"`
	SourceAmount   int64     `json:"source_amount,omitempty"`
	TargetAmount   int64     `json:"target_amount,omitempty"`
	MidRate        string    `json:"mid_rate"`
	AppliedRate    string    `json:"applied_rate"`
	FeeRate        string    `json:"fee_rate"`
	FeeAmount      int64     `json:"fee_amount"`
	ValidUntil     time.Time `json:"valid_until"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// ── Fee Rule (§9.2) ──────────────────────────────

type FeeScenario string

const (
	FeeSameCurrency  FeeScenario = "CARD_SAME_CURRENCY_SPEND"
	FeeCrossCurrency FeeScenario = "CARD_CROSS_CURRENCY_SPEND"
	FeeATM           FeeScenario = "CARD_ATM_WITHDRAWAL"
	FeeRefund        FeeScenario = "CARD_REFUND"
	FeeChargeback    FeeScenario = "CARD_CHARGEBACK"
)

type FeeRule struct {
	ID              int64       `json:"-"`
	RuleID          string      `json:"rule_id"`
	Scenario        FeeScenario `json:"scenario"`
	SourceCurrency  string      `json:"source_currency,omitempty"`
	TargetCurrency  string      `json:"target_currency,omitempty"`
	Country         string      `json:"country,omitempty"`
	CardNetwork     string      `json:"card_network,omitempty"`
	PercentageFee   string      `json:"percentage_fee"`
	FixedFee        int64       `json:"fixed_fee"`
	MinFee          int64       `json:"min_fee,omitempty"`
	MaxFee          int64       `json:"max_fee,omitempty"`
	RiskLevel       string      `json:"risk_level,omitempty"`
	Status          string      `json:"status"`
}

// ── Spend Quote Request/Response (§16.2) ─────────

type SpendQuoteRequest struct {
	TransactionAmount   int64  `json:"transaction_amount"`
	TransactionCurrency string `json:"transaction_currency"`
	MerchantCountry     string `json:"merchant_country"`
	MCC                 string `json:"merchant_category_code"`
}

type SpendQuoteResponse struct {
	TransactionAmount   int64  `json:"transaction_amount"`
	TransactionCurrency string `json:"transaction_currency"`
	DebitAmount         int64  `json:"debit_amount"`
	DebitCurrency       string `json:"debit_currency"`
	FXRate              string `json:"fx_rate"`
	FXFee               int64  `json:"fx_fee"`
	FixedFee            int64  `json:"fixed_fee"`
	TotalFee            int64  `json:"total_fee"`
	ValidUntil          int64  `json:"valid_until"`
}

// ── Create Card Request (§16.1) ──────────────────

type CreateCardRequest struct {
	OwnerType       string `json:"owner_type"`
	OwnerID         string `json:"owner_id"`
	CardNetwork     string `json:"card_network"`
	CardForm        string `json:"card_form"`
	DefaultCurrency string `json:"default_currency"`
	DailyLimit      int64  `json:"daily_limit"`
	MonthlyLimit    int64  `json:"monthly_limit"`
}

type CreateCardResponse struct {
	CardID      string `json:"card_id"`
	CardToken   string `json:"card_token"`
	Last4       string `json:"last4"`
	CardNetwork string `json:"card_network"`
	Status      string `json:"status"`
}
