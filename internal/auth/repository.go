package auth

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

// Repository provides database operations for user management.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new authentication repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser creates a new user.
func (r *Repository) CreateUser(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by ID.
func (r *Repository) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email.
func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username.
func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// UpdateUser updates a user.
func (r *Repository) UpdateUser(user *models.User) error {
	// First check if user exists
	var existing models.User
	if err := r.db.Where("id = ?", user.ID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	// Update the user
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser soft deletes a user.
func (r *Repository) DeleteUser(userID uint) error {
	if err := r.db.Where("id = ?", userID).Delete(&models.User{}).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers retrieves a list of users with pagination and filtering.
func (r *Repository) ListUsers(page, pageSize int, filters map[string]interface{}) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.Model(&models.User{})

	// Apply filters
	if isActive, ok := filters["is_active"].(bool); ok {
		query = query.Where("is_active = ?", isActive)
	}
	if search, ok := filters["search"].(string); ok && search != "" {
		query = query.Where("email LIKE ? OR username LIKE ? OR first_name LIKE ? OR last_name LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// UpdateLastLogin updates the user's last login timestamp.
func (r *Repository) UpdateLastLogin(userID uint) error {
	now := time.Now()
	result := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to update last login: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdatePassword updates the user's password.
func (r *Repository) UpdatePassword(userID uint, passwordHash string) error {
	result := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("password_hash", passwordHash)

	if result.Error != nil {
		return fmt.Errorf("failed to update password: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// DeactivateUser deactivates a user account.
func (r *Repository) DeactivateUser(userID uint) error {
	if err := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}
	return nil
}

// ActivateUser activates a user account.
func (r *Repository) ActivateUser(userID uint) error {
	if err := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("is_active", true).Error; err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}
	return nil
}

// CheckEmailExists checks if an email already exists.
func (r *Repository) CheckEmailExists(email string, excludeUserID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&models.User{}).Where("email = ?", email)

	if len(excludeUserID) > 0 && excludeUserID[0] > 0 {
		query = query.Where("id != ?", excludeUserID[0])
	}

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return count > 0, nil
}

// CheckUsernameExists checks if a username already exists.
func (r *Repository) CheckUsernameExists(username string, excludeUserID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&models.User{}).Where("username = ?", username)

	if len(excludeUserID) > 0 && excludeUserID[0] > 0 {
		query = query.Where("id != ?", excludeUserID[0])
	}

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return count > 0, nil
}

// GetUserStats returns user statistics.
func (r *Repository) GetUserStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Total users
	var totalUsers int64
	if err := r.db.Model(&models.User{}).Count(&totalUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}
	stats["total_users"] = totalUsers

	// Active users
	var activeUsers int64
	if err := r.db.Model(&models.User{}).
		Where("is_active = ?", true).
		Count(&activeUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count active users: %w", err)
	}
	stats["active_users"] = activeUsers

	// Users who logged in last 30 days
	var recentUsers int64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := r.db.Model(&models.User{}).
		Where("is_active = ? AND last_login_at >= ?", true, thirtyDaysAgo).
		Count(&recentUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count recent users: %w", err)
	}
	stats["recent_users"] = recentUsers

	return stats, nil
}
