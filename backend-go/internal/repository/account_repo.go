package repository

import (
	"database/sql"
	"fmt"

	"github.com/aspira/aspira-pay/internal/domain/ledger"
)

// CreateAccount inserts a new account.
func (db *DB) CreateAccount(a *ledger.LegacyAccount) error {
	query := `
		INSERT INTO accounts (account_id, user_id, currency, available_balance, frozen_balance, settled_balance, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	return db.QueryRow(query,
		a.AccountID, a.UserID, a.Currency,
		a.AvailableBalance, a.FrozenBalance, a.SettledBalance,
		a.Status,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// GetAccount retrieves an account by account_id.
func (db *DB) GetAccount(accountID string) (*ledger.LegacyAccount, error) {
	a := &ledger.LegacyAccount{}
	query := `SELECT id, account_id, user_id, currency, available_balance, frozen_balance, settled_balance, status, created_at, updated_at
		FROM accounts WHERE account_id = $1`
	err := db.QueryRow(query, accountID).Scan(
		&a.ID, &a.AccountID, &a.UserID, &a.Currency,
		&a.AvailableBalance, &a.FrozenBalance, &a.SettledBalance,
		&a.Status, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}
	return a, err
}

// GetAccountsByUser retrieves all accounts for a user.
func (db *DB) GetAccountsByUser(userID string) ([]ledger.LegacyAccount, error) {
	query := `SELECT id, account_id, user_id, currency, available_balance, frozen_balance, settled_balance, status, created_at, updated_at
		FROM accounts WHERE user_id = $1 ORDER BY currency`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []ledger.LegacyAccount
	for rows.Next() {
		var a ledger.LegacyAccount
		if err := rows.Scan(
			&a.ID, &a.AccountID, &a.UserID, &a.Currency,
			&a.AvailableBalance, &a.FrozenBalance, &a.SettledBalance,
			&a.Status, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

// GetAccountByUserAndCurrency retrieves a specific currency account for a user.
func (db *DB) GetAccountByUserAndCurrency(userID, currency string) (*ledger.LegacyAccount, error) {
	a := &ledger.LegacyAccount{}
	query := `SELECT id, account_id, user_id, currency, available_balance, frozen_balance, settled_balance, status, created_at, updated_at
		FROM accounts WHERE user_id = $1 AND currency = $2`
	err := db.QueryRow(query, userID, currency).Scan(
		&a.ID, &a.AccountID, &a.UserID, &a.Currency,
		&a.AvailableBalance, &a.FrozenBalance, &a.SettledBalance,
		&a.Status, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found: user=%s currency=%s", userID, currency)
	}
	return a, err
}

// FreezeFunds moves available balance to frozen (within a transaction).
func (db *DB) FreezeFunds(accountID string, amount int64) error {
	query := `
		UPDATE accounts
		SET available_balance = available_balance - $1,
		    frozen_balance = frozen_balance + $1,
		    updated_at = now()
		WHERE account_id = $2 AND available_balance >= $1`
	result, err := db.Exec(query, amount, accountID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient funds for freeze: account=%s amount=%d", accountID, amount)
	}
	return nil
}

// UnfreezeFunds moves frozen balance back to available.
func (db *DB) UnfreezeFunds(accountID string, amount int64) error {
	query := `
		UPDATE accounts
		SET frozen_balance = frozen_balance - $1,
		    available_balance = available_balance + $1,
		    updated_at = now()
		WHERE account_id = $2 AND frozen_balance >= $1`
	result, err := db.Exec(query, amount, accountID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient frozen funds: account=%s amount=%d", accountID, amount)
	}
	return nil
}

// DebitAccount debits an account (reduces frozen, increases settled).
func (db *DB) DebitAccount(accountID string, amount int64) error {
	query := `
		UPDATE accounts
		SET frozen_balance = frozen_balance - $1,
		    settled_balance = settled_balance + $1,
		    updated_at = now()
		WHERE account_id = $2 AND frozen_balance >= $1`
	result, err := db.Exec(query, amount, accountID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient frozen funds for debit: account=%s amount=%d", accountID, amount)
	}
	return nil
}

// CreditAccount credits an account (increases available or settled).
func (db *DB) CreditAccount(accountID string, amount int64) error {
	query := `
		UPDATE accounts
		SET available_balance = available_balance + $1,
		    updated_at = now()
		WHERE account_id = $2`
	_, err := db.Exec(query, amount, accountID)
	return err
}

// AddAvailableBalance adds to available balance (for test deposits).
func (db *DB) AddAvailableBalance(accountID string, amount int64) error {
	query := `
		UPDATE accounts
		SET available_balance = available_balance + $1,
		    updated_at = now()
		WHERE account_id = $2`
	result, err := db.Exec(query, amount, accountID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("account not found: %s", accountID)
	}
	return nil
}
