package repository

import (
	"database/sql"
	"fmt"

	"github.com/aspira/aspira-pay/internal/domain/user"
)

// CreateUser inserts a new user record.
func (db *DB) CreateUser(u *user.User) error {
	query := `
		INSERT INTO users (user_id, username, email, password_hash, phone, status, risk_level, default_currency)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	if u.DefaultCurrency == "" {
		u.DefaultCurrency = "USD"
	}
	return db.QueryRow(query,
		u.UserID, u.Username, u.Email, u.PasswordHash, u.Phone,
		u.Status, u.RiskLevel, u.DefaultCurrency,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

// GetUserByID retrieves a user by user_id.
func (db *DB) GetUserByID(userID string) (*user.User, error) {
	u := &user.User{}
	query := `SELECT id, user_id, username, email, password_hash, phone, status, risk_level, COALESCE(default_currency,'USD'), created_at, updated_at
		FROM users WHERE user_id = $1`
	err := db.QueryRow(query, userID).Scan(
		&u.ID, &u.UserID, &u.Username, &u.Email, &u.PasswordHash, &u.Phone,
		&u.Status, &u.RiskLevel, &u.DefaultCurrency, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	return u, err
}

// GetUserByUsername retrieves a user by username.
func (db *DB) GetUserByUsername(username string) (*user.User, error) {
	u := &user.User{}
	query := `SELECT id, user_id, username, email, password_hash, phone, status, risk_level, COALESCE(default_currency,'USD'), created_at, updated_at
		FROM users WHERE username = $1`
	err := db.QueryRow(query, username).Scan(
		&u.ID, &u.UserID, &u.Username, &u.Email, &u.PasswordHash, &u.Phone,
		&u.Status, &u.RiskLevel, &u.DefaultCurrency, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	return u, err
}

// GetUserByEmail retrieves a user by email.
func (db *DB) GetUserByEmail(email string) (*user.User, error) {
	u := &user.User{}
	query := `SELECT id, user_id, username, email, password_hash, phone, status, risk_level, COALESCE(default_currency,'USD'), created_at, updated_at
		FROM users WHERE email = $1`
	err := db.QueryRow(query, email).Scan(
		&u.ID, &u.UserID, &u.Username, &u.Email, &u.PasswordHash, &u.Phone,
		&u.Status, &u.RiskLevel, &u.DefaultCurrency, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found by email: %s", email)
	}
	return u, err
}

// UpdateUserStatus updates user status with transition validation.
func (db *DB) UpdateUserStatus(userID string, newStatus user.UserStatus) error {
	current, err := db.GetUserByID(userID)
	if err != nil {
		return err
	}
	if !user.CanTransition(current.Status, newStatus) {
		return fmt.Errorf("invalid user status transition: %s -> %s", current.Status, newStatus)
	}

	query := `UPDATE users SET status = $1, updated_at = now() WHERE user_id = $2`
	result, err := db.Exec(query, newStatus, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}
	return nil
}

// UpdateUserRiskLevel updates the user's risk level.
func (db *DB) UpdateUserRiskLevel(userID, riskLevel string) error {
	query := `UPDATE users SET risk_level = $1, updated_at = now() WHERE user_id = $2`
	_, err := db.Exec(query, riskLevel, userID)
	return err
}

// ListUsers returns a paginated list of users.
func (db *DB) ListUsers(page, pageSize int) ([]user.User, int64, error) {
	var total int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	query := `SELECT id, user_id, username, email, password_hash, phone, status, risk_level, COALESCE(default_currency,'USD'), created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []user.User
	for rows.Next() {
		var u user.User
		if err := rows.Scan(
			&u.ID, &u.UserID, &u.Username, &u.Email, &u.PasswordHash, &u.Phone,
			&u.Status, &u.RiskLevel, &u.DefaultCurrency, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}
