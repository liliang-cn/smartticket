package user

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/models"
)

// Service provides user management business logic.
type Service struct {
	db            *gorm.DB
	repo          *auth.Repository
	authService   *auth.Service
	passwordRules PasswordRules
}

// PasswordRules defines password complexity requirements.
type PasswordRules struct {
	MinLength        int    `json:"min_length"`
	RequireUppercase bool   `json:"require_uppercase"`
	RequireLowercase bool   `json:"require_lowercase"`
	RequireDigit     bool   `json:"require_digit"`
	RequireSpecial   bool   `json:"require_special"`
	SpecialChars     string `json:"special_chars"`
}

// DefaultPasswordRules returns sensible default password rules.
func DefaultPasswordRules() PasswordRules {
	return PasswordRules{
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   true,
		SpecialChars:     "!@#$%^&*()_+-=[]{}|;:,.<>?",
	}
}

// NewService creates a new user management service.
func NewService(db *gorm.DB, repo *auth.Repository, authService *auth.Service) *Service {
	return &Service{
		db:            db,
		repo:          repo,
		authService:   authService,
		passwordRules: DefaultPasswordRules(),
	}
}

// CreateUserRequest represents user creation request.
type CreateUserRequest struct {
	Email       string `json:"email" binding:"required,email" example:"user@example.com"`
	Username    string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
	FirstName   string `json:"first_name" binding:"required,min=1,max=100" example:"John"`
	LastName    string `json:"last_name" binding:"required,min=1,max=100" example:"Doe"`
	Password    string `json:"password" binding:"required,min=8" example:"SecurePass123!"`
	Role        string `json:"role" binding:"required,oneof=admin engineer support customer sales" example:"customer"`
	IsActive    bool   `json:"is_active" example:"true"`
	Preferences string `json:"preferences,omitempty" example:"{\"timezone\": \"UTC\", \"language\": \"en\"}"`
}

// UpdateUserRequest represents user update request.
type UpdateUserRequest struct {
	Email       string `json:"email,omitempty" binding:"omitempty,email" example:"user@example.com"`
	Username    string `json:"username,omitempty" binding:"omitempty,min=3,max=50" example:"johndoe"`
	FirstName   string `json:"first_name,omitempty" binding:"omitempty,min=1,max=100" example:"John"`
	LastName    string `json:"last_name,omitempty" binding:"omitempty,min=1,max=100" example:"Doe"`
	Role        string `json:"role,omitempty" binding:"omitempty,oneof=admin engineer support customer sales" example:"customer"`
	IsActive    *bool  `json:"is_active,omitempty" example:"true"`
	Preferences string `json:"preferences,omitempty" example:"{\"timezone\": \"UTC\", \"language\": \"en\"}"`
}

// UserListRequest represents user listing request with filters.
type UserListRequest struct {
	Page     int    `form:"page,default=1" binding:"min=1" example:"1"`
	PageSize int    `form:"page_size,default=20" binding:"min=1,max=100" example:"20"`
	Search   string `form:"search,omitempty" example:"john"`
	Role     string `form:"role,omitempty" example:"customer"`
	IsActive *bool  `form:"is_active,omitempty" example:"true"`
}

// UserListResponse represents user listing response.
type UserListResponse struct {
	Success bool            `json:"success"`
	Data    []auth.UserInfo `json:"data"`
	Meta    *PaginationMeta `json:"meta,omitempty"`
}

// PaginationMeta represents pagination metadata.
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// CreateUser creates a new user with validation.
func (s *Service) CreateUser(tenantID uint, req *CreateUserRequest) (*auth.UserInfo, error) {
	// Normalize email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.TrimSpace(req.Username)

	// Validate email format
	if !s.isValidEmail(req.Email) {
		return nil, errors.New("invalid email format")
	}

	// Validate username format
	if !s.isValidUsername(req.Username) {
		return nil, errors.New("username can only contain letters, numbers, underscores, and hyphens")
	}

	// Validate password complexity
	if err := s.validatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("password validation failed: %w", err)
	}

	// Check if email already exists
	exists, err := s.repo.CheckEmailExists(req.Email, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, errors.New("email already exists")
	}

	// Check if username already exists
	exists, err = s.repo.CheckUsernameExists(req.Username, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return nil, errors.New("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user (role is managed separately through role assignments)
	user := &models.User{
		TenantID:     tenantID,
		Email:        req.Email,
		Username:     req.Username,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PasswordHash: string(hashedPassword),
		IsActive:     req.IsActive,
		Preferences:  req.Preferences,
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Return user info
	return s.createUserInfo(user), nil
}

// GetUser retrieves a user by ID.
func (s *Service) GetUser(userID, tenantID uint) (*auth.UserInfo, error) {
	user, err := s.repo.GetUserByID(userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.createUserInfo(user), nil
}

// UpdateUser updates user information.
func (s *Service) UpdateUser(userID, tenantID uint, req *UpdateUserRequest) (*auth.UserInfo, error) {
	// Get existing user
	user, err := s.repo.GetUserByID(userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Update fields if provided
	if req.Email != "" {
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		if !s.isValidEmail(req.Email) {
			return nil, errors.New("invalid email format")
		}

		// Check if email already exists (excluding current user)
		exists, err := s.repo.CheckEmailExists(req.Email, tenantID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to check email existence: %w", err)
		}
		if exists {
			return nil, errors.New("email already exists")
		}
		user.Email = req.Email
	}

	if req.Username != "" {
		req.Username = strings.TrimSpace(req.Username)
		if !s.isValidUsername(req.Username) {
			return nil, errors.New("username can only contain letters, numbers, underscores, and hyphens")
		}

		// Check if username already exists (excluding current user)
		exists, err := s.repo.CheckUsernameExists(req.Username, tenantID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to check username existence: %w", err)
		}
		if exists {
			return nil, errors.New("username already exists")
		}
		user.Username = req.Username
	}

	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}

	if req.LastName != "" {
		user.LastName = req.LastName
	}

	// Note: Role is managed separately through role assignments, not direct user updates

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if req.Preferences != "" {
		user.Preferences = req.Preferences
	}

	// Update user
	if err := s.repo.UpdateUser(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.createUserInfo(user), nil
}

// DeleteUser soft deletes a user.
func (s *Service) DeleteUser(userID, tenantID uint) error {
	// Get user to verify existence
	_, err := s.repo.GetUserByID(userID, tenantID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Soft delete user
	if err := s.repo.DeleteUser(userID, tenantID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ListUsers retrieves a paginated list of users with filters.
func (s *Service) ListUsers(tenantID uint, req *UserListRequest) (*UserListResponse, error) {
	// Prepare filters
	filters := make(map[string]interface{})
	if req.Search != "" {
		filters["search"] = req.Search
	}
	if req.Role != "" {
		filters["role"] = req.Role
	}
	if req.IsActive != nil {
		filters["is_active"] = *req.IsActive
	}

	// Get users
	users, total, err := s.repo.ListUsers(tenantID, req.Page, req.PageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert to user info
	userInfos := make([]auth.UserInfo, len(users))
	for i, user := range users {
		userInfos[i] = *s.createUserInfo(&user)
	}

	// Calculate pagination
	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &UserListResponse{
		Success: true,
		Data:    userInfos,
		Meta: &PaginationMeta{
			Page:       req.Page,
			PageSize:   req.PageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// ActivateUser activates a user account.
func (s *Service) ActivateUser(userID, tenantID uint) error {
	return s.repo.ActivateUser(userID, tenantID)
}

// DeactivateUser deactivates a user account.
func (s *Service) DeactivateUser(userID, tenantID uint) error {
	return s.repo.DeactivateUser(userID, tenantID)
}

// ChangeUserPassword changes a user's password (admin function).
func (s *Service) ChangeUserPassword(userID, tenantID uint, newPassword string) error {
	// Validate password complexity
	if err := s.validatePassword(newPassword); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.repo.UpdatePassword(userID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// GetUserStats returns user statistics for a tenant.
func (s *Service) GetUserStats(tenantID uint) (map[string]interface{}, error) {
	stats, err := s.repo.GetUserStats(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// Convert to map with string keys
	result := make(map[string]interface{})
	for k, v := range stats {
		result[k] = v
	}

	return result, nil
}

// Helper methods

// isValidEmail validates email format.
func (s *Service) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// isValidUsername validates username format.
func (s *Service) isValidUsername(username string) bool {
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return usernameRegex.MatchString(username) && len(username) >= 3 && len(username) <= 50
}

// validatePassword validates password complexity.
func (s *Service) validatePassword(password string) error {
	rules := s.passwordRules

	// Check minimum length
	if len(password) < rules.MinLength {
		return fmt.Errorf("password must be at least %d characters long", rules.MinLength)
	}

	// Check uppercase requirement
	if rules.RequireUppercase {
		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
		if !hasUpper {
			return errors.New("password must contain at least one uppercase letter")
		}
	}

	// Check lowercase requirement
	if rules.RequireLowercase {
		hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
		if !hasLower {
			return errors.New("password must contain at least one lowercase letter")
		}
	}

	// Check digit requirement
	if rules.RequireDigit {
		hasDigit := regexp.MustCompile(`\d`).MatchString(password)
		if !hasDigit {
			return errors.New("password must contain at least one digit")
		}
	}

	// Check special character requirement
	if rules.RequireSpecial {
		hasSpecial := regexp.MustCompile(`[` + regexp.QuoteMeta(rules.SpecialChars) + `]`).MatchString(password)
		if !hasSpecial {
			return fmt.Errorf("password must contain at least one special character: %s", rules.SpecialChars)
		}
	}

	return nil
}

// createUserInfo creates safe user info for responses.
func (s *Service) createUserInfo(user *models.User) *auth.UserInfo {
	// Use auth service to get user info with effective role
	return s.authService.CreateUserInfo(user)
}
