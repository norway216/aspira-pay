// Package kyc defines the KYC (Know Your Customer) domain model.
package kyc

import "time"

// KYCStatus represents the KYC verification state.
type KYCStatus string

const (
	KYCPending    KYCStatus = "PENDING"
	KYCReviewing  KYCStatus = "MANUAL_REVIEW"
	KYCApproved   KYCStatus = "APPROVED"
	KYCRejected   KYCStatus = "REJECTED"
	KYCExpired    KYCStatus = "EXPIRED"
)

// KYCRiskLevel represents the risk classification from KYC.
type KYCRiskLevel string

const (
	RiskLow    KYCRiskLevel = "LOW"
	RiskMedium KYCRiskLevel = "MEDIUM"
	RiskHigh   KYCRiskLevel = "HIGH"
)

// ValidKYCStatuses is the set of valid KYC statuses.
var ValidKYCStatuses = map[KYCStatus]bool{
	KYCPending:   true,
	KYCReviewing: true,
	KYCApproved:  true,
	KYCRejected:  true,
	KYCExpired:   true,
}

// ValidTransitions defines allowed KYC state transitions.
var ValidTransitions = map[KYCStatus][]KYCStatus{
	KYCPending:   {KYCReviewing, KYCApproved, KYCRejected},
	KYCReviewing: {KYCApproved, KYCRejected},
	KYCApproved:  {KYCExpired},
	KYCRejected:  {KYCPending}, // Can resubmit
	KYCExpired:   {KYCPending}, // Can re-verify
}

// CanTransition checks if a KYC status transition is valid.
func CanTransition(from, to KYCStatus) bool {
	for _, valid := range ValidTransitions[from] {
		if valid == to {
			return true
		}
	}
	return false
}

// Profile represents a user's KYC profile.
type Profile struct {
	ID                 int64        `json:"-"`
	UserID             string       `json:"user_id"`
	FullName           string       `json:"full_name"`
	Nationality        string       `json:"nationality,omitempty"`
	DateOfBirth        string       `json:"date_of_birth,omitempty"`
	DocumentType       string       `json:"document_type,omitempty"`
	DocumentNumberHash string       `json:"-"` // Hashed, never exposed
	DocumentHash       string       `json:"-"` // Hashed document file
	AddressHash        string       `json:"-"` // Hashed address
	KYCStatus          KYCStatus    `json:"kyc_status"`
	RiskLevel          KYCRiskLevel `json:"risk_level"`
	RejectionReason    string       `json:"rejection_reason,omitempty"`
	ReviewedBy         string       `json:"reviewed_by,omitempty"`
	ReviewedAt         *time.Time   `json:"reviewed_at,omitempty"`
	SubmittedAt        time.Time    `json:"submitted_at"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

// IsApproved checks if KYC is approved.
func (p *Profile) IsApproved() bool { return p.KYCStatus == KYCApproved }

// IsPending checks if KYC is pending.
func (p *Profile) IsPending() bool { return p.KYCStatus == KYCPending }

// SubmitRequest is the input for KYC submission.
type SubmitRequest struct {
	FullName    string `json:"full_name" binding:"required"`
	Nationality string `json:"nationality"`
	DateOfBirth string `json:"date_of_birth"`
	DocumentType string `json:"document_type"`
	DocumentNumber string `json:"document_number"` // Will be hashed before storage
	Address      string `json:"address"`            // Will be hashed before storage
}

// ReviewRequest is the input for KYC review (admin action).
type ReviewRequest struct {
	UserID  string    `json:"user_id" binding:"required"`
	Action  KYCStatus `json:"action" binding:"required"` // APPROVED or REJECTED
	Reason  string    `json:"reason,omitempty"`
	RiskLevel KYCRiskLevel `json:"risk_level,omitempty"`
}
