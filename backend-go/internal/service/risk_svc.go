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
	// Temporary request context for check function closures during assessment
	senderUserID   string
	receiverUserID string
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
func (s *RiskService) AssessPayment(req payment.CreateRequest) (*risk.RiskResult, error) {
	// Store request data temporarily for use by check closures
	s.senderUserID = req.SenderUserID
	s.receiverUserID = req.ReceiverUserID
	s.sourceAmount = req.SourceAmount
	s.countryFrom = req.CountryFrom
	s.countryTo = req.CountryTo

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

			// Find the rule to check if it's blocking
			for _, r := range s.rules {
				if r.Name == check.name && r.Blocking {
					result.Decision = risk.RiskReject
					return result, nil
				}
			}
		}
	}

	// Determine final decision
	if result.Score >= 100 {
		result.Decision = risk.RiskReject
	} else if result.Score >= 50 {
		result.Decision = risk.RiskReview
	}

	return result, nil
}

func (s *RiskService) checkKYCCompleted() (bool, string, int) {
	u, err := s.db.GetUserByID(s.senderUserID)
	if err != nil || u.Status != user.UserActive {
		return true, "Sender has not completed KYC", 100
	}

	kycProfile, err := s.db.GetKYCProfile(s.senderUserID)
	if err != nil || kycProfile.KYCStatus != kyc.KYCApproved {
		return true, "Sender KYC not approved", 100
	}

	return false, "", 0
}

func (s *RiskService) checkUserFrozen() (bool, string, int) {
	u, err := s.db.GetUserByID(s.senderUserID)
	if err != nil {
		return true, "Sender user not found", 100
	}
	if u.IsFrozen() {
		return true, "Sender account is frozen", 100
	}

	receiver, err := s.db.GetUserByID(s.receiverUserID)
	if err != nil {
		return true, "Receiver user not found", 100
	}
	if receiver.IsFrozen() {
		return true, "Receiver account is frozen", 100
	}

	return false, "", 0
}

func (s *RiskService) checkAmountLimit() (bool, string, int) {
	u, _ := s.db.GetUserByID(s.senderUserID)
	riskLevel := "LOW"
	if u != nil {
		riskLevel = u.RiskLevel
	}

	limits := risk.LimitsByRiskLevel(riskLevel)
	if s.sourceAmount > limits.SingleTxLimit {
		return true, "Amount exceeds single transaction limit", 40
	}
	return false, "", 0
}

func (s *RiskService) checkDailyLimit() (bool, string, int) {
	dailyTotal, err := s.db.GetDailyTotal(s.senderUserID)
	if err != nil {
		return false, "", 0 // Don't block on DB error
	}

	u, _ := s.db.GetUserByID(s.senderUserID)
	riskLevel := "LOW"
	if u != nil {
		riskLevel = u.RiskLevel
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
	count, err := s.db.GetRecentTxCount(s.senderUserID, 60)
	if err != nil {
		return false, "", 0
	}
	if count >= 10 {
		return true, "High frequency trading detected (>10 tx/min)", 50
	}
	return false, "", 0
}

func (s *RiskService) checkNewUserLargeAmount() (bool, string, int) {
	u, err := s.db.GetUserByID(s.senderUserID)
	if err != nil {
		return false, "", 0
	}

	// In production: check if user was created within last 24 hours
	if u.CreatedAt.Unix() > 0 && s.sourceAmount > 100000 { // $1,000
		return true, "New user making large transaction", 60
	}
	return false, "", 0
}

func (s *RiskService) checkSanctionedCountry() (bool, string, int) {
	// Same as restricted countries in Sandbox
	restricted := risk.RestrictedCountries()
	if restricted[s.countryFrom] || restricted[s.countryTo] {
		return true, "Country under international sanctions", 100
	}
	return false, "", 0
}
