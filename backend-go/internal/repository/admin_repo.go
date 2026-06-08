package repository

import (
	"encoding/json"
	"time"
)

// AdminAuditEntry represents a recorded admin action (§14.2).
type AdminAuditEntry struct {
	ID         int64     `json:"-"`
	AdminID    string    `json:"admin_id"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Details    string    `json:"details,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (db *DB) InsertAdminAudit(adminID, action, targetType, targetID, ip string, details map[string]interface{}) error {
	var detJSON []byte
	if details != nil {
		detJSON, _ = json.Marshal(details)
	}
	_, err := db.Exec(`INSERT INTO admin_audit_logs (admin_id, action, target_type, target_id, details, ip_address)
		VALUES ($1,$2,$3,$4,$5,$6)`, adminID, action, targetType, targetID, string(detJSON), ip)
	return err
}

func (db *DB) ListAdminAuditLogs(adminID, targetType string, page, pageSize int) ([]AdminAuditEntry, int64, error) {
	var total int64
	db.QueryRow(`SELECT COUNT(*) FROM admin_audit_logs WHERE ($1='' OR admin_id=$1) AND ($2='' OR target_type=$2)`, adminID, targetType).Scan(&total)

	offset := (page - 1) * pageSize
	if offset < 0 { offset = 0 }
	query := `SELECT id, admin_id, action, target_type, target_id, COALESCE(details::text,''), COALESCE(ip_address,''), created_at
		FROM admin_audit_logs WHERE ($1='' OR admin_id=$1) AND ($2='' OR target_type=$2)
		ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	rows, err := db.Query(query, adminID, targetType, pageSize, offset)
	if err != nil { return nil, 0, err }
	defer rows.Close()

	var entries []AdminAuditEntry
	for rows.Next() {
		var e AdminAuditEntry
		if err := rows.Scan(&e.ID, &e.AdminID, &e.Action, &e.TargetType, &e.TargetID, &e.Details, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, nil
}

// ── Login Rate Limiting (§5.3) ──────────────────

func (db *DB) RecordLoginAttempt(username, ip string, success bool) {
	db.Exec(`INSERT INTO login_attempts (username, ip_address, success) VALUES ($1,$2,$3)`, username, ip, success)
}

func (db *DB) IncrementFailedLoginCount(username string) (int, error) {
	var count int
	db.QueryRow(`UPDATE users SET failed_login_count = failed_login_count + 1 WHERE username = $1 RETURNING failed_login_count`, username).Scan(&count)
	return count, nil
}

func (db *DB) ResetFailedLoginCount(username string) {
	db.Exec(`UPDATE users SET failed_login_count = 0, locked_until = NULL WHERE username = $1`, username)
}

func (db *DB) LockUserUntil(username string, until time.Time) {
	db.Exec(`UPDATE users SET locked_until = $1 WHERE username = $2`, until, username)
}

func (db *DB) IsUserLocked(username string) bool {
	var lockedUntil *time.Time
	db.QueryRow(`SELECT locked_until FROM users WHERE username = $1`, username).Scan(&lockedUntil)
	return lockedUntil != nil && lockedUntil.After(time.Now())
}

// ── Card Application Review (§7.2) ──────────────

func (db *DB) ReviewCardApplication(cardID, decision, reviewer, notes string) error {
	_, err := db.Exec(`UPDATE cards SET application_status=$1, reviewed_by=$2, review_notes=$3,
		status = CASE WHEN $1 = 'REJECTED' THEN 'CANCELLED' ELSE status END,
		updated_at=now() WHERE card_id=$4`, decision, reviewer, notes, cardID)
	return err
}

func (db *DB) ListPendingCardApplications(page, pageSize int) ([]map[string]interface{}, int64, error) {
	var total int64
	db.QueryRow(`SELECT COUNT(*) FROM cards WHERE application_status = 'PENDING'`).Scan(&total)
	offset := (page - 1) * pageSize
	if offset < 0 { offset = 0 }
	rows, err := db.Query(`SELECT card_id, owner_id, card_network, default_currency,
		COALESCE(kyc_full_name,''), COALESCE(kyc_nationality,''), COALESCE(kyc_date_of_birth,''),
		COALESCE(kyc_address,''), COALESCE(kyc_document_type,''), status, created_at
		FROM cards WHERE application_status = 'PENDING' ORDER BY created_at ASC LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil { return nil, 0, err }
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var cardID, ownerID, network, currency, fullName, nationality, dob, address, docType, status string
		var createdAt time.Time
		rows.Scan(&cardID, &ownerID, &network, &currency, &fullName, &nationality, &dob, &address, &docType, &status, &createdAt)
		results = append(results, map[string]interface{}{
			"card_id": cardID, "owner_id": ownerID, "card_network": network,
			"default_currency": currency, "full_name": fullName, "nationality": nationality,
			"date_of_birth": dob, "address": address, "document_type": docType,
			"status": status, "created_at": createdAt,
		})
	}
	return results, total, nil
}

// ── External Card Binding (§7.1) ────────────────

func (db *DB) BindExternalCard(userID, cardID, cardToken, cardholderName, last4, network, cardType, country string, expiryMonth, expiryYear int) error {
	_, err := db.Exec(`INSERT INTO external_cards (card_id, user_id, card_token, cardholder_name, last4, card_network, card_type, issuing_country, expiry_month, expiry_year)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, cardID, userID, cardToken, cardholderName, last4, network, cardType, country, expiryMonth, expiryYear)
	return err
}

func (db *DB) ListExternalCards(userID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(`SELECT card_id, cardholder_name, last4, card_network, card_type, issuing_country, expiry_month, expiry_year, is_default, status, created_at
		FROM external_cards WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil { return nil, err }
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var cardID, name, last4, network, cardType, country, status string
		var expMonth, expYear int
		var isDefault bool
		var createdAt time.Time
		rows.Scan(&cardID, &name, &last4, &network, &cardType, &country, &expMonth, &expYear, &isDefault, &status, &createdAt)
		results = append(results, map[string]interface{}{
			"card_id": cardID, "cardholder_name": name, "last4": last4, "card_network": network,
			"card_type": cardType, "issuing_country": country, "expiry_month": expMonth,
			"expiry_year": expYear, "is_default": isDefault, "status": status, "created_at": createdAt,
		})
	}
	return results, nil
}
