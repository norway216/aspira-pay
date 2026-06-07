// Package money provides int64-based monetary operations.
// All amounts are in the smallest currency unit (cents, sen, etc.).
// NEVER use float64 for money — this is a hard architectural constraint.
package money

import (
	"fmt"
	"math"
)

// Amount represents a monetary value in the smallest currency unit (e.g., cents).
// Using int64 avoids floating-point precision errors in financial calculations.
type Amount int64

// New creates an Amount from the smallest unit. Returns error if negative.
func New(v int64) (Amount, error) {
	if v < 0 {
		return 0, fmt.Errorf("money: amount must be non-negative, got %d", v)
	}
	return Amount(v), nil
}

// MustNew creates an Amount, panicking if negative.
func MustNew(v int64) Amount {
	a, err := New(v)
	if err != nil {
		panic(err)
	}
	return a
}

// FromMajor creates an Amount from major units (e.g., dollars).
// decimals specifies the number of decimal places (e.g., 2 for USD cents).
func FromMajor(major float64, decimals int) (Amount, error) {
	if decimals < 0 || decimals > 8 {
		return 0, fmt.Errorf("money: decimals must be 0-8, got %d", decimals)
	}
	multiplier := math.Pow10(decimals)
	v := int64(major * multiplier)
	if v < 0 {
		return 0, fmt.Errorf("money: amount must be non-negative")
	}
	return Amount(v), nil
}

// Int64 returns the raw int64 value (smallest unit).
func (a Amount) Int64() int64 { return int64(a) }

// ToMajor converts to major units as float64 (for display only, NOT for calculations).
func (a Amount) ToMajor(decimals int) float64 {
	return float64(a) / math.Pow10(decimals)
}

// String formats the amount with currency symbol.
func (a Amount) String(currency string) string {
	switch currency {
	case "USD", "EUR", "GBP", "AUD", "CAD":
		return fmt.Sprintf("%s %.2f", currency, a.ToMajor(2))
	case "JPY":
		return fmt.Sprintf("%s %d", currency, a.Int64())
	case "CNY":
		return fmt.Sprintf("%s %.2f", currency, a.ToMajor(2))
	default:
		return fmt.Sprintf("%s %d", currency, a.Int64())
	}
}

// Add adds two amounts.
func (a Amount) Add(b Amount) Amount { return Amount(int64(a) + int64(b)) }

// Sub subtracts b from a. Returns error if result would be negative.
func (a Amount) Sub(b Amount) (Amount, error) {
	if int64(b) > int64(a) {
		return 0, fmt.Errorf("money: insufficient funds: %d - %d", a, b)
	}
	return Amount(int64(a) - int64(b)), nil
}

// MustSub subtracts b from a, panicking if result would be negative.
func (a Amount) MustSub(b Amount) Amount {
	r, err := a.Sub(b)
	if err != nil {
		panic(err)
	}
	return r
}

// Mult multiplies amount by a factor (e.g., for FX conversion).
// The result is rounded down.
func (a Amount) Mult(numerator, denominator int64) Amount {
	if denominator == 0 {
		panic("money: division by zero")
	}
	return Amount(int64(a) * numerator / denominator)
}

// Fee calculates a fee as percentage of the amount in basis points (1/100th of a percent).
// e.g., 100 bps = 1%, 15 bps = 0.15%
func (a Amount) Fee(basisPoints int64) Amount {
	return Amount(int64(a) * basisPoints / 10000)
}

// Equals checks equality.
func (a Amount) Equals(b Amount) bool { return int64(a) == int64(b) }

// LessThan checks if a < b.
func (a Amount) LessThan(b Amount) bool { return int64(a) < int64(b) }

// GreaterThan checks if a > b.
func (a Amount) GreaterThan(b Amount) bool { return int64(a) > int64(b) }

// Zero returns true if amount is zero.
func (a Amount) Zero() bool { return int64(a) == 0 }

// ValidCurrency checks if a currency code is supported.
func ValidCurrency(currency string) bool {
	switch currency {
	case "USD", "EUR", "GBP", "JPY", "CNY", "AUD", "CAD", "CHF", "HKD", "SGD":
		return true
	default:
		return false
	}
}

// CurrencyDecimals returns the number of decimal places for a currency.
func CurrencyDecimals(currency string) int {
	switch currency {
	case "JPY", "KRW":
		return 0
	case "BHD", "KWD", "OMR":
		return 3
	default:
		return 2
	}
}

// FXDecimals is the precision used for FX rates (12 decimal places).
const FXDecimals = 12
