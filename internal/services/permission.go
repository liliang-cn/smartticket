// Package services provides business logic and service layer implementations for the SmartTicket platform.
// It includes services for user management, permission handling, and core business operations.
package services

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

// PermissionService provides permission checking and role management functionality
type PermissionService struct {
	db *gorm.DB
}

// NewPermissionService creates a new permission service
func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{
		db: db,
	}
}

// GetDatabase returns the underlying database connection
func (ps *PermissionService) GetDatabase() *gorm.DB {
	return ps.db
}

// GetUserPermissions returns all permissions assigned to a user directly
func (ps *PermissionService) GetUserPermissions(ctx context.Context, userID uint, _ string) ([]models.Permission, error) {
	var permissions []models.Permission

	err := ps.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN user_permissions ON permissions.id = user_permissions.permission_id").
		Where("user_permissions.user_id = ?", userID).
		Find(&permissions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	return permissions, nil
}

// GetUserRoles returns all roles assigned to a user
func (ps *PermissionService) GetUserRoles(ctx context.Context, userID uint, tenantID string) ([]models.Role, error) {
	var roles []models.Role

	err := ps.db.WithContext(ctx).
		Preload("Permissions").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.tenant_id = ?", userID, tenantID).
		Find(&roles).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return roles, nil
}

// GetRolePermissions returns all permissions for a role
func (ps *PermissionService) GetRolePermissions(ctx context.Context, roleID uint) ([]models.Permission, error) {
	var permissions []models.Permission

	err := ps.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	return permissions, nil
}

// GetAllPermissions returns all permissions in the system
func (ps *PermissionService) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	var permissions []models.Permission

	err := ps.db.WithContext(ctx).Order("category, code").Find(&permissions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all permissions: %w", err)
	}

	return permissions, nil
}

// GetAllRoles returns all roles in the system
func (ps *PermissionService) GetAllRoles(ctx context.Context, tenantID string) ([]models.Role, error) {
	var roles []models.Role

	query := ps.db.WithContext(ctx)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	err := query.Preload("Permissions").Order("name").Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all roles: %w", err)
	}

	return roles, nil
}

// CreatePermission creates a new permission
func (ps *PermissionService) CreatePermission(ctx context.Context, permission *models.Permission) error {
	err := ps.db.WithContext(ctx).Create(permission).Error
	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	return nil
}

// CreateRole creates a new role
func (ps *PermissionService) CreateRole(ctx context.Context, role *models.Role) error {
	err := ps.db.WithContext(ctx).Create(role).Error
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// AssignPermissionToRole assigns a permission to a role
func (ps *PermissionService) AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error {
	rolePermission := &models.RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
	}

	err := ps.db.WithContext(ctx).Create(rolePermission).Error
	if err != nil {
		return fmt.Errorf("failed to assign permission to role: %w", err)
	}

	return nil
}

// RemovePermissionFromRole removes a permission from a role
func (ps *PermissionService) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error {
	err := ps.db.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Unscoped().
		Delete(&models.RolePermission{}).Error

	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}

	return nil
}

// AssignRoleToUser assigns a role to a user
func (ps *PermissionService) AssignRoleToUser(ctx context.Context, userID uint, roleID uint, _ string) error {
	userRole := &models.UserRole{
		UserID:     userID,
		RoleID:     roleID,
		AssignedAt: time.Now(),
		AssignedBy: userID, // For now, assign by self
	}

	err := ps.db.WithContext(ctx).Create(userRole).Error
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	return nil
}

// RemoveRoleFromUser removes a role from a user
func (ps *PermissionService) RemoveRoleFromUser(ctx context.Context, userID, roleID uint, _ string) error {
	err := ps.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&models.UserRole{}).Error

	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	return nil
}

// AssignPermissionToUser assigns a permission directly to a user
func (ps *PermissionService) AssignPermissionToUser(ctx context.Context, userID, permissionID uint, _ string) error {
	userPermission := &models.UserPermission{
		UserID:       userID,
		PermissionID: permissionID,
		GrantedBy:    userID, // For now, granted by self
	}

	err := ps.db.WithContext(ctx).Create(userPermission).Error
	if err != nil {
		return fmt.Errorf("failed to assign permission to user: %w", err)
	}

	return nil
}

// RemovePermissionFromUser removes a permission directly from a user
func (ps *PermissionService) RemovePermissionFromUser(ctx context.Context, userID, permissionID uint, tenantID string) error {
	err := ps.db.WithContext(ctx).
		Where("user_id = ? AND permission_id = ?", userID, permissionID).
		Delete(&models.UserPermission{}).Error

	if err != nil {
		return fmt.Errorf("failed to remove permission from user: %w", err)
	}

	return nil
}

// HasPermission checks if a user has a specific permission (either directly or through roles)
func (ps *PermissionService) HasPermission(ctx context.Context, userID uint, tenantID string, permissionCode string) (bool, error) {
	// Check direct user permissions
	var count int64
	err := ps.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN user_permissions ON permissions.id = user_permissions.permission_id").
		Where("user_permissions.user_id = ? AND permissions.code = ?",
			userID, permissionCode).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check user permission: %w", err)
	}

	if count > 0 {
		return true, nil
	}

	// Check role permissions
	err = ps.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Joins("JOIN roles ON role_permissions.role_id = roles.id").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.tenant_id = ? AND permissions.code = ?",
			userID, tenantID, permissionCode).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check role permission: %w", err)
	}

	return count > 0, nil
}

// GetPermissionByID returns a permission by ID
func (ps *PermissionService) GetPermissionByID(ctx context.Context, id uint) (*models.Permission, error) {
	var permission models.Permission

	err := ps.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &permission, nil
}

// GetRoleByID returns a role by ID
func (ps *PermissionService) GetRoleByID(ctx context.Context, id uint) (*models.Role, error) {
	var role models.Role

	err := ps.db.WithContext(ctx).
		Preload("Permissions").
		First(&role, id).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

// UpdatePermission updates an existing permission
func (ps *PermissionService) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	err := ps.db.WithContext(ctx).Save(permission).Error
	if err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}

	return nil
}

// UpdateRole updates an existing role
func (ps *PermissionService) UpdateRole(ctx context.Context, role *models.Role) error {
	err := ps.db.WithContext(ctx).Save(role).Error
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// DeletePermission deletes a permission (if not system permission)
func (ps *PermissionService) DeletePermission(ctx context.Context, id uint) error {
	// Check if it's a system permission
	var permission models.Permission
	err := ps.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return fmt.Errorf("failed to get permission: %w", err)
	}

	if permission.IsSystem {
		return fmt.Errorf("cannot delete system permission")
	}

	// Delete role assignments
	err = ps.db.WithContext(ctx).
		Where("permission_id = ?", id).
		Delete(&models.RolePermission{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete role permission assignments: %w", err)
	}

	// Delete user assignments
	err = ps.db.WithContext(ctx).
		Where("permission_id = ?", id).
		Delete(&models.UserPermission{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete user permission assignments: %w", err)
	}

	// Delete permission
	err = ps.db.WithContext(ctx).Delete(&models.Permission{}, id).Error
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	return nil
}

// DeleteRole deletes a role (if not system role)
func (ps *PermissionService) DeleteRole(ctx context.Context, id uint) error {
	// Check if it's a system role
	var role models.Role
	err := ps.db.WithContext(ctx).First(&role, id).Error
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	if role.IsSystem {
		return fmt.Errorf("cannot delete system role")
	}

	// Delete role permissions
	err = ps.db.WithContext(ctx).
		Where("role_id = ?", id).
		Delete(&models.RolePermission{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete role permissions: %w", err)
	}

	// Delete user roles
	err = ps.db.WithContext(ctx).
		Where("role_id = ?", id).
		Delete(&models.UserRole{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete user roles: %w", err)
	}

	// Delete role
	err = ps.db.WithContext(ctx).Delete(&models.Role{}, id).Error
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}
