// Package ledger defines the Ledger Entry and Account domain models.
package ledger

import "time"

// Direction indicates debit or credit in double-entry accounting.
type Direction string

const (
	DirectionDebit  Direction = "DEBIT"
	DirectionCredit Direction = "CREDIT"
)

// Entry represents a single ledger entry (append-only).
// Each entry is part of a double-entry pair: every debit must have
// a corresponding credit of equal amount, ensuring 借贷平衡.
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

// Account represents a financial account with balance tracking.
type Account struct {
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

// TotalBalance returns the sum of all balance components.
func (a *Account) TotalBalance() int64 {
	return a.AvailableBalance + a.FrozenBalance + a.SettledBalance
}

// CanDebit checks if there are sufficient available funds.
func (a *Account) CanDebit(amount int64) bool {
	return a.AvailableBalance >= amount
}

// IsSystemAccount checks if this is a platform internal account.
func (a *Account) IsSystemAccount() bool {
	return a.UserID == "system"
}

// LedgerSummary provides a summary of entries for a payment.
type LedgerSummary struct {
	PaymentID    string  `json:"payment_id"`
	TotalDebit   int64   `json:"total_debit"`
	TotalCredit  int64   `json:"total_credit"`
	EntryCount   int     `json:"entry_count"`
	IsBalanced   bool    `json:"is_balanced"`
	Entries      []Entry `json:"entries"`
}

// CheckBalance verifies that total debits equal total credits per currency.
// Cross-border payments involve different currencies, so balance must be checked
// within each currency separately.
func CheckBalance(entries []Entry) bool {
	balances := make(map[string]int64) // currency -> net (debit - credit)
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
