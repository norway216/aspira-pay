// Package fx defines the FX Quote domain model.
// IMPORTANT: FX rates use string/NUMERIC representation, NEVER float64.
package fx

import "time"

// Quote represents an FX exchange rate quote.
type Quote struct {
	ID             int64     `json:"-"`
	QuoteID        string    `json:"quote_id"`
	SourceCurrency string    `json:"source_currency"`
	TargetCurrency string    `json:"target_currency"`
	Rate           string    `json:"rate"`           // NUMERIC string, e.g., "156.000000000000"
	SourceAmount   int64     `json:"source_amount"`   // In smallest currency unit
	TargetAmount   int64     `json:"target_amount"`   // In smallest currency unit
	FeeAmount      int64     `json:"fee_amount"`      // In smallest currency unit
	FeeBasisPoints int64     `json:"fee_basis_points"` // e.g., 100 = 1%
	ExpiresAt      int64     `json:"expires_at"`      // Unix timestamp
	Status         string    `json:"status"`          // ACTIVE, USED, EXPIRED, CANCELLED
	CreatedAt      time.Time `json:"created_at"`
}

// IsExpired checks if the quote has expired.
func (q *Quote) IsExpired(nowUnix int64) bool {
	return nowUnix > q.ExpiresAt
}

// IsActive checks if the quote is usable.
func (q *Quote) IsActive() bool {
	return q.Status == "ACTIVE"
}

// QuoteRequest is the API input for requesting an FX quote.
type QuoteRequest struct {
	SourceCurrency string `json:"source_currency" binding:"required"`
	TargetCurrency string `json:"target_currency" binding:"required"`
	SourceAmount   int64  `json:"source_amount" binding:"required"`
}

// QuoteResponse is the API output for an FX quote.
type QuoteResponse struct {
	QuoteID      string `json:"quote_id"`
	SourceCurrency string `json:"source_currency"`
	TargetCurrency string `json:"target_currency"`
	Rate         string `json:"rate"`
	SourceAmount int64  `json:"source_amount"`
	TargetAmount int64  `json:"target_amount"`
	FeeAmount    int64  `json:"fee_amount"`
	ExpiresAt    int64  `json:"expires_at"`
}

// RateConfig holds simulated exchange rates for the Sandbox environment.
type RateConfig struct {
	Pairs map[string]string `json:"pairs"` // "USD/JPY" -> "156.000000000000"
}

// DefaultRates returns default Sandbox FX rates.
func DefaultRates() map[string]string {
	return map[string]string{
		"USD/JPY": "156.000000000000",
		"USD/EUR": "0.920000000000",
		"USD/CNY": "7.250000000000",
		"USD/GBP": "0.790000000000",
		"EUR/JPY": "169.565217391304",
		"EUR/USD": "1.086956521739",
		"CNY/USD": "0.137931034483",
		"CNY/JPY": "21.517241379310",
		"GBP/USD": "1.265822784810",
		"GBP/JPY": "197.468354430380",
	}
}

// FeeBasisPoints returns the fee in basis points for a given currency pair.
// Default is 100 bps = 1%.
func FeeBasisPoints(source, target string) int64 {
	if source != target {
		return 100 // 1%
	}
	return 50 // 0.5%
}

// FeeBasisPointsUSD returns the fee for USD-based settlement.
// All transactions settle in USD, so fee is calculated on USD equivalent.
func FeeBasisPointsUSD() int64 {
	return 100 // 1% fee on USD settlement amount
}

// DefaultUSDRates returns fallback USD-based rates (XXX per 1 USD).
// Used when live API is unavailable.
func DefaultUSDRates() map[string]string {
	return map[string]string{
		"JPY": "156.000000000000",
		"EUR": "0.920000000000",
		"CNY": "7.250000000000",
		"GBP": "0.790000000000",
		"AUD": "1.520000000000",
		"CAD": "1.360000000000",
		"CHF": "0.890000000000",
		"HKD": "7.810000000000",
		"SGD": "1.340000000000",
		"KRW": "1320.000000000000",
		"INR": "83.000000000000",
		"MXN": "17.500000000000",
		"BRL": "5.100000000000",
	}
}

// Supported currencies list
var SupportedCurrencies = []string{
	"USD", "EUR", "GBP", "JPY", "CNY", "AUD", "CAD", "CHF", "HKD", "SGD",
}

// QuoteTTLSeconds is the default quote validity period.
const QuoteTTLSeconds = 120 // 2 minutes
