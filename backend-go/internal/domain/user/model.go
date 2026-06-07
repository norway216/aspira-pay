// Package user defines the User domain model.
package user

import "time"

// UserStatus represents the lifecycle state of a user.
type UserStatus string

const (
	UserPendingKYC UserStatus = "PENDING_KYC"
	UserActive     UserStatus = "ACTIVE"
	UserFrozen     UserStatus = "FROZEN"
	UserRejected   UserStatus = "REJECTED"
)

// ValidUserStatuses is the set of valid user statuses.
var ValidUserStatuses = map[UserStatus]bool{
	UserPendingKYC: true,
	UserActive:     true,
	UserFrozen:     true,
	UserRejected:   true,
}

// ValidTransitions defines allowed state transitions.
var ValidTransitions = map[UserStatus][]UserStatus{
	UserPendingKYC: {UserActive, UserRejected, UserFrozen},
	UserActive:     {UserFrozen},
	UserFrozen:     {UserActive},
	UserRejected:   {}, // Terminal state
}

// CanTransition checks if a state transition is valid.
func CanTransition(from, to UserStatus) bool {
	for _, valid := range ValidTransitions[from] {
		if valid == to {
			return true
		}
	}
	return false
}

// User represents a registered user in the system.
type User struct {
	ID              int64      `json:"-"`
	UserID          string     `json:"user_id"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	Phone           string     `json:"phone,omitempty"`
	Status          UserStatus `json:"status"`
	RiskLevel       string     `json:"risk_level"`
	DefaultCurrency string     `json:"default_currency"` // User's preferred currency for display
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// GetDefaultCurrency returns the user's preferred currency, defaulting to USD.
func (u *User) GetDefaultCurrency() string {
	if u.DefaultCurrency == "" {
		return "USD"
	}
	return u.DefaultCurrency
}

// IsActive checks if the user is in ACTIVE status.
func (u *User) IsActive() bool { return u.Status == UserActive }

// IsFrozen checks if the user is frozen.
func (u *User) IsFrozen() bool { return u.Status == UserFrozen }

// CanTransact checks if the user can perform transactions.
func (u *User) CanTransact() bool { return u.Status == UserActive }

// RegisterRequest is the input for user registration.
type RegisterRequest struct {
	Username        string `json:"username" binding:"required"`
	Email           string `json:"email" binding:"required"`
	Password        string `json:"password" binding:"required"`
	Phone           string `json:"phone,omitempty"`
	DefaultCurrency string `json:"default_currency,omitempty"` // USD, EUR, JPY, etc.
}

// LoginRequest is the input for user login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is the output for user login.
type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
