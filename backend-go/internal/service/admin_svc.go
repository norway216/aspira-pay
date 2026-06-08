package service

import (
	"fmt"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/kyc"
	"github.com/aspira/aspira-pay/internal/repository"
)

// AdminService handles admin operations with audit logging (§14.2).
type AdminService struct {
	db *repository.DB
}

func NewAdminService(db *repository.DB) *AdminService {
	return &AdminService{db: db}
}

// AuditLog records an admin action for compliance.
func (s *AdminService) AuditLog(adminID, action, targetType, targetID, ip string, details map[string]interface{}) {
	s.db.InsertAdminAudit(adminID, action, targetType, targetID, ip, details)
}

// ReviewCardApplication approves or rejects a card application (§7.2).
func (s *AdminService) ReviewCardApplication(adminID, cardID, decision, notes, ip string) error {
	if decision != "APPROVED" && decision != "REJECTED" {
		return fmt.Errorf("decision must be APPROVED or REJECTED")
	}
	if err := s.db.ReviewCardApplication(cardID, decision, adminID, notes); err != nil {
		return err
	}
	s.AuditLog(adminID, "REVIEW_CARD", "card", cardID, ip, map[string]interface{}{
		"decision": decision, "notes": notes,
	})
	return nil
}

// ReviewKYC approves or rejects a KYC submission (§6.1).
func (s *AdminService) ReviewKYC(adminID, userID, decision, reason, ip string) error {
	if err := s.db.UpdateKYCStatus(userID, kyc.KYCStatus(decision), adminID, reason); err != nil {
		return err
	}
	s.AuditLog(adminID, "REVIEW_KYC", "user", userID, ip, map[string]interface{}{
		"decision": decision, "reason": reason,
	})
	return nil
}

// GetAuditLogs returns admin audit logs with filters.
func (s *AdminService) GetAuditLogs(adminID, targetType string, page, pageSize int) ([]repository.AdminAuditEntry, int64, error) {
	return s.db.ListAdminAuditLogs(adminID, targetType, page, pageSize)
}

// RecordLoginAttempt records a login attempt for rate limiting (§5.3).
// Returns true if the account is locked due to too many failures.
func (s *AdminService) RecordLoginAttempt(username, ip string, success bool) (bool, error) {
	s.db.RecordLoginAttempt(username, ip, success)
	if success {
		s.db.ResetFailedLoginCount(username)
		return false, nil
	}
	count, _ := s.db.IncrementFailedLoginCount(username)
	if count >= 5 {
		// Lock account for 15 minutes
		s.db.LockUserUntil(username, time.Now().Add(15*time.Minute))
		return true, nil
	}
	return false, nil
}

// IsUserLocked checks if a user account is locked.
func (s *AdminService) IsUserLocked(username string) bool {
	return s.db.IsUserLocked(username)
}

// PendingApps returns card applications waiting for admin review.
func (s *AdminService) PendingApps(page, pageSize int) ([]map[string]interface{}, int64, error) {
	return s.db.ListPendingCardApplications(page, pageSize)
}
