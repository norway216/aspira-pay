package service

import (
	"fmt"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/kyc"
	"github.com/aspira/aspira-pay/internal/domain/user"
	"github.com/aspira/aspira-pay/internal/repository"
	pkgerrors "github.com/aspira/aspira-pay/pkg/errors"
	"github.com/aspira/aspira-pay/pkg/crypto"
)

// KYCService handles KYC verification workflow.
type KYCService struct {
	db *repository.DB
}

// NewKYCService creates a new KYCService.
func NewKYCService(db *repository.DB) *KYCService {
	return &KYCService{db: db}
}

// SubmitKYC submits a KYC profile for review.
func (s *KYCService) SubmitKYC(userID string, req kyc.SubmitRequest) (*kyc.Profile, error) {
	// Check user exists
	_, err := s.db.GetUserByID(userID)
	if err != nil {
		return nil, pkgerrors.NotFound("user not found")
	}

	// Check if KYC already exists
	if existing, _ := s.db.GetKYCProfile(userID); existing != nil {
		if existing.KYCStatus == kyc.KYCApproved {
			return nil, pkgerrors.Conflict("KYC already approved")
		}
		if existing.KYCStatus == kyc.KYCPending || existing.KYCStatus == kyc.KYCReviewing {
			return nil, pkgerrors.Conflict("KYC already submitted and pending review")
		}
		// Rejected or Expired — allow resubmit
	}

	// Hash sensitive fields before storage
	profile := &kyc.Profile{
		UserID:             userID,
		FullName:           req.FullName,
		Nationality:        req.Nationality,
		DateOfBirth:        req.DateOfBirth,
		DocumentType:       req.DocumentType,
		DocumentNumberHash: crypto.SHA256("doc:" + req.DocumentNumber),
		DocumentHash:       crypto.SHA256("docfile:" + req.DocumentNumber),
		AddressHash:        crypto.SHA256("addr:" + req.Address),
		KYCStatus:          kyc.KYCPending,
		RiskLevel:          kyc.RiskLow,
		SubmittedAt:        time.Now(),
	}

	if err := s.db.CreateKYCProfile(profile); err != nil {
		return nil, fmt.Errorf("cannot create KYC profile: %w", err)
	}

	// Auto-transition to MANUAL_REVIEW for Sandbox
	if err := s.db.UpdateKYCStatus(userID, kyc.KYCReviewing, "system", ""); err != nil {
		return nil, fmt.Errorf("cannot update KYC status: %w", err)
	}
	profile.KYCStatus = kyc.KYCReviewing

	return profile, nil
}

// GetKYCStatus retrieves the KYC profile for a user.
func (s *KYCService) GetKYCStatus(userID string) (*kyc.Profile, error) {
	return s.db.GetKYCProfile(userID)
}

// ReviewKYC processes a KYC review decision (admin action).
func (s *KYCService) ReviewKYC(reviewerID string, req kyc.ReviewRequest) error {
	profile, err := s.db.GetKYCProfile(req.UserID)
	if err != nil {
		return pkgerrors.NotFound("KYC profile not found")
	}

	if req.Action != kyc.KYCApproved && req.Action != kyc.KYCRejected {
		return pkgerrors.InvalidInput("action must be APPROVED or REJECTED")
	}

	if err := s.db.UpdateKYCStatus(req.UserID, req.Action, reviewerID, req.Reason); err != nil {
		return fmt.Errorf("cannot update KYC status: %w", err)
	}

	// Update user status based on KYC decision
	if req.Action == kyc.KYCApproved {
		if err := s.db.UpdateUserStatus(req.UserID, user.UserActive); err != nil {
			return fmt.Errorf("cannot activate user: %w", err)
		}
		// Apply risk level if specified
		if req.RiskLevel != "" {
			if err := s.db.UpdateKYCRiskLevel(req.UserID, req.RiskLevel); err != nil {
				return fmt.Errorf("cannot update risk level: %w", err)
			}
			if err := s.db.UpdateUserRiskLevel(req.UserID, string(req.RiskLevel)); err != nil {
				return fmt.Errorf("cannot update user risk level: %w", err)
			}
		}
	} else {
		if err := s.db.UpdateUserStatus(req.UserID, user.UserRejected); err != nil {
			return fmt.Errorf("cannot reject user: %w", err)
		}
	}

	_ = profile // Used for audit
	return nil
}

// ListPendingReviews returns KYC profiles awaiting manual review.
func (s *KYCService) ListPendingReviews(page, pageSize int) ([]kyc.Profile, int64, error) {
	return s.db.ListKYCPending(page, pageSize)
}
