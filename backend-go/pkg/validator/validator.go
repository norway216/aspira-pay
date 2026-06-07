// Package validator provides input validation helpers for API requests.
package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aspira/aspira-pay/pkg/money"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	userIDRegex   = regexp.MustCompile(`^u_[a-z0-9]{12}$`)
	paymentIDRegex = regexp.MustCompile(`^pay_\d{8}_[a-z0-9]{8}$`)
	currencyRegex  = regexp.MustCompile(`^[A-Z]{3}$`)
	countryRegex   = regexp.MustCompile(`^[A-Z]{2}$`)
)

// ValidateEmail checks email format.
func ValidateEmail(email string) error {
	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("email is required")
	}
	if len(email) > 255 {
		return fmt.Errorf("email too long")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidateUsername checks username format.
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if len(username) > 64 {
		return fmt.Errorf("username must be at most 64 characters")
	}
	return nil
}

// ValidatePassword checks password strength.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return fmt.Errorf("password too long")
	}
	return nil
}

// ValidateCurrency checks if a currency code is valid and supported.
func ValidateCurrency(currency string) error {
	if !currencyRegex.MatchString(currency) {
		return fmt.Errorf("currency must be a 3-letter ISO code")
	}
	if !money.ValidCurrency(currency) {
		return fmt.Errorf("unsupported currency: %s", currency)
	}
	return nil
}

// ValidateCountry checks if country code is valid.
func ValidateCountry(country string) error {
	if !countryRegex.MatchString(country) {
		return fmt.Errorf("country must be a 2-letter ISO code")
	}
	return nil
}

// ValidateAmount checks that an amount is positive.
func ValidateAmount(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive, got %d", amount)
	}
	return nil
}

// ValidateUserID checks user ID format.
func ValidateUserID(id string) error {
	if !userIDRegex.MatchString(id) {
		return fmt.Errorf("invalid user_id format")
	}
	return nil
}

// ValidatePaymentID checks payment ID format.
func ValidatePaymentID(id string) error {
	if !paymentIDRegex.MatchString(id) {
		return fmt.Errorf("invalid payment_id format")
	}
	return nil
}

// ValidateRequired checks that a string field is non-empty.
func ValidateRequired(field, name string) error {
	if strings.TrimSpace(field) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

// ValidateName checks a person name.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) < 1 {
		return fmt.Errorf("name is required")
	}
	if len(name) > 255 {
		return fmt.Errorf("name too long")
	}
	return nil
}
