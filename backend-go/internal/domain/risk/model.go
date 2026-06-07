// Package risk defines the Risk/AML domain model.
package risk

// RiskDecision is the outcome of a risk assessment.
type RiskDecision string

const (
	RiskPass   RiskDecision = "PASS"
	RiskReject RiskDecision = "REJECT"
	RiskReview RiskDecision = "MANUAL_REVIEW"
)

// RiskResult contains the detailed outcome of risk assessment.
type RiskResult struct {
	Decision RiskDecision `json:"decision"`
	Score    int          `json:"score"`
	Reasons  []string     `json:"reasons"`
}

// RiskRule defines a single risk check rule.
type RiskRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Score       int    `json:"score"`  // Points added if triggered
	Blocking    bool   `json:"blocking"` // If true, immediate REJECT
}

// DefaultRules returns the standard AML/risk rule set from the architecture doc.
func DefaultRules() []RiskRule {
	return []RiskRule{
		{
			Name:        "KYC_NOT_COMPLETED",
			Description: "User has not completed KYC verification",
			Score:       100,
			Blocking:    true,
		},
		{
			Name:        "USER_FROZEN",
			Description: "User account is frozen",
			Score:       100,
			Blocking:    true,
		},
		{
			Name:        "AMOUNT_EXCEEDS_LIMIT",
			Description: "Single transaction amount exceeds user risk level limit",
			Score:       40,
			Blocking:    false,
		},
		{
			Name:        "DAILY_LIMIT_EXCEEDED",
			Description: "Daily cumulative amount exceeds limit",
			Score:       30,
			Blocking:    false,
		},
		{
			Name:        "RESTRICTED_COUNTRY",
			Description: "Source or destination country is restricted",
			Score:       100,
			Blocking:    true,
		},
		{
			Name:        "HIGH_FREQUENCY",
			Description: "More than 10 transactions in 1 minute",
			Score:       50,
			Blocking:    false,
		},
		{
			Name:        "NEW_USER_LARGE_AMOUNT",
			Description: "New user (within 24h) making large transaction",
			Score:       60,
			Blocking:    false,
		},
		{
			Name:        "BLACKLISTED_SENDER",
			Description: "Sender is on the blacklist",
			Score:       100,
			Blocking:    true,
		},
		{
			Name:        "BLACKLISTED_RECEIVER",
			Description: "Receiver is on the blacklist",
			Score:       100,
			Blocking:    true,
		},
		{
			Name:        "SANCTIONED_COUNTRY",
			Description: "Country is under international sanctions",
			Score:       100,
			Blocking:    true,
		},
	}
}

// RiskLimits defines amount limits by risk level.
type RiskLimits struct {
	SingleTxLimit int64 `json:"single_tx_limit"` // In smallest currency unit
	DailyLimit    int64 `json:"daily_limit"`
	MonthlyLimit  int64 `json:"monthly_limit"`
}

// LimitsByRiskLevel returns transaction limits based on risk level.
func LimitsByRiskLevel(level string) RiskLimits {
	switch level {
	case "LOW":
		return RiskLimits{
			SingleTxLimit: 500000,   // $5,000
			DailyLimit:    2500000,  // $25,000
			MonthlyLimit:  5000000,  // $50,000
		}
	case "MEDIUM":
		return RiskLimits{
			SingleTxLimit: 100000,   // $1,000
			DailyLimit:    500000,   // $5,000
			MonthlyLimit:  1000000,  // $10,000
		}
	case "HIGH":
		return RiskLimits{
			SingleTxLimit: 50000,    // $500
			DailyLimit:    200000,   // $2,000
			MonthlyLimit:  500000,   // $5,000
		}
	default:
		return RiskLimits{
			SingleTxLimit: 100000,
			DailyLimit:    500000,
			MonthlyLimit:  1000000,
		}
	}
}

// RestrictedCountries returns the list of restricted/blocked countries.
func RestrictedCountries() map[string]bool {
	return map[string]bool{
		"KP": true, // North Korea
		"IR": true, // Iran
		"SY": true, // Syria
		"CU": true, // Cuba
	}
}

// BlacklistEntry represents a blacklisted entity.
type BlacklistEntry struct {
	ID       int64  `json:"-"`
	EntityID string `json:"entity_id"`
	Type     string `json:"type"` // "USER", "MERCHANT", "ADDRESS"
	Reason   string `json:"reason"`
}
