package auth

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

// Repository provides database operations for user management
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new authentication repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser creates a new user
func (r *Repository) CreateUser(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by ID
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

// GetUserByEmail retrieves a user by email
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

// GetUserByUsername retrieves a user by username
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

// UpdateUser updates a user
func (r *Repository) UpdateUser(user *models.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser soft deletes a user
func (r *Repository) DeleteUser(userID, tenantID uint) error {
	if err := r.db.Where("id = ? AND tenant_id = ?", userID, tenantID).
		Delete(&models.User{}).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers retrieves a list of users with pagination and filtering
func (r *Repository) ListUsers(tenantID uint, page, pageSize int, filters map[string]interface{}) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.Where("tenant_id = ?", tenantID)

	// Apply filters
	if role, ok := filters["role"].(string); ok && role != "" {
		query = query.Where("role = ?", role)
	}
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

// UpdateLastLogin updates the user's last login timestamp
func (r *Repository) UpdateLastLogin(userID uint) error {
	now := time.Now()
	if err := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", now).Error; err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// UpdatePassword updates the user's password
func (r *Repository) UpdatePassword(userID uint, passwordHash string) error {
	if err := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("password_hash", passwordHash).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// UpdateUserRole updates the user's role
func (r *Repository) UpdateUserRole(userID, tenantID uint, role string) error {
	if err := r.db.Model(&models.User{}).
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Update("role", role).Error; err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	return nil
}

// DeactivateUser deactivates a user account
func (r *Repository) DeactivateUser(userID, tenantID uint) error {
	if err := r.db.Model(&models.User{}).
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}
	return nil
}

// ActivateUser activates a user account
func (r *Repository) ActivateUser(userID, tenantID uint) error {
	if err := r.db.Model(&models.User{}).
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Update("is_active", true).Error; err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}
	return nil
}

// CheckEmailExists checks if an email already exists for a tenant
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

// CheckUsernameExists checks if a username already exists for a tenant
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

// GetUserStats returns user statistics for a tenant
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

	// Users by role
	roles := []string{"admin", "engineer", "support", "customer", "sales"}
	for _, role := range roles {
		var count int64
		if err := r.db.Model(&models.User{}).
			Where("tenant_id = ? AND role = ? AND is_active = ?", tenantID, role, true).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to count users by role %s: %w", role, err)
		}
		stats["users_"+role] = count
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

// CreateTenant creates a new tenant
func (r *Repository) CreateTenant(tenant *models.Tenant) error {
	if err := r.db.Create(tenant).Error; err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}
	return nil
}

// GetTenantByID retrieves a tenant by ID
func (r *Repository) GetTenantByID(tenantID uint) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return &tenant, nil
}

// GetTenantByDomain retrieves a tenant by domain
func (r *Repository) GetTenantByDomain(domain string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.Where("domain = ? AND is_active = ?", domain, true).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return &tenant, nil
}
