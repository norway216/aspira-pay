// Package ledger defines the V3.0 Ledger domain models (§5.5).
// Introduces Voucher concept for proper double-entry accounting.
package ledger

import "time"

// ── Direction ──────────────────────────────────

type Direction string

const (
	DirectionDebit  Direction = "DEBIT"
	DirectionCredit Direction = "CREDIT"
)

// ── Account Types (§5.5.2) ─────────────────────

type AccountType string

const (
	AccountUserAvailable    AccountType = "USER_AVAILABLE"
	AccountUserFrozen       AccountType = "USER_FROZEN"
	AccountMerchantSettlement AccountType = "MERCHANT_SETTLEMENT"
	AccountPlatformFee      AccountType = "PLATFORM_FEE"
	AccountChannelClearing  AccountType = "CHANNEL_CLEARING"
	AccountFXGainLoss       AccountType = "FX_GAIN_LOSS"
	AccountRefund           AccountType = "REFUND"
	AccountReserve          AccountType = "RESERVE"
)

type OwnerType string

const (
	OwnerUser     OwnerType = "USER"
	OwnerMerchant OwnerType = "MERCHANT"
	OwnerSystem   OwnerType = "SYSTEM"
	OwnerChannel  OwnerType = "CHANNEL"
)

// ── Account (§5.5.4) ───────────────────────────

type Account struct {
	ID               int64       `json:"-"`
	AccountNo        string      `json:"account_no"`
	OwnerType        OwnerType   `json:"owner_type"`
	OwnerID          string      `json:"owner_id"`
	AccountType      AccountType `json:"account_type"`
	Currency         string      `json:"currency"`
	Status           string      `json:"status"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// ── Account Balance (§5.5.4) ────────────────────

type AccountBalance struct {
	AccountNo        string    `json:"account_no"`
	AvailableBalance int64     `json:"available_balance"`
	FrozenBalance    int64     `json:"frozen_balance"`
	Currency         string    `json:"currency"`
	Version          int64     `json:"version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ── Voucher (§5.5.4) ────────────────────────────

type Voucher struct {
	ID           int64     `json:"-"`
	VoucherNo    string    `json:"voucher_no"`
	BusinessType string    `json:"business_type"`
	BusinessID   string    `json:"business_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// ── V3 Voucher Entry (§5.5.4) ───────────────────

type VoucherEntry struct {
	ID          int64     `json:"-"`
	EntryID     string    `json:"entry_id"`
	VoucherNo   string    `json:"voucher_no"`
	AccountNo   string    `json:"account_no"`
	Direction   Direction `json:"direction"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ── Legacy Entry (backward compat) ──────────────
// Used by existing ledger_repo.go and settlement_svc.go.

type Entry struct {
	ID           int64     `json:"-"`
	EntryID      string    `json:"entry_id"`
	EventID      string    `json:"event_id"`
	PaymentID    string    `json:"payment_id"`
	AccountID    string    `json:"account_id"`
	Currency     string    `json:"currency"`
	Direction    Direction `json:"direction"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balance_after"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ── Ledger Summary ──────────────────────────────

type LedgerSummary struct {
	PaymentID   string  `json:"payment_id"`
	TotalDebit  int64   `json:"total_debit"`
	TotalCredit int64   `json:"total_credit"`
	EntryCount  int     `json:"entry_count"`
	IsBalanced  bool    `json:"is_balanced"`
	Entries     []Entry `json:"entries"`
}

// CheckBalance verifies that total debits equal total credits per currency.
func CheckBalance(entries []Entry) bool {
	balances := make(map[string]int64)
	for _, e := range entries {
		switch e.Direction {
		case DirectionDebit:
			balances[e.Currency] += e.Amount
		case DirectionCredit:
			balances[e.Currency] -= e.Amount
		}
	}
	for _, net := range balances {
		if net != 0 {
			return false
		}
	}
	return true
}

// ── Legacy Account (backward compat) ────────────

type LegacyAccount struct {
	ID               int64     `json:"-"`
	AccountID        string    `json:"account_id"`
	UserID           string    `json:"user_id"`
	Currency         string    `json:"currency"`
	AvailableBalance int64     `json:"available_balance"`
	FrozenBalance    int64     `json:"frozen_balance"`
	SettledBalance   int64     `json:"settled_balance"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (a *LegacyAccount) TotalBalance() int64 {
	return a.AvailableBalance + a.FrozenBalance + a.SettledBalance
}
func (a *LegacyAccount) CanDebit(amount int64) bool {
	return a.AvailableBalance >= amount
}
func (a *LegacyAccount) IsSystemAccount() bool {
	return a.UserID == "system"
}
