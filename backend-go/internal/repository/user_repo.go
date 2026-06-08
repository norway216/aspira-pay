package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/aspira/aspira-pay/internal/domain/user"
)

// scanUser scans a user row with all V2 fields including KYC and admin_key.
func scanUser(row interface{ Scan(...interface{}) error }) (*user.User, error) {
	u := &user.User{}
	var phone, adminKey, pin, fullName, nationality, dob sql.NullString
	err := row.Scan(
		&u.ID, &u.UserID, &u.Username, &u.Email, &u.PasswordHash, &phone,
		&u.Status, &u.RiskLevel, &u.DefaultCurrency, &adminKey, &pin,
		&fullName, &nationality, &dob, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil { return nil, err }
	u.Phone = phone.String
	u.AdminKey = adminKey.String
	u.TransactionPIN = pin.String
	u.HasPIN = pin.Valid && pin.String != ""
	u.FullName = fullName.String
	u.Nationality = nationality.String
	u.DateOfBirth = dob.String
	return u, nil
}

var userColumns = `id, user_id, username, email, password_hash, COALESCE(phone,''),
	status, COALESCE(risk_level,'LOW'), COALESCE(default_currency,'USD'),
	COALESCE(admin_key,''), COALESCE(transaction_pin,''),
	COALESCE(full_name,''), COALESCE(nationality,''), COALESCE(date_of_birth,''),
	created_at, updated_at`

func (db *DB) CreateUser(u *user.User) error {
	if u.DefaultCurrency == "" { u.DefaultCurrency = "USD" }
	query := `INSERT INTO users (user_id, username, email, password_hash, phone, status, risk_level, default_currency,
		admin_key, full_name, nationality, date_of_birth)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id, created_at, updated_at`
	return db.QueryRow(query,
		u.UserID, u.Username, u.Email, u.PasswordHash, u.Phone,
		u.Status, u.RiskLevel, u.DefaultCurrency,
		u.AdminKey, u.FullName, u.Nationality, u.DateOfBirth,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (db *DB) GetUserByID(userID string) (*user.User, error) {
	u, err := scanUser(db.QueryRow(`SELECT `+userColumns+` FROM users WHERE user_id = $1`, userID))
	if err == sql.ErrNoRows { return nil, fmt.Errorf("user not found: %s", userID) }
	return u, err
}

func (db *DB) GetUserByUsername(username string) (*user.User, error) {
	u, err := scanUser(db.QueryRow(`SELECT `+userColumns+` FROM users WHERE username = $1`, username))
	if err == sql.ErrNoRows { return nil, fmt.Errorf("user not found: %s", username) }
	return u, err
}

func (db *DB) GetUserByEmail(email string) (*user.User, error) {
	u, err := scanUser(db.QueryRow(`SELECT `+userColumns+` FROM users WHERE email = $1`, email))
	if err == sql.ErrNoRows { return nil, fmt.Errorf("user not found by email: %s", email) }
	return u, err
}

func (db *DB) UpdateUserStatus(userID string, newStatus user.UserStatus) error {
	var validFrom []user.UserStatus
	for from, tos := range user.ValidTransitions {
		for _, to := range tos {
			if to == newStatus { validFrom = append(validFrom, from) }
		}
	}
	if len(validFrom) == 0 { return fmt.Errorf("no valid transition to %s", newStatus) }

	phs := make([]string, len(validFrom))
	args := []interface{}{newStatus, userID}
	for i, s := range validFrom {
		phs[i] = fmt.Sprintf("$%d", i+3)
		args = append(args, s)
	}
	query := fmt.Sprintf(`UPDATE users SET status=$1, updated_at=now() WHERE user_id=$2 AND status IN (%s)`, strings.Join(phs, ","))
	result, _ := db.Exec(query, args...)
	rows, _ := result.RowsAffected()
	if rows == 0 { return fmt.Errorf("invalid user status transition to %s", newStatus) }
	return nil
}

func (db *DB) UpdateUserPIN(userID, pinHash string) error {
	_, err := db.Exec(`UPDATE users SET transaction_pin=$1, updated_at=now() WHERE user_id=$2`, pinHash, userID)
	return err
}

func (db *DB) UpdateUserRiskLevel(userID, riskLevel string) error {
	_, err := db.Exec(`UPDATE users SET risk_level=$1, updated_at=now() WHERE user_id=$2`, riskLevel, userID)
	return err
}

func (db *DB) ListUsers(page, pageSize int) ([]user.User, int64, error) {
	var total int64
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&total)
	offset := (page - 1) * pageSize
	if offset < 0 { offset = 0 }
	query := `SELECT ` + userColumns + ` FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := db.Query(query, pageSize, offset)
	if err != nil { return nil, 0, err }
	defer rows.Close()

	var users []user.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil { return nil, 0, err }
		users = append(users, *u)
	}
	return users, total, nil
}

// CountCardsByUser returns the number of active cards for a user.
func (db *DB) CountCardsByUser(userID string) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM cards WHERE owner_id = $1 AND status != 'CANCELLED'`, userID).Scan(&count)
	return count, err
}
