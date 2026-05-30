package mcp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/company/smartticket/internal/models"
)

// --- read handlers ---

func TestRBACListRoles(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:read"))

	roles := []models.Role{
		{BaseModel: models.BaseModel{ID: 1}, Name: "admin"},
		{BaseModel: models.BaseModel{ID: 2}, Name: "agent"},
	}
	mb.On("GetAllRoles", ctx).Return(roles, nil)

	out, summary, err := rbacListRoles(ctx, mb, rbacListRolesInput{})
	assert.NoError(t, err)
	assert.Len(t, out.Roles, 2)
	assert.Equal(t, 2, out.Total)
	assert.Equal(t, "Found 2 role(s).", summary)
	mb.AssertExpectations(t)
}

func TestRBACListRolesError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:read"))

	wantErr := errors.New("db error")
	mb.On("GetAllRoles", ctx).Return(nil, wantErr)

	_, _, err := rbacListRoles(ctx, mb, rbacListRolesInput{})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestRBACGetRole(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:read"))

	role := &models.Role{
		BaseModel:   models.BaseModel{ID: 7},
		Name:        "admin",
		Permissions: []models.Permission{{BaseModel: models.BaseModel{ID: 1}}},
	}
	mb.On("GetRoleByID", ctx, uint(7)).Return(role, nil)

	out, summary, err := rbacGetRole(ctx, mb, rbacGetRoleInput{ID: 7})
	assert.NoError(t, err)
	assert.Equal(t, uint(7), out.ID)
	assert.Equal(t, `Role #7 "admin" (1 permission(s)).`, summary)
	mb.AssertExpectations(t)
}

func TestRBACListPermissions(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:read"))

	perms := []models.Permission{
		{BaseModel: models.BaseModel{ID: 1}, Code: "ticket:read"},
		{BaseModel: models.BaseModel{ID: 2}, Code: "ticket:write"},
	}
	mb.On("GetAllPermissions", ctx).Return(perms, nil)

	out, summary, err := rbacListPermissions(ctx, mb, rbacListPermissionsInput{})
	assert.NoError(t, err)
	assert.Len(t, out.Permissions, 2)
	assert.Equal(t, 2, out.Total)
	assert.Equal(t, "Found 2 permission(s).", summary)
	mb.AssertExpectations(t)
}

func TestRBACGetPermission(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:read"))

	perm := &models.Permission{BaseModel: models.BaseModel{ID: 3}, Name: "Read Tickets", Code: "ticket:read"}
	mb.On("GetPermissionByID", ctx, uint(3)).Return(perm, nil)

	out, summary, err := rbacGetPermission(ctx, mb, rbacGetPermissionInput{ID: 3})
	assert.NoError(t, err)
	assert.Equal(t, uint(3), out.ID)
	assert.Equal(t, `Permission #3 "Read Tickets" (code: ticket:read).`, summary)
	mb.AssertExpectations(t)
}

func TestRBACGetUserRoles(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:read"))

	roles := []models.Role{{BaseModel: models.BaseModel{ID: 1}, Name: "admin"}}
	mb.On("GetUserRoles", ctx, uint(5)).Return(roles, nil)

	out, summary, err := rbacGetUserRoles(ctx, mb, rbacGetUserRolesInput{UserID: 5})
	assert.NoError(t, err)
	assert.Len(t, out.Roles, 1)
	assert.Equal(t, "User #5 has 1 role(s).", summary)
	mb.AssertExpectations(t)
}

func TestRBACGetUserPermissions(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:read"))

	perms := []models.Permission{
		{BaseModel: models.BaseModel{ID: 1}, Code: "ticket:read"},
		{BaseModel: models.BaseModel{ID: 2}, Code: "ticket:write"},
		{BaseModel: models.BaseModel{ID: 3}, Code: "user:read"},
	}
	mb.On("GetUserPermissions", ctx, uint(5)).Return(perms, nil)

	out, summary, err := rbacGetUserPermissions(ctx, mb, rbacGetUserPermissionsInput{UserID: 5})
	assert.NoError(t, err)
	assert.Len(t, out.Permissions, 3)
	assert.Equal(t, "User #5 has 3 effective permission(s).", summary)
	mb.AssertExpectations(t)
}

func TestRBACGetRolePermissions(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:read"))

	perms := []models.Permission{{BaseModel: models.BaseModel{ID: 1}, Code: "ticket:read"}}
	mb.On("GetRolePermissions", ctx, uint(2)).Return(perms, nil)

	out, summary, err := rbacGetRolePermissions(ctx, mb, rbacGetRolePermissionsInput{RoleID: 2})
	assert.NoError(t, err)
	assert.Len(t, out.Permissions, 1)
	assert.Equal(t, "Role #2 grants 1 permission(s).", summary)
	mb.AssertExpectations(t)
}

// --- write handlers ---

func TestRBACCreateRole(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	in := rbacCreateRoleInput{Name: "support", Description: "Support staff"}

	mb.On("CreateRole", ctx, mock.MatchedBy(func(r *models.Role) bool {
		return r.Name == "support" && r.Description == "Support staff" && r.IsActive
	})).Run(func(args mock.Arguments) {
		// simulate DB assigning an ID.
		args.Get(1).(*models.Role).ID = 42
	}).Return(nil)

	out, summary, err := rbacCreateRole(ctx, mb, in)
	assert.NoError(t, err)
	assert.Equal(t, uint(42), out.ID)
	assert.Equal(t, "support", out.Name)
	assert.Equal(t, `Created role #42 "support".`, summary)
	mb.AssertExpectations(t)
}

func TestRBACCreateRoleInactive(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	inactive := false
	in := rbacCreateRoleInput{Name: "frozen", IsActive: &inactive}

	mb.On("CreateRole", ctx, mock.MatchedBy(func(r *models.Role) bool {
		return r.Name == "frozen" && !r.IsActive
	})).Return(nil)

	_, _, err := rbacCreateRole(ctx, mb, in)
	assert.NoError(t, err)
	mb.AssertExpectations(t)
}

func TestRBACCreateRoleError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	wantErr := errors.New("duplicate role")
	mb.On("CreateRole", ctx, mock.Anything).Return(wantErr)

	_, _, err := rbacCreateRole(ctx, mb, rbacCreateRoleInput{Name: "x"})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestRBACUpdateRole(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	existing := &models.Role{BaseModel: models.BaseModel{ID: 8}, Name: "old", Description: "old desc", IsActive: true}
	mb.On("GetRoleByID", ctx, uint(8)).Return(existing, nil)

	newName := "new"
	inactive := false
	in := rbacUpdateRoleInput{ID: 8, Name: &newName, IsActive: &inactive}

	mb.On("UpdateRole", ctx, mock.MatchedBy(func(r *models.Role) bool {
		return r.ID == 8 && r.Name == "new" && r.Description == "old desc" && !r.IsActive
	})).Return(nil)

	out, summary, err := rbacUpdateRole(ctx, mb, in)
	assert.NoError(t, err)
	assert.Equal(t, "new", out.Name)
	assert.Equal(t, `Updated role #8 "new".`, summary)
	mb.AssertExpectations(t)
}

func TestRBACUpdateRoleGetError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	wantErr := errors.New("role not found")
	mb.On("GetRoleByID", ctx, uint(8)).Return(nil, wantErr)

	_, _, err := rbacUpdateRole(ctx, mb, rbacUpdateRoleInput{ID: 8})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestRBACDeleteRole(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	mb.On("DeleteRole", ctx, uint(9)).Return(nil)

	out, summary, err := rbacDeleteRole(ctx, mb, rbacDeleteRoleInput{ID: 9})
	assert.NoError(t, err)
	assert.Equal(t, uint(9), out.ID)
	assert.True(t, out.Deleted)
	assert.Equal(t, "Deleted role #9.", summary)
	mb.AssertExpectations(t)
}

func TestRBACDeleteRoleError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	wantErr := errors.New("system role")
	mb.On("DeleteRole", ctx, uint(9)).Return(wantErr)

	_, _, err := rbacDeleteRole(ctx, mb, rbacDeleteRoleInput{ID: 9})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestRBACCreatePermission(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	in := rbacCreatePermissionInput{
		Code:        "ticket:close",
		Name:        "Close Tickets",
		Category:    "tickets",
		Description: "Allow closing tickets",
	}

	mb.On("CreatePermission", ctx, mock.MatchedBy(func(p *models.Permission) bool {
		return p.Code == "ticket:close" && p.Name == "Close Tickets" &&
			p.Category == "tickets" && p.Description == "Allow closing tickets"
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*models.Permission).ID = 100
	}).Return(nil)

	out, summary, err := rbacCreatePermission(ctx, mb, in)
	assert.NoError(t, err)
	assert.Equal(t, uint(100), out.ID)
	assert.Equal(t, `Created permission #100 "Close Tickets" (code: ticket:close).`, summary)
	mb.AssertExpectations(t)
}

func TestRBACUpdatePermission(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	existing := &models.Permission{
		BaseModel: models.BaseModel{ID: 4},
		Code:      "ticket:read",
		Name:      "Read",
		Category:  "tickets",
	}
	mb.On("GetPermissionByID", ctx, uint(4)).Return(existing, nil)

	newName := "Read Tickets"
	in := rbacUpdatePermissionInput{ID: 4, Name: &newName}

	mb.On("UpdatePermission", ctx, mock.MatchedBy(func(p *models.Permission) bool {
		return p.ID == 4 && p.Name == "Read Tickets" && p.Code == "ticket:read" && p.Category == "tickets"
	})).Return(nil)

	out, summary, err := rbacUpdatePermission(ctx, mb, in)
	assert.NoError(t, err)
	assert.Equal(t, "Read Tickets", out.Name)
	assert.Equal(t, `Updated permission #4 "Read Tickets" (code: ticket:read).`, summary)
	mb.AssertExpectations(t)
}

func TestRBACDeletePermission(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	mb.On("DeletePermission", ctx, uint(11)).Return(nil)

	out, summary, err := rbacDeletePermission(ctx, mb, rbacDeletePermissionInput{ID: 11})
	assert.NoError(t, err)
	assert.Equal(t, uint(11), out.ID)
	assert.True(t, out.Deleted)
	assert.Equal(t, "Deleted permission #11.", summary)
	mb.AssertExpectations(t)
}

// --- assign/remove handlers ---

func TestRBACAssignRoleToUser(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	mb.On("AssignRoleToUser", ctx, uint(5), uint(2)).Return(nil)

	out, summary, err := rbacAssignRoleToUser(ctx, mb, rbacAssignRoleToUserInput{UserID: 5, RoleID: 2})
	assert.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "Assigned role #2 to user #5.", summary)
	mb.AssertExpectations(t)
}

func TestRBACAssignRoleToUserError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	wantErr := errors.New("user not found")
	mb.On("AssignRoleToUser", ctx, uint(5), uint(2)).Return(wantErr)

	_, _, err := rbacAssignRoleToUser(ctx, mb, rbacAssignRoleToUserInput{UserID: 5, RoleID: 2})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestRBACRemoveRoleFromUser(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	mb.On("RemoveRoleFromUser", ctx, uint(5), uint(2)).Return(nil)

	out, summary, err := rbacRemoveRoleFromUser(ctx, mb, rbacRemoveRoleFromUserInput{UserID: 5, RoleID: 2})
	assert.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "Removed role #2 from user #5.", summary)
	mb.AssertExpectations(t)
}

func TestRBACAssignPermissionToUser(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	mb.On("AssignPermissionToUser", ctx, uint(5), uint(3)).Return(nil)

	out, summary, err := rbacAssignPermissionToUser(ctx, mb, rbacAssignPermissionToUserInput{UserID: 5, PermissionID: 3})
	assert.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "Assigned permission #3 to user #5.", summary)
	mb.AssertExpectations(t)
}

func TestRBACRemovePermissionFromUser(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	mb.On("RemovePermissionFromUser", ctx, uint(5), uint(3)).Return(nil)

	out, summary, err := rbacRemovePermissionFromUser(ctx, mb, rbacRemovePermissionFromUserInput{UserID: 5, PermissionID: 3})
	assert.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "Removed permission #3 from user #5.", summary)
	mb.AssertExpectations(t)
}

func TestRBACAssignPermissionToRole(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	mb.On("AssignPermissionToRole", ctx, uint(2), uint(3)).Return(nil)

	out, summary, err := rbacAssignPermissionToRole(ctx, mb, rbacAssignPermissionToRoleInput{RoleID: 2, PermissionID: 3})
	assert.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "Assigned permission #3 to role #2.", summary)
	mb.AssertExpectations(t)
}

func TestRBACRemovePermissionFromRole(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("rbac:write"))

	mb.On("RemovePermissionFromRole", ctx, uint(2), uint(3)).Return(nil)

	out, summary, err := rbacRemovePermissionFromRole(ctx, mb, rbacRemovePermissionFromRoleInput{RoleID: 2, PermissionID: 3})
	assert.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "Removed permission #3 from role #2.", summary)
	mb.AssertExpectations(t)
}
