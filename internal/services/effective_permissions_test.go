package services

import (
	"context"
	"testing"

	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetEffectivePermissions must return permissions granted via a role, not just
// those attached directly to the user.
func TestGetEffectivePermissions_IncludesRoleDerived(t *testing.T) {
	db := setupTestDB(t)
	ps := NewPermissionService(db)
	ctx := context.Background()

	user := models.User{Email: "u@example.com", Username: "u", PasswordHash: "x", Role: "engineer"}
	require.NoError(t, db.Create(&user).Error)
	role := models.Role{Name: "engineer"}
	require.NoError(t, db.Create(&role).Error)
	permRole := models.Permission{Code: "ticket:write", Name: "Write tickets", Category: "tickets"}
	permDirect := models.Permission{Code: "knowledge:read", Name: "Read KB", Category: "knowledge"}
	require.NoError(t, db.Create(&permRole).Error)
	require.NoError(t, db.Create(&permDirect).Error)

	// role-derived grant
	require.NoError(t, db.Create(&models.UserRole{UserID: user.ID, RoleID: role.ID}).Error)
	require.NoError(t, db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: permRole.ID}).Error)
	// direct user grant
	require.NoError(t, db.Create(&models.UserPermission{UserID: user.ID, PermissionID: permDirect.ID}).Error)

	perms, err := ps.GetEffectivePermissions(ctx, user.ID)
	require.NoError(t, err)
	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		codes = append(codes, p.Code)
	}
	assert.ElementsMatch(t, []string{"ticket:write", "knowledge:read"}, codes)
}
