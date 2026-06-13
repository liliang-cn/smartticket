package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

// JWTClaims represents the claims structure for JWT tokens.
type JWTClaims struct {
	UserID     uint   `json:"user_id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	CustomerID *uint  `json:"customer_id,omitempty"`
	jwt.RegisteredClaims
}

// TokenPair represents both access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// LoginRequest represents the login request payload.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"admin@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

// LoginResponse represents the login response.
type LoginResponse struct {
	Success   bool       `json:"success"`
	User      *UserInfo  `json:"user"`
	Tokens    *TokenPair `json:"tokens"`
	ExpiresIn int64      `json:"expires_in"` // seconds
	RefreshIn int64      `json:"refresh_in"` // seconds
}

// UserInfo represents safe user information for responses.
type UserInfo struct {
	ID           uint       `json:"id"`
	Email        string     `json:"email"`
	Username     string     `json:"username"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Role         string     `json:"role"`
	CustomerID   *uint      `json:"customer_id,omitempty"`
	DepartmentID *uint      `json:"department_id,omitempty"`
	IsActive     bool       `json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

// RefreshTokenRequest represents the refresh token request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ChangePasswordRequest represents the password change request.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=NewPassword"`
}

// Service provides authentication and authorization functionality.
type Service struct {
	db            *gorm.DB
	jwtSecret     []byte
	accessTokenT  time.Duration
	refreshTokenT time.Duration
	issuer        string
}

// NewService creates a new authentication service.
func NewService(db *gorm.DB, jwtSecret string, accessTokenT, refreshTokenT time.Duration, issuer string) *Service {
	return &Service{
		db:            db,
		jwtSecret:     []byte(jwtSecret),
		accessTokenT:  accessTokenT,
		refreshTokenT: refreshTokenT,
		issuer:        issuer,
	}
}

// Login authenticates a user and returns tokens.
func (s *Service) Login(req *LoginRequest, clientIP, userAgent string) (*LoginResponse, error) {
	// Find user by email
	var user models.User
	if err := s.db.Where("email = ? AND is_active = ?", req.Email, true).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := s.verifyPassword(req.Password, user.PasswordHash); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.db.Save(&user).Error; err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create user info
	userInfo := s.createUserInfo(&user)

	return &LoginResponse{
		Success:   true,
		User:      userInfo,
		Tokens:    tokens,
		ExpiresIn: int64(s.accessTokenT.Seconds()),
		RefreshIn: int64(s.refreshTokenT.Seconds()),
	}, nil
}

// RefreshToken generates new tokens using a refresh token.
func (s *Service) RefreshToken(refreshToken string) (*TokenPair, error) {
	// Parse and validate refresh token
	token, err := jwt.ParseWithClaims(refreshToken, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token claims")
	}

	// Check if token is refresh token
	if claims.Subject != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// Find user
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", claims.UserID, true).
		First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found or inactive: %w", err)
	}

	// Generate new token pair
	return s.generateTokenPair(&user)
}

// ValidateToken validates an access token and returns claims.
func (s *Service) ValidateToken(accessToken string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(accessToken, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid access token claims")
	}

	// Check if token is access token
	if claims.Subject != "access" {
		return nil, errors.New("invalid token type")
	}

	// Check if user is still active
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", claims.UserID, true).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found or inactive: %w", err)
	}

	return claims, nil
}

// ChangePassword changes a user's password.
func (s *Service) ChangePassword(userID uint, req *ChangePasswordRequest) error {
	// Find user
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify current password
	if err := s.verifyPassword(req.CurrentPassword, user.PasswordHash); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	user.PasswordHash = hashedPassword
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// GetUserInfo returns user information by ID.
func (s *Service) GetUserInfo(userID uint) (*UserInfo, error) {
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).
		First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	userInfo := s.createUserInfo(&user)
	return userInfo, nil
}

// CreateUserInfo creates safe user info for responses (public method for services).
func (s *Service) CreateUserInfo(user *models.User) *UserInfo {
	return s.createUserInfo(user)
}

// Helper methods

// generateTokenPair generates both access and refresh tokens for a user.
func (s *Service) generateTokenPair(user *models.User) (*TokenPair, error) {
	now := time.Now()

	// Use user's role directly for token generation
	effectiveRole := user.Role

	// Generate access token
	accessClaims := &JWTClaims{
		UserID:     user.ID,
		Email:      user.Email,
		Role:       effectiveRole,
		CustomerID: user.CustomerID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "access",
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenT)),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := &JWTClaims{
		UserID:     user.ID,
		Email:      user.Email,
		Role:       effectiveRole,
		CustomerID: user.CustomerID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "refresh",
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenT)),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    now.Add(s.accessTokenT),
		TokenType:    "Bearer",
	}, nil
}

// hashPassword hashes a password using bcrypt.
func (s *Service) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// verifyPassword verifies a password against its hash.
func (s *Service) verifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// createUserInfo creates safe user info for responses.
func (s *Service) createUserInfo(user *models.User) *UserInfo {
	info := &UserInfo{
		ID:           user.ID,
		Email:        user.Email,
		Username:     user.Username,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Role:         user.Role,
		CustomerID:   user.CustomerID,
		DepartmentID: user.DepartmentID,
		IsActive:     user.IsActive,
		LastLoginAt:  user.LastLoginAt,
	}

	return info
}
