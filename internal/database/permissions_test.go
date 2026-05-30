package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

func newPermTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Role{}, &models.Permission{}, &models.RolePermission{},
		&models.User{}, &models.UserRole{}, &models.UserPermission{},
	))
	return db
}

func TestEnsureRolesAndPermissions_IdempotentAndGrants(t *testing.T) {
	db := newPermTestDB(t)

	require.NoError(t, EnsureRolesAndPermissions(db))
	// Idempotent: a second run must not error or duplicate.
	require.NoError(t, EnsureRolesAndPermissions(db))

	var roleCount, permCount int64
	db.Model(&models.Role{}).Count(&roleCount)
	db.Model(&models.Permission{}).Count(&permCount)
	assert.Equal(t, int64(3), roleCount, "admin/engineer/customer")
	assert.Equal(t, int64(len(permissionCatalog)), permCount)

	// admin role is granted every permission code.
	var adminRole models.Role
	require.NoError(t, db.Where("name = ?", "admin").First(&adminRole).Error)
	var adminGrants int64
	db.Model(&models.RolePermission{}).Where("role_id = ?", adminRole.ID).Count(&adminGrants)
	assert.Equal(t, int64(len(permissionCatalog)), adminGrants)

	// customer role is granted only its limited set (no ticket:write missing).
	var customerRole models.Role
	require.NoError(t, db.Where("name = ?", "customer").First(&customerRole).Error)
	var custCodes []string
	db.Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", customerRole.ID).
		Pluck("permissions.code", &custCodes)
	assert.ElementsMatch(t, []string{"ticket:read", "ticket:write", "knowledge:read"}, custCodes)
}
