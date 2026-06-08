package user

import "time"

type UserStatus string

const (
	UserPendingKYC UserStatus = "PENDING_KYC"
	UserActive     UserStatus = "ACTIVE"
	UserFrozen     UserStatus = "FROZEN"
	UserRejected   UserStatus = "REJECTED"
)

var ValidTransitions = map[UserStatus][]UserStatus{
	UserPendingKYC: {UserActive, UserRejected, UserFrozen},
	UserActive:     {UserFrozen},
	UserFrozen:     {UserActive},
	UserRejected:   {},
}

func CanTransition(from, to UserStatus) bool {
	for _, v := range ValidTransitions[from] {
		if v == to { return true }
	}
	return false
}

// User represents a registered user.
// Admin users have an admin_key starting with "aspira-".
// Regular users do not have an admin_key.
type User struct {
	ID              int64      `json:"-"`
	UserID          string     `json:"user_id"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	Phone           string     `json:"phone,omitempty"`
	Status          UserStatus `json:"status"`
	RiskLevel       string     `json:"risk_level"`
	DefaultCurrency string     `json:"default_currency"`
	AdminKey        string     `json:"admin_key,omitempty"`     // "aspira-" prefix = admin
	TransactionPIN  string     `json:"-"`                        // Hashed transaction PIN
	HasPIN          bool       `json:"has_pin"`                  // Whether PIN is set
	FullName        string     `json:"full_name,omitempty"`      // KYC
	Nationality     string     `json:"nationality,omitempty"`    // KYC
	DateOfBirth     string     `json:"date_of_birth,omitempty"`  // KYC
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (u *User) IsAdmin() bool {
	return len(u.AdminKey) > 0 && u.AdminKey[:7] == "aspira-"
}

func (u *User) GetDefaultCurrency() string {
	if u.DefaultCurrency == "" { return "USD" }
	return u.DefaultCurrency
}

func (u *User) IsActive() bool    { return u.Status == UserActive }
func (u *User) IsFrozen() bool    { return u.Status == UserFrozen }
func (u *User) CanTransact() bool { return u.Status == UserActive }

// RegisterRequest is the input for user registration.
type RegisterRequest struct {
	Username        string `json:"username" binding:"required"`
	Email           string `json:"email" binding:"required"`
	Password        string `json:"password" binding:"required"`
	Phone           string `json:"phone,omitempty"`
	DefaultCurrency string `json:"default_currency,omitempty"`
	AdminKey        string `json:"admin_key,omitempty"`      // "aspira-xxx" for admin
	FullName        string `json:"full_name,omitempty"`       // KYC
	Nationality     string `json:"nationality,omitempty"`     // KYC
	DateOfBirth     string `json:"date_of_birth,omitempty"`   // KYC format: YYYY-MM-DD
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	IsAdmin      bool   `json:"is_admin"`
}

// SetPINRequest is the input for setting a transaction PIN.
type SetPINRequest struct {
	PIN string `json:"pin" binding:"required,min=4,max=12"`
}

// CardApplicationRequest includes KYC data for card issuance.
type CardApplicationRequest struct {
	CardNetwork     string `json:"card_network"`
	DefaultCurrency string `json:"default_currency"`
	FullName        string `json:"full_name" binding:"required"`
	Nationality     string `json:"nationality" binding:"required"`
	DateOfBirth     string `json:"date_of_birth" binding:"required"`
	Address         string `json:"address"`
	DocumentType    string `json:"document_type"`
	DocumentNumber  string `json:"document_number"`
}
