// Package errors provides domain-specific error types for Aspira Pay.
package errors

import "fmt"

// ErrorCode represents a machine-readable error code.
type ErrorCode string

const (
	ErrCodeInvalidInput       ErrorCode = "INVALID_INPUT"
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden          ErrorCode = "FORBIDDEN"
	ErrCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrCodeConflict           ErrorCode = "CONFLICT"
	ErrCodeDuplicate          ErrorCode = "DUPLICATE"
	ErrCodeRateLimited        ErrorCode = "RATE_LIMITED"
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"

	// Domain errors
	ErrCodeInsufficientFunds  ErrorCode = "INSUFFICIENT_FUNDS"
	ErrCodeInvalidState       ErrorCode = "INVALID_STATE"
	ErrCodeInvalidTransition  ErrorCode = "INVALID_TRANSITION"
	ErrCodeKYCPending         ErrorCode = "KYC_PENDING"
	ErrCodeKYCRejected        ErrorCode = "KYC_REJECTED"
	ErrCodeUserFrozen         ErrorCode = "USER_FROZEN"
	ErrCodeRiskRejected       ErrorCode = "RISK_REJECTED"
	ErrCodeRiskReview         ErrorCode = "RISK_MANUAL_REVIEW"
	ErrCodeQuoteExpired       ErrorCode = "QUOTE_EXPIRED"
	ErrCodeCurrencyMismatch   ErrorCode = "CURRENCY_MISMATCH"
	ErrCodeNegativeAmount     ErrorCode = "NEGATIVE_AMOUNT"
	ErrCodeIdempotencyMismatch ErrorCode = "IDEMPOTENCY_MISMATCH"
	ErrCodeEngineUnavailable  ErrorCode = "ENGINE_UNAVAILABLE"
	ErrCodeSettlementFailed   ErrorCode = "SETTLEMENT_FAILED"
	ErrCodeChainFailed        ErrorCode = "CHAIN_FAILED"
	ErrCodeLedgerImbalance    ErrorCode = "LEDGER_IMBALANCE"
)

// DomainError is a typed error with code, message, and optional cause.
type DomainError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Cause   error     `json:"-"`
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error { return e.Cause }

// New creates a new DomainError.
func New(code ErrorCode, message string) *DomainError {
	return &DomainError{Code: code, Message: message}
}

// Wrap creates a DomainError with an underlying cause.
func Wrap(code ErrorCode, message string, cause error) *DomainError {
	return &DomainError{Code: code, Message: message, Cause: cause}
}

// Convenience constructors for common errors.

func InvalidInput(msg string) *DomainError    { return New(ErrCodeInvalidInput, msg) }
func Unauthorized(msg string) *DomainError    { return New(ErrCodeUnauthorized, msg) }
func NotFound(msg string) *DomainError        { return New(ErrCodeNotFound, msg) }
func Conflict(msg string) *DomainError        { return New(ErrCodeConflict, msg) }
func Duplicate(msg string) *DomainError       { return New(ErrCodeDuplicate, msg) }
func InsufficientFunds(available, required int64) *DomainError {
	return New(ErrCodeInsufficientFunds,
		fmt.Sprintf("insufficient funds: available=%d, required=%d", available, required))
}
func InvalidState(current, expected string) *DomainError {
	return New(ErrCodeInvalidState,
		fmt.Sprintf("invalid state: current=%s, expected=%s", current, expected))
}
func InvalidTransition(from, to string) *DomainError {
	return New(ErrCodeInvalidTransition,
		fmt.Sprintf("invalid state transition: %s -> %s", from, to))
}

// IsDomainError checks if an error is a DomainError with the given code.
func IsDomainError(err error, code ErrorCode) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == code
	}
	return false
}
