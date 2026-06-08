package service

import (
	"github.com/aspira/aspira-pay/internal/domain/kyc"
	"github.com/aspira/aspira-pay/internal/domain/payment"
	"github.com/aspira/aspira-pay/internal/domain/risk"
	"github.com/aspira/aspira-pay/internal/domain/user"
	"github.com/aspira/aspira-pay/internal/repository"
)

// RiskService evaluates AML/risk rules for payment transactions.
type RiskService struct {
	db    *repository.DB
	rules []risk.RiskRule
	// Cached context for the current assessment (set by AssessPaymentWithContext)
	sender         *user.User
	receiver       *user.User
	kycProfile     *kyc.Profile
	sourceAmount   int64
	countryFrom    string
	countryTo      string
}

// NewRiskService creates a new RiskService with default rules.
func NewRiskService(db *repository.DB) *RiskService {
	return &RiskService{
		db:    db,
		rules: risk.DefaultRules(),
	}
}

// AssessPayment evaluates all risk rules for a payment request.
// Deprecated: prefer AssessPaymentWithContext to avoid redundant DB lookups.
// Kept for backward compatibility with tests and external callers.
func (s *RiskService) AssessPayment(req payment.CreateRequest) (*risk.RiskResult, error) {
	return s.assessInternal(req.SenderUserID, req.ReceiverUserID,
		req.SourceAmount, req.CountryFrom, req.CountryTo,
		nil, nil, nil)
}

// AssessPaymentWithContext evaluates risk rules using pre-fetched data.
// This avoids redundant DB queries when the caller already has user/KYC objects.
// The caller (PaymentService.CreatePayment) already fetches sender and KYC for
// its own validation — passing them here eliminates 5+ redundant DB round-trips.
func (s *RiskService) AssessPaymentWithContext(
	req payment.CreateRequest,
	sender *user.User,
	senderKYC *kyc.Profile,
	receiver *user.User,
) (*risk.RiskResult, error) {
	return s.assessInternal(req.SenderUserID, req.ReceiverUserID,
		req.SourceAmount, req.CountryFrom, req.CountryTo,
		sender, senderKYC, receiver)
}

// assessInternal is the shared implementation. If sender/kyc/receiver are nil,
// they are fetched from DB (backward-compatible fallback).
func (s *RiskService) assessInternal(
	senderID, receiverID string,
	amount int64, countryFrom, countryTo string,
	sender *user.User, kycProfile *kyc.Profile, receiver *user.User,
) (*risk.RiskResult, error) {
	// Store context for check closures
	s.sender = sender
	s.receiver = receiver
	s.kycProfile = kycProfile
	s.sourceAmount = amount
	s.countryFrom = countryFrom
	s.countryTo = countryTo

	// Fetch receiver on demand if not provided
	if s.receiver == nil {
		rcv, err := s.db.GetUserByID(receiverID)
		if err == nil {
			s.receiver = rcv
		}
	}

	result := &risk.RiskResult{
		Decision: risk.RiskPass,
		Score:    0,
		Reasons:  []string{},
	}

	checks := []struct {
		name string
		fn   func() (bool, string, int)
	}{
		{"KYC_NOT_COMPLETED", s.checkKYCCompleted},
		{"USER_FROZEN", s.checkUserFrozen},
		{"AMOUNT_EXCEEDS_LIMIT", s.checkAmountLimit},
		{"DAILY_LIMIT_EXCEEDED", s.checkDailyLimit},
		{"RESTRICTED_COUNTRY", s.checkRestrictedCountry},
		{"HIGH_FREQUENCY", s.checkHighFrequency},
		{"NEW_USER_LARGE_AMOUNT", s.checkNewUserLargeAmount},
		{"SANCTIONED_COUNTRY", s.checkSanctionedCountry},
	}

	for _, check := range checks {
		triggered, reason, score := check.fn()
		if triggered {
			result.Score += score
			result.Reasons = append(result.Reasons, reason)

			for _, r := range s.rules {
				if r.Name == check.name && r.Blocking {
					result.Decision = risk.RiskReject
					return result, nil
				}
			}
		}
	}

	if result.Score >= 100 {
		result.Decision = risk.RiskReject
	} else if result.Score >= 50 {
		result.Decision = risk.RiskReview
	}

	return result, nil
}

// ── Check implementations — use pre-fetched data when available ──

func (s *RiskService) checkKYCCompleted() (bool, string, int) {
	// Use pre-fetched data if available (from AssessPaymentWithContext)
	if s.sender != nil && s.kycProfile != nil {
		if s.sender.Status != user.UserActive {
			return true, "Sender has not completed KYC", 100
		}
		if s.kycProfile.KYCStatus != kyc.KYCApproved {
			return true, "Sender KYC not approved", 100
		}
		return false, "", 0
	}

	// Fallback: fetch from DB (backward-compatible)
	u, err := s.db.GetUserByID(s.senderUserID())
	if err != nil || u.Status != user.UserActive {
		return true, "Sender has not completed KYC", 100
	}
	kycProfile, err := s.db.GetKYCProfile(s.senderUserID())
	if err != nil || kycProfile.KYCStatus != kyc.KYCApproved {
		return true, "Sender KYC not approved", 100
	}
	return false, "", 0
}

func (s *RiskService) checkUserFrozen() (bool, string, int) {
	// Use pre-fetched sender
	if s.sender != nil {
		if s.sender.IsFrozen() {
			return true, "Sender account is frozen", 100
		}
	} else {
		u, err := s.db.GetUserByID(s.senderUserID())
		if err != nil {
			return true, "Sender user not found", 100
		}
		if u.IsFrozen() {
			return true, "Sender account is frozen", 100
		}
	}

	// Use pre-fetched receiver
	if s.receiver != nil {
		if s.receiver.IsFrozen() {
			return true, "Receiver account is frozen", 100
		}
	} else {
		receiver, err := s.db.GetUserByID(s.receiverUserID())
		if err != nil {
			return true, "Receiver user not found", 100
		}
		if receiver.IsFrozen() {
			return true, "Receiver account is frozen", 100
		}
	}

	return false, "", 0
}

func (s *RiskService) checkAmountLimit() (bool, string, int) {
	riskLevel := "LOW"
	if s.sender != nil {
		riskLevel = s.sender.RiskLevel
	} else {
		u, _ := s.db.GetUserByID(s.senderUserID())
		if u != nil {
			riskLevel = u.RiskLevel
		}
	}

	limits := risk.LimitsByRiskLevel(riskLevel)
	if s.sourceAmount > limits.SingleTxLimit {
		return true, "Amount exceeds single transaction limit", 40
	}
	return false, "", 0
}

func (s *RiskService) checkDailyLimit() (bool, string, int) {
	dailyTotal, err := s.db.GetDailyTotal(s.senderUserID())
	if err != nil {
		return false, "", 0
	}

	riskLevel := "LOW"
	if s.sender != nil {
		riskLevel = s.sender.RiskLevel
	} else {
		u, _ := s.db.GetUserByID(s.senderUserID())
		if u != nil {
			riskLevel = u.RiskLevel
		}
	}
	limits := risk.LimitsByRiskLevel(riskLevel)

	if dailyTotal+s.sourceAmount > limits.DailyLimit {
		return true, "Daily transaction limit would be exceeded", 30
	}
	return false, "", 0
}

func (s *RiskService) checkRestrictedCountry() (bool, string, int) {
	restricted := risk.RestrictedCountries()
	if restricted[s.countryFrom] {
		return true, "Source country is restricted", 100
	}
	if restricted[s.countryTo] {
		return true, "Destination country is restricted", 100
	}
	return false, "", 0
}

func (s *RiskService) checkHighFrequency() (bool, string, int) {
	count, err := s.db.GetRecentTxCount(s.senderUserID(), 60)
	if err != nil {
		return false, "", 0
	}
	if count >= 10 {
		return true, "High frequency trading detected (>10 tx/min)", 50
	}
	return false, "", 0
}

func (s *RiskService) checkNewUserLargeAmount() (bool, string, int) {
	if s.sender != nil {
		if s.sender.CreatedAt.Unix() > 0 && s.sourceAmount > 100000 {
			return true, "New user making large transaction", 60
		}
		return false, "", 0
	}

	u, err := s.db.GetUserByID(s.senderUserID())
	if err != nil {
		return false, "", 0
	}
	if u.CreatedAt.Unix() > 0 && s.sourceAmount > 100000 {
		return true, "New user making large transaction", 60
	}
	return false, "", 0
}

func (s *RiskService) checkSanctionedCountry() (bool, string, int) {
	restricted := risk.RestrictedCountries()
	if restricted[s.countryFrom] || restricted[s.countryTo] {
		return true, "Country under international sanctions", 100
	}
	return false, "", 0
}

// senderUserID returns the sender ID from the stored context state.
func (s *RiskService) senderUserID() string {
	if s.sender != nil {
		return s.sender.UserID
	}
	return ""
}

// receiverUserID returns the receiver ID from the stored context state.
func (s *RiskService) receiverUserID() string {
	if s.receiver != nil {
		return s.receiver.UserID
	}
	return ""
}
