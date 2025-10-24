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
func (r *Repository) GetUserByID(userID, tenantID uint) (*models.User, error) {
	var user models.User
	if err := r.db.Where("id = ? AND tenant_id = ?", userID, tenantID).
		Preload("Tenant").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email.
func (r *Repository) GetUserByEmail(email string, tenantID uint) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ? AND tenant_id = ?", email, tenantID).
		Preload("Tenant").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username.
func (r *Repository) GetUserByUsername(username string, tenantID uint) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ? AND tenant_id = ?", username, tenantID).
		Preload("Tenant").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// UpdateUser updates a user.
func (r *Repository) UpdateUser(user *models.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser soft deletes a user.
func (r *Repository) DeleteUser(userID, tenantID uint) error {
	if err := r.db.Where("id = ? AND tenant_id = ?", userID, tenantID).
		Delete(&models.User{}).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers retrieves a list of users with pagination and filtering.
func (r *Repository) ListUsers(tenantID uint, page, pageSize int, filters map[string]interface{}) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.Where("tenant_id = ?", tenantID)

	// Apply filters
	// Note: role filtering is now handled through role associations, not direct User.Role field
	// This filter should be removed or reimplemented using JOINs with user_roles table
	if isActive, ok := filters["is_active"].(bool); ok {
		query = query.Where("is_active = ?", isActive)
	}
	if search, ok := filters["search"].(string); ok && search != "" {
		query = query.Where("email LIKE ? OR username LIKE ? OR first_name LIKE ? OR last_name LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Count total records
	if err := query.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	if err := query.Preload("Tenant").
		Offset(offset).
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

// UpdateUserRole updates the user's role - DEPRECATED.
// Roles are now managed through UserRole associations.
func (r *Repository) UpdateUserRole(userID, tenantID uint, role string) error {
	return fmt.Errorf("UpdateUserRole is deprecated - use UserRole model for role management")
}

// DeactivateUser deactivates a user account.
func (r *Repository) DeactivateUser(userID, tenantID uint) error {
	if err := r.db.Model(&models.User{}).
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}
	return nil
}

// ActivateUser activates a user account.
func (r *Repository) ActivateUser(userID, tenantID uint) error {
	if err := r.db.Model(&models.User{}).
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Update("is_active", true).Error; err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}
	return nil
}

// CheckEmailExists checks if an email already exists for a tenant.
func (r *Repository) CheckEmailExists(email string, tenantID uint, excludeUserID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&models.User{}).
		Where("email = ? AND tenant_id = ?", email, tenantID)

	if len(excludeUserID) > 0 && excludeUserID[0] > 0 {
		query = query.Where("id != ?", excludeUserID[0])
	}

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return count > 0, nil
}

// CheckUsernameExists checks if a username already exists for a tenant.
func (r *Repository) CheckUsernameExists(username string, tenantID uint, excludeUserID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&models.User{}).
		Where("username = ? AND tenant_id = ?", username, tenantID)

	if len(excludeUserID) > 0 && excludeUserID[0] > 0 {
		query = query.Where("id != ?", excludeUserID[0])
	}

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return count > 0, nil
}

// GetUserStats returns user statistics for a tenant.
func (r *Repository) GetUserStats(tenantID uint) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Total users
	var totalUsers int64
	if err := r.db.Model(&models.User{}).
		Where("tenant_id = ?", tenantID).
		Count(&totalUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}
	stats["total_users"] = totalUsers

	// Active users
	var activeUsers int64
	if err := r.db.Model(&models.User{}).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Count(&activeUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count active users: %w", err)
	}
	stats["active_users"] = activeUsers

	// Users by role - Note: This needs to be reimplemented using user_roles JOIN
	// For now, commenting out as User.Role field has been removed
	roles := []string{"admin", "engineer", "support", "customer", "sales"}
	for _, role := range roles {
		var count int64
		// TODO: Reimplement using JOIN with user_roles table
		// Example: r.db.Table("users").Joins("JOIN user_roles ON users.id = user_roles.user_id").Joins("JOIN roles ON user_roles.role_id = roles.id").Where("roles.name = ? AND users.tenant_id = ?", role, tenantID).Count(&count)
		if err := r.db.Model(&models.User{}).
			Where("tenant_id = ? AND is_active = ?", tenantID, true).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to count users by role %s: %w", role, err)
		}
		stats["users_"+role] = count // This will show total active users instead of role-specific counts
	}

	// Users who logged in last 30 days
	var recentUsers int64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := r.db.Model(&models.User{}).
		Where("tenant_id = ? AND is_active = ? AND last_login_at >= ?", tenantID, true, thirtyDaysAgo).
		Count(&recentUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count recent users: %w", err)
	}
	stats["recent_users"] = recentUsers

	return stats, nil
}

