package service

import (
	"fmt"

	"golang.org/x/crypto/argon2"

	"github.com/aspira/aspira-pay/internal/config"
	"github.com/aspira/aspira-pay/internal/domain/user"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/idgen"
	pkgerrors "github.com/aspira/aspira-pay/pkg/errors"
)

type UserService struct {
	db      *repository.DB
	authCfg config.AuthConfig
	jwt     *JWTManager
}

type JWTManager struct {
	secret        []byte
	tokenExpiry   int64
	refreshExpiry int64
}

func NewJWTManager(secret string, tokenExpiry, refreshExpiry int64) *JWTManager {
	return &JWTManager{secret: []byte(secret), tokenExpiry: tokenExpiry, refreshExpiry: refreshExpiry}
}

func NewUserService(db *repository.DB, jwt *JWTManager) *UserService {
	return &UserService{db: db, jwt: jwt}
}

// Register creates a new user with optional KYC data.
// Admin users are identified by admin_key starting with "aspira-".
func (s *UserService) Register(req user.RegisterRequest) (*user.User, error) {
	if existing, _ := s.db.GetUserByUsername(req.Username); existing != nil {
		return nil, pkgerrors.Conflict("username already exists")
	}
	if existing, _ := s.db.GetUserByEmail(req.Email); existing != nil {
		return nil, pkgerrors.Conflict("email already exists")
	}

	currency := req.DefaultCurrency
	if currency == "" { currency = "USD" }

	u := &user.User{
		UserID:          idgen.UserID(),
		Username:        req.Username,
		Email:           req.Email,
		PasswordHash:    s.hashPassword(req.Password),
		Phone:           req.Phone,
		Status:          user.UserPendingKYC,
		RiskLevel:       "LOW",
		DefaultCurrency: currency,
		AdminKey:        req.AdminKey,
		FullName:        req.FullName,
		Nationality:     req.Nationality,
		DateOfBirth:     req.DateOfBirth,
	}

	// Auto-activate admin users (with valid aspira- key)
	if u.IsAdmin() {
		u.Status = user.UserActive
	}

	if err := s.db.CreateUser(u); err != nil {
		return nil, fmt.Errorf("cannot create user: %w", err)
	}
	return u, nil
}

// Login authenticates and returns JWT with role info.
// §5.3: Login failure rate limiting — 5 failures = 15min lockout.
func (s *UserService) Login(req user.LoginRequest) (*user.LoginResponse, error) {
	// Check if account is locked
	if s.db.IsUserLocked(req.Username) {
		return nil, pkgerrors.New("ACCOUNT_LOCKED", "account temporarily locked due to too many failed attempts")
	}

	u, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		s.db.RecordLoginAttempt(req.Username, "", false)
		return nil, pkgerrors.Unauthorized("invalid username or password")
	}
	if !s.verifyPassword(req.Password, u.PasswordHash) {
		s.db.RecordLoginAttempt(req.Username, "", false)
		s.db.IncrementFailedLoginCount(req.Username)
		return nil, pkgerrors.Unauthorized("invalid username or password")
	}

	// Reset failed count on success
	s.db.ResetFailedLoginCount(req.Username)
	s.db.RecordLoginAttempt(req.Username, "", true)

	token, _ := s.jwt.GenerateToken(u.UserID, u.Username, u.IsAdmin())
	refreshToken, _ := s.jwt.GenerateRefreshToken(u.UserID)

	return &user.LoginResponse{
		Token: token, RefreshToken: refreshToken,
		ExpiresIn: s.jwt.tokenExpiry,
		UserID: u.UserID, Username: u.Username, IsAdmin: u.IsAdmin(),
	}, nil
}

// SetTransactionPIN sets the transaction PIN for a user.
func (s *UserService) SetTransactionPIN(userID, pin string) error {
	return s.db.UpdateUserPIN(userID, s.hashPassword(pin))
}

// VerifyTransactionPIN checks if the provided PIN matches.
func (s *UserService) VerifyTransactionPIN(userID, pin string) bool {
	u, err := s.db.GetUserByID(userID)
	if err != nil { return false }
	return u.TransactionPIN == s.hashPassword(pin)
}

func (s *UserService) GetUser(userID string) (*user.User, error) { return s.db.GetUserByID(userID) }

func (s *UserService) ListUsers(page, pageSize int) ([]user.User, int64, error) {
	return s.db.ListUsers(page, pageSize)
}

func (s *UserService) UpdateStatus(userID string, status user.UserStatus) error {
	return s.db.UpdateUserStatus(userID, status)
}

func (s *UserService) hashPassword(password string) string {
	hash := argon2.IDKey([]byte(password), []byte("aspira-pay-salt"), 1, 64*1024, 4, 32)
	return fmt.Sprintf("%x", hash)
}

func (s *UserService) verifyPassword(password, hash string) bool {
	return hash == s.hashPassword(password)
}
