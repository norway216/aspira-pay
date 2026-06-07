// Package service implements business logic for Aspira Pay.
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

// UserService handles user registration, authentication, and management.
type UserService struct {
	db      *repository.DB
	authCfg config.AuthConfig
	jwt     *JWTManager
}

// JWTManager handles JWT token generation and validation.
type JWTManager struct {
	secret         []byte
	tokenExpiry    int64 // seconds
	refreshExpiry  int64 // seconds
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(secret string, tokenExpiry, refreshExpiry int64) *JWTManager {
	return &JWTManager{
		secret:        []byte(secret),
		tokenExpiry:   tokenExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// NewUserService creates a new UserService.
func NewUserService(db *repository.DB, jwt *JWTManager) *UserService {
	return &UserService{db: db, jwt: jwt}
}

// Register creates a new user account.
func (s *UserService) Register(req user.RegisterRequest) (*user.User, error) {
	// Check if username exists
	if existing, _ := s.db.GetUserByUsername(req.Username); existing != nil {
		return nil, pkgerrors.Conflict("username already exists")
	}

	// Check if email exists
	if existing, _ := s.db.GetUserByEmail(req.Email); existing != nil {
		return nil, pkgerrors.Conflict("email already exists")
	}

	// Hash password using Argon2id
	passwordHash := s.hashPassword(req.Password)

	currency := req.DefaultCurrency
	if currency == "" {
		currency = "USD"
	}

	u := &user.User{
		UserID:          idgen.UserID(),
		Username:        req.Username,
		Email:           req.Email,
		PasswordHash:    passwordHash,
		Phone:           req.Phone,
		Status:          user.UserPendingKYC,
		RiskLevel:       "LOW",
		DefaultCurrency: currency,
	}

	if err := s.db.CreateUser(u); err != nil {
		return nil, fmt.Errorf("cannot create user: %w", err)
	}

	return u, nil
}

// Login authenticates a user and returns JWT tokens.
func (s *UserService) Login(req user.LoginRequest) (*user.LoginResponse, error) {
	u, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		return nil, pkgerrors.Unauthorized("invalid username or password")
	}

	if !s.verifyPassword(req.Password, u.PasswordHash) {
		return nil, pkgerrors.Unauthorized("invalid username or password")
	}

	token, err := s.jwt.GenerateToken(u.UserID, u.Username)
	if err != nil {
		return nil, fmt.Errorf("cannot generate token: %w", err)
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(u.UserID)
	if err != nil {
		return nil, fmt.Errorf("cannot generate refresh token: %w", err)
	}

	return &user.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwt.tokenExpiry,
	}, nil
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(userID string) (*user.User, error) {
	return s.db.GetUserByID(userID)
}

// ListUsers returns a paginated user list.
func (s *UserService) ListUsers(page, pageSize int) ([]user.User, int64, error) {
	return s.db.ListUsers(page, pageSize)
}

// UpdateStatus updates a user's status.
func (s *UserService) UpdateStatus(userID string, status user.UserStatus) error {
	return s.db.UpdateUserStatus(userID, status)
}

// hashPassword hashes a password using Argon2id.
func (s *UserService) hashPassword(password string) string {
	salt := []byte("aspira-pay-salt") // In production, use per-user random salt
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		1,        // time
		64*1024,  // memory (64MB)
		4,        // threads
		32,       // key length
	)
	return fmt.Sprintf("%x", hash)
}

// verifyPassword verifies a password against its Argon2id hash.
func (s *UserService) verifyPassword(password, hash string) bool {
	expected := s.hashPassword(password)
	return hash == expected
}
