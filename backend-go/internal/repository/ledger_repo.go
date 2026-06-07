package repository

import (
	"fmt"

	"github.com/aspira/aspira-pay/internal/domain/ledger"
)

// InsertLedgerEntry inserts a single ledger entry (append-only).
func (db *DB) InsertLedgerEntry(e *ledger.Entry) error {
	query := `
		INSERT INTO ledger_entries (entry_id, event_id, payment_id, account_id, currency, direction, amount, balance_after, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return db.QueryRow(query,
		e.EntryID, e.EventID, e.PaymentID, e.AccountID,
		e.Currency, e.Direction, e.Amount, e.BalanceAfter, e.Description,
	).Scan(&e.ID, &e.CreatedAt)
}

// InsertLedgerEntries inserts multiple entries in a single transaction.
func (db *DB) InsertLedgerEntries(entries []ledger.Entry) error {
	tx, err := db.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i := range entries {
		query := `
			INSERT INTO ledger_entries (entry_id, event_id, payment_id, account_id, currency, direction, amount, balance_after, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, created_at`
		if err := tx.QueryRow(query,
			entries[i].EntryID, entries[i].EventID, entries[i].PaymentID,
			entries[i].AccountID, entries[i].Currency, entries[i].Direction,
			entries[i].Amount, entries[i].BalanceAfter, entries[i].Description,
		).Scan(&entries[i].ID, &entries[i].CreatedAt); err != nil {
			return fmt.Errorf("ledger entry insert failed at index %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// GetLedgerEntriesByPayment retrieves all ledger entries for a payment.
func (db *DB) GetLedgerEntriesByPayment(paymentID string) ([]ledger.Entry, error) {
	query := `SELECT id, entry_id, event_id, payment_id, account_id, currency, direction, amount, balance_after, COALESCE(description, ''), created_at
		FROM ledger_entries WHERE payment_id = $1 ORDER BY created_at ASC`
	rows, err := db.Query(query, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ledger.Entry
	for rows.Next() {
		var e ledger.Entry
		if err := rows.Scan(
			&e.ID, &e.EntryID, &e.EventID, &e.PaymentID,
			&e.AccountID, &e.Currency, &e.Direction,
			&e.Amount, &e.BalanceAfter, &e.Description, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// GetAccountBalance retrieves the current balance for an account.
func (db *DB) GetAccountBalance(accountID string) (available, frozen, settled int64, err error) {
	query := `SELECT available_balance, frozen_balance, settled_balance FROM accounts WHERE account_id = $1`
	err = db.QueryRow(query, accountID).Scan(&available, &frozen, &settled)
	return
}

// GetLedgerSummary returns a balanced summary for a payment.
func (db *DB) GetLedgerSummary(paymentID string) (*ledger.LedgerSummary, error) {
	entries, err := db.GetLedgerEntriesByPayment(paymentID)
	if err != nil {
		return nil, err
	}

	summary := &ledger.LedgerSummary{
		PaymentID:  paymentID,
		EntryCount: len(entries),
		Entries:    entries,
	}

	for _, e := range entries {
		switch e.Direction {
		case ledger.DirectionDebit:
			summary.TotalDebit += e.Amount
		case ledger.DirectionCredit:
			summary.TotalCredit += e.Amount
		}
	}
	summary.IsBalanced = summary.TotalDebit == summary.TotalCredit
	return summary, nil
}
