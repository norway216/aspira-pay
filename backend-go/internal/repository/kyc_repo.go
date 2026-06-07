package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/kyc"
)

// CreateKYCProfile inserts a new KYC profile.
func (db *DB) CreateKYCProfile(p *kyc.Profile) error {
	// Handle empty date_of_birth — PostgreSQL doesn't accept empty string as DATE
	var dob interface{}
	if p.DateOfBirth == "" {
		dob = nil
	} else {
		dob = p.DateOfBirth
	}

	query := `
		INSERT INTO kyc_profiles (user_id, full_name, nationality, date_of_birth, document_type,
			document_number_hash, document_hash, address_hash, kyc_status, risk_level)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, submitted_at, created_at, updated_at`
	return db.QueryRow(query,
		p.UserID, p.FullName, p.Nationality, dob, p.DocumentType,
		p.DocumentNumberHash, p.DocumentHash, p.AddressHash,
		p.KYCStatus, p.RiskLevel,
	).Scan(&p.ID, &p.SubmittedAt, &p.CreatedAt, &p.UpdatedAt)
}

// GetKYCProfile retrieves a KYC profile by user_id.
func (db *DB) GetKYCProfile(userID string) (*kyc.Profile, error) {
	p := &kyc.Profile{}
	var dob sql.NullString
	query := `SELECT id, user_id, full_name, nationality, date_of_birth, document_type,
		document_number_hash, document_hash, address_hash, kyc_status, risk_level,
		COALESCE(rejection_reason, ''), COALESCE(reviewed_by, ''), reviewed_at, submitted_at, created_at, updated_at
		FROM kyc_profiles WHERE user_id = $1`
	err := db.QueryRow(query, userID).Scan(
		&p.ID, &p.UserID, &p.FullName, &p.Nationality, &dob, &p.DocumentType,
		&p.DocumentNumberHash, &p.DocumentHash, &p.AddressHash,
		&p.KYCStatus, &p.RiskLevel,
		&p.RejectionReason, &p.ReviewedBy, &p.ReviewedAt,
		&p.SubmittedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if dob.Valid {
		p.DateOfBirth = dob.String
	}
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("KYC profile not found for user: %s", userID)
	}
	return p, err
}

// UpdateKYCStatus updates KYC status with transition validation.
func (db *DB) UpdateKYCStatus(userID string, newStatus kyc.KYCStatus, reviewerID, reason string) error {
	current, err := db.GetKYCProfile(userID)
	if err != nil {
		return err
	}
	if !kyc.CanTransition(current.KYCStatus, newStatus) {
		return fmt.Errorf("invalid KYC transition: %s -> %s", current.KYCStatus, newStatus)
	}

	now := time.Now()
	query := `UPDATE kyc_profiles
		SET kyc_status = $1, reviewed_by = $2, rejection_reason = $3, reviewed_at = $4, updated_at = now()
		WHERE user_id = $5`
	_, err = db.Exec(query, newStatus, reviewerID, reason, now, userID)
	return err
}

// UpdateKYCRiskLevel updates the risk level from KYC.
func (db *DB) UpdateKYCRiskLevel(userID string, riskLevel kyc.KYCRiskLevel) error {
	query := `UPDATE kyc_profiles SET risk_level = $1, updated_at = now() WHERE user_id = $2`
	_, err := db.Exec(query, riskLevel, userID)
	return err
}

// ListKYCPending returns profiles awaiting review.
func (db *DB) ListKYCPending(page, pageSize int) ([]kyc.Profile, int64, error) {
	var total int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM kyc_profiles WHERE kyc_status IN ('PENDING', 'MANUAL_REVIEW')`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	query := `SELECT id, user_id, full_name, nationality, date_of_birth, document_type,
		document_number_hash, document_hash, address_hash, kyc_status, risk_level,
		COALESCE(rejection_reason, ''), COALESCE(reviewed_by, ''), reviewed_at, submitted_at, created_at, updated_at
		FROM kyc_profiles WHERE kyc_status IN ('PENDING', 'MANUAL_REVIEW')
		ORDER BY submitted_at ASC LIMIT $1 OFFSET $2`
	rows, err := db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var profiles []kyc.Profile
	for rows.Next() {
		var p kyc.Profile
		var dob sql.NullString
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.FullName, &p.Nationality, &dob, &p.DocumentType,
			&p.DocumentNumberHash, &p.DocumentHash, &p.AddressHash,
			&p.KYCStatus, &p.RiskLevel,
			&p.RejectionReason, &p.ReviewedBy, &p.ReviewedAt,
			&p.SubmittedAt, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if dob.Valid {
			p.DateOfBirth = dob.String
		}
		profiles = append(profiles, p)
	}
	return profiles, total, nil
}
