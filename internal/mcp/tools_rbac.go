package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/models"
)

// ----------------------------------------------------------------------------
// Cycle-safe output views.
//
// models.Role and models.Permission cannot be used directly as MCP tool Output
// types: they carry GORM association fields (Role.Permissions []Permission ↔
// Permission.RoleAssignments / Role.Creator *User → Product↔Service↔Ticket↔User)
// that form reference cycles the SDK's JSON-schema reflector rejects with
// "cycle detected". The rbacRoleView / rbacPermissionView types below are flat
// DTOs that surface only scalar fields plus a one-level-deep, terminal list of
// permission views, with associations otherwise reduced to IDs/counts.
// ----------------------------------------------------------------------------

// rbacPermissionView is the cycle-safe MCP output view of a models.Permission.
type rbacPermissionView struct {
	ID          uint      `json:"id" jsonschema:"the permission's numeric ID"`
	Code        string    `json:"code" jsonschema:"the permission code in resource:action format"`
	Name        string    `json:"name" jsonschema:"the human-readable permission name"`
	Description string    `json:"description,omitempty" jsonschema:"the permission description"`
	Category    string    `json:"category" jsonschema:"the permission category, e.g. tickets, users, knowledge"`
	IsSystem    bool      `json:"is_system" jsonschema:"whether this is a system permission that cannot be deleted"`
	CreatedAt   time.Time `json:"created_at" jsonschema:"when the permission was created"`
	UpdatedAt   time.Time `json:"updated_at" jsonschema:"when the permission was last updated"`
}

// rbacPermissionViewFrom converts a models.Permission into its cycle-safe view.
func rbacPermissionViewFrom(p *models.Permission) rbacPermissionView {
	return rbacPermissionView{
		ID:          p.ID,
		Code:        p.Code,
		Name:        p.Name,
		Description: p.Description,
		Category:    p.Category,
		IsSystem:    p.IsSystem,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// rbacPermissionViewsFrom converts a slice of models.Permission into views.
func rbacPermissionViewsFrom(perms []models.Permission) []rbacPermissionView {
	views := make([]rbacPermissionView, len(perms))
	for i := range perms {
		views[i] = rbacPermissionViewFrom(&perms[i])
	}
	return views
}

// rbacRoleView is the cycle-safe MCP output view of a models.Role. The embedded
// Permissions are flattened to terminal rbacPermissionView values, and the
// *User Creator association is reduced to its CreatedBy numeric ID.
type rbacRoleView struct {
	ID              uint                 `json:"id" jsonschema:"the role's numeric ID"`
	Name            string               `json:"name" jsonschema:"the role name"`
	Description     string               `json:"description,omitempty" jsonschema:"the role description"`
	IsSystem        bool                 `json:"is_system" jsonschema:"whether this is a system role that cannot be deleted"`
	IsActive        bool                 `json:"is_active" jsonschema:"whether the role is active"`
	CreatedBy       uint                 `json:"created_by,omitempty" jsonschema:"numeric ID of the user who created the role"`
	Permissions     []rbacPermissionView `json:"permissions,omitempty" jsonschema:"the permissions granted to this role"`
	PermissionCount int                  `json:"permission_count" jsonschema:"number of permissions granted to this role"`
	CreatedAt       time.Time            `json:"created_at" jsonschema:"when the role was created"`
	UpdatedAt       time.Time            `json:"updated_at" jsonschema:"when the role was last updated"`
}

// rbacRoleViewFrom converts a models.Role into its cycle-safe view.
func rbacRoleViewFrom(r *models.Role) rbacRoleView {
	return rbacRoleView{
		ID:              r.ID,
		Name:            r.Name,
		Description:     r.Description,
		IsSystem:        r.IsSystem,
		IsActive:        r.IsActive,
		CreatedBy:       r.CreatedBy,
		Permissions:     rbacPermissionViewsFrom(r.Permissions),
		PermissionCount: len(r.Permissions),
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// rbacRoleViewsFrom converts a slice of models.Role into views.
func rbacRoleViewsFrom(roles []models.Role) []rbacRoleView {
	views := make([]rbacRoleView, len(roles))
	for i := range roles {
		views[i] = rbacRoleViewFrom(&roles[i])
	}
	return views
}

// ----------------------------------------------------------------------------
// RBAC-domain MCP tools.
//
// These tools cover role/permission CRUD plus the assign/remove operations that
// wire users, roles, and permissions together, and the read operations that list
// or look them up. Each tool declares its own MCP-specific Input struct (json +
// jsonschema tags), translates it into the models / Backend arguments, and
// delegates to the Backend. Most RBAC Backend methods take a context.Context as
// their first argument; the closures forward the handler's ctx unchanged.
//
// All identifiers carry an "rbac" prefix to avoid collisions with sibling
// domains in this package. See server.go for the conventions and auth_whoami
// reference implementation.
// ----------------------------------------------------------------------------

// --- Input schemas: reads ---

// rbacListRolesInput is the MCP input schema for rbac_list_roles. No arguments.
type rbacListRolesInput struct{}

// rbacGetRoleInput is the MCP input schema for rbac_get_role.
type rbacGetRoleInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the role to retrieve"`
}

// rbacListPermissionsInput is the MCP input schema for rbac_list_permissions. No arguments.
type rbacListPermissionsInput struct{}

// rbacGetPermissionInput is the MCP input schema for rbac_get_permission.
type rbacGetPermissionInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the permission to retrieve"`
}

// rbacGetUserRolesInput is the MCP input schema for rbac_get_user_roles.
type rbacGetUserRolesInput struct {
	UserID uint `json:"user_id" jsonschema:"the numeric ID of the user whose roles to list"`
}

// rbacGetUserPermissionsInput is the MCP input schema for rbac_get_user_permissions.
type rbacGetUserPermissionsInput struct {
	UserID uint `json:"user_id" jsonschema:"the numeric ID of the user whose effective permissions to list"`
}

// rbacGetRolePermissionsInput is the MCP input schema for rbac_get_role_permissions.
type rbacGetRolePermissionsInput struct {
	RoleID uint `json:"role_id" jsonschema:"the numeric ID of the role whose permissions to list"`
}

// --- Input schemas: role/permission writes ---

// rbacCreateRoleInput is the MCP input schema for rbac_create_role.
type rbacCreateRoleInput struct {
	Name        string `json:"name" jsonschema:"the role name (required, up to 100 characters)"`
	Description string `json:"description,omitempty" jsonschema:"optional human-readable description of the role"`
	IsActive    *bool  `json:"is_active,omitempty" jsonschema:"whether the role is active (defaults to true)"`
}

// rbacUpdateRoleInput is the MCP input schema for rbac_update_role.
type rbacUpdateRoleInput struct {
	ID          uint    `json:"id" jsonschema:"the numeric ID of the role to update"`
	Name        *string `json:"name,omitempty" jsonschema:"new role name (omit to leave unchanged)"`
	Description *string `json:"description,omitempty" jsonschema:"new description (omit to leave unchanged)"`
	IsActive    *bool   `json:"is_active,omitempty" jsonschema:"new active state (omit to leave unchanged)"`
}

// rbacDeleteRoleInput is the MCP input schema for rbac_delete_role.
type rbacDeleteRoleInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the role to delete"`
}

// rbacCreatePermissionInput is the MCP input schema for rbac_create_permission.
type rbacCreatePermissionInput struct {
	Code        string `json:"code" jsonschema:"the permission code in resource:action format (required, unique)"`
	Name        string `json:"name" jsonschema:"the human-readable permission name (required)"`
	Category    string `json:"category" jsonschema:"the permission category, e.g. tickets, users, knowledge (required)"`
	Description string `json:"description,omitempty" jsonschema:"optional human-readable description"`
}

// rbacUpdatePermissionInput is the MCP input schema for rbac_update_permission.
type rbacUpdatePermissionInput struct {
	ID          uint    `json:"id" jsonschema:"the numeric ID of the permission to update"`
	Code        *string `json:"code,omitempty" jsonschema:"new permission code in resource:action format (omit to leave unchanged)"`
	Name        *string `json:"name,omitempty" jsonschema:"new permission name (omit to leave unchanged)"`
	Category    *string `json:"category,omitempty" jsonschema:"new category (omit to leave unchanged)"`
	Description *string `json:"description,omitempty" jsonschema:"new description (omit to leave unchanged)"`
}

// rbacDeletePermissionInput is the MCP input schema for rbac_delete_permission.
type rbacDeletePermissionInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the permission to delete"`
}

// --- Input schemas: assign/remove ---

// rbacAssignRoleToUserInput is the MCP input schema for rbac_assign_role_to_user.
type rbacAssignRoleToUserInput struct {
	UserID uint `json:"user_id" jsonschema:"the numeric ID of the user"`
	RoleID uint `json:"role_id" jsonschema:"the numeric ID of the role to assign"`
}

// rbacRemoveRoleFromUserInput is the MCP input schema for rbac_remove_role_from_user.
type rbacRemoveRoleFromUserInput struct {
	UserID uint `json:"user_id" jsonschema:"the numeric ID of the user"`
	RoleID uint `json:"role_id" jsonschema:"the numeric ID of the role to remove"`
}

// rbacAssignPermissionToUserInput is the MCP input schema for rbac_assign_permission_to_user.
type rbacAssignPermissionToUserInput struct {
	UserID       uint `json:"user_id" jsonschema:"the numeric ID of the user"`
	PermissionID uint `json:"permission_id" jsonschema:"the numeric ID of the permission to assign directly to the user"`
}

// rbacRemovePermissionFromUserInput is the MCP input schema for rbac_remove_permission_from_user.
type rbacRemovePermissionFromUserInput struct {
	UserID       uint `json:"user_id" jsonschema:"the numeric ID of the user"`
	PermissionID uint `json:"permission_id" jsonschema:"the numeric ID of the permission to remove from the user"`
}

// rbacAssignPermissionToRoleInput is the MCP input schema for rbac_assign_permission_to_role.
type rbacAssignPermissionToRoleInput struct {
	RoleID       uint `json:"role_id" jsonschema:"the numeric ID of the role"`
	PermissionID uint `json:"permission_id" jsonschema:"the numeric ID of the permission to assign to the role"`
}

// rbacRemovePermissionFromRoleInput is the MCP input schema for rbac_remove_permission_from_role.
type rbacRemovePermissionFromRoleInput struct {
	RoleID       uint `json:"role_id" jsonschema:"the numeric ID of the role"`
	PermissionID uint `json:"permission_id" jsonschema:"the numeric ID of the permission to remove from the role"`
}

// --- Output schemas ---

// rbacRolesOutput is the structured output for tools returning a list of roles.
type rbacRolesOutput struct {
	Roles []rbacRoleView `json:"roles,omitempty" jsonschema:"the matching roles"`
	Total int            `json:"total" jsonschema:"the number of roles returned"`
}

// rbacPermissionsOutput is the structured output for tools returning a list of permissions.
type rbacPermissionsOutput struct {
	Permissions []rbacPermissionView `json:"permissions,omitempty" jsonschema:"the matching permissions"`
	Total       int                  `json:"total" jsonschema:"the number of permissions returned"`
}

// rbacDeleteOutput is the structured output of rbac_delete_role and rbac_delete_permission.
type rbacDeleteOutput struct {
	ID      uint `json:"id" jsonschema:"the ID of the deleted entity"`
	Deleted bool `json:"deleted" jsonschema:"whether the entity was deleted"`
}

// rbacAssignmentOutput is the structured output of the assign/remove tools.
type rbacAssignmentOutput struct {
	Success bool `json:"success" jsonschema:"whether the assignment operation succeeded"`
}

// registerRBACTools registers the RBAC-domain MCP tools.
// See server.go for the tool implementation conventions and auth_whoami template.
func registerRBACTools(s *mcp.Server, b Backend) {
	// --- reads (rbac:read) ---

	registerTool(s,
		"rbac_list_roles",
		"List all defined roles.",
		"rbac:read",
		func(ctx context.Context, in rbacListRolesInput) (rbacRolesOutput, string, error) {
			return rbacListRoles(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_get_role",
		"Retrieve a single role by ID, including its permissions.",
		"rbac:read",
		func(ctx context.Context, in rbacGetRoleInput) (rbacRoleView, string, error) {
			return rbacGetRole(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_list_permissions",
		"List all defined permissions.",
		"rbac:read",
		func(ctx context.Context, in rbacListPermissionsInput) (rbacPermissionsOutput, string, error) {
			return rbacListPermissions(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_get_permission",
		"Retrieve a single permission by ID.",
		"rbac:read",
		func(ctx context.Context, in rbacGetPermissionInput) (rbacPermissionView, string, error) {
			return rbacGetPermission(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_get_user_roles",
		"List the roles assigned to a user.",
		"rbac:read",
		func(ctx context.Context, in rbacGetUserRolesInput) (rbacRolesOutput, string, error) {
			return rbacGetUserRoles(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_get_user_permissions",
		"List the effective permissions held by a user (via roles and direct grants).",
		"rbac:read",
		func(ctx context.Context, in rbacGetUserPermissionsInput) (rbacPermissionsOutput, string, error) {
			return rbacGetUserPermissions(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_get_role_permissions",
		"List the permissions granted to a role.",
		"rbac:read",
		func(ctx context.Context, in rbacGetRolePermissionsInput) (rbacPermissionsOutput, string, error) {
			return rbacGetRolePermissions(ctx, b, in)
		},
	)

	// --- writes (rbac:write) ---

	registerTool(s,
		"rbac_create_role",
		"Create a new role.",
		"rbac:write",
		func(ctx context.Context, in rbacCreateRoleInput) (rbacRoleView, string, error) {
			return rbacCreateRole(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_update_role",
		"Update an existing role; omitted fields are left unchanged.",
		"rbac:write",
		func(ctx context.Context, in rbacUpdateRoleInput) (rbacRoleView, string, error) {
			return rbacUpdateRole(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_delete_role",
		"Delete a role by ID.",
		"rbac:write",
		func(ctx context.Context, in rbacDeleteRoleInput) (rbacDeleteOutput, string, error) {
			return rbacDeleteRole(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_create_permission",
		"Create a new permission.",
		"rbac:write",
		func(ctx context.Context, in rbacCreatePermissionInput) (rbacPermissionView, string, error) {
			return rbacCreatePermission(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_update_permission",
		"Update an existing permission; omitted fields are left unchanged.",
		"rbac:write",
		func(ctx context.Context, in rbacUpdatePermissionInput) (rbacPermissionView, string, error) {
			return rbacUpdatePermission(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_delete_permission",
		"Delete a permission by ID.",
		"rbac:write",
		func(ctx context.Context, in rbacDeletePermissionInput) (rbacDeleteOutput, string, error) {
			return rbacDeletePermission(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_assign_role_to_user",
		"Assign a role to a user.",
		"rbac:write",
		func(ctx context.Context, in rbacAssignRoleToUserInput) (rbacAssignmentOutput, string, error) {
			return rbacAssignRoleToUser(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_remove_role_from_user",
		"Remove a role from a user.",
		"rbac:write",
		func(ctx context.Context, in rbacRemoveRoleFromUserInput) (rbacAssignmentOutput, string, error) {
			return rbacRemoveRoleFromUser(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_assign_permission_to_user",
		"Assign a permission directly to a user.",
		"rbac:write",
		func(ctx context.Context, in rbacAssignPermissionToUserInput) (rbacAssignmentOutput, string, error) {
			return rbacAssignPermissionToUser(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_remove_permission_from_user",
		"Remove a directly-granted permission from a user.",
		"rbac:write",
		func(ctx context.Context, in rbacRemovePermissionFromUserInput) (rbacAssignmentOutput, string, error) {
			return rbacRemovePermissionFromUser(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_assign_permission_to_role",
		"Assign a permission to a role.",
		"rbac:write",
		func(ctx context.Context, in rbacAssignPermissionToRoleInput) (rbacAssignmentOutput, string, error) {
			return rbacAssignPermissionToRole(ctx, b, in)
		},
	)

	registerTool(s,
		"rbac_remove_permission_from_role",
		"Remove a permission from a role.",
		"rbac:write",
		func(ctx context.Context, in rbacRemovePermissionFromRoleInput) (rbacAssignmentOutput, string, error) {
			return rbacRemovePermissionFromRole(ctx, b, in)
		},
	)
}

// --- read handlers ---

// rbacListRoles handles rbac_list_roles.
func rbacListRoles(ctx context.Context, b Backend, _ rbacListRolesInput) (rbacRolesOutput, string, error) {
	roles, err := b.GetAllRoles(ctx)
	if err != nil {
		return rbacRolesOutput{}, "", err
	}
	out := rbacRolesOutput{Roles: rbacRoleViewsFrom(roles), Total: len(roles)}
	return out, fmt.Sprintf("Found %d role(s).", len(roles)), nil
}

// rbacGetRole handles rbac_get_role.
func rbacGetRole(ctx context.Context, b Backend, in rbacGetRoleInput) (rbacRoleView, string, error) {
	role, err := b.GetRoleByID(ctx, in.ID)
	if err != nil {
		return rbacRoleView{}, "", err
	}
	return rbacRoleViewFrom(role), fmt.Sprintf("Role #%d %q (%d permission(s)).", role.ID, role.Name, len(role.Permissions)), nil
}

// rbacListPermissions handles rbac_list_permissions.
func rbacListPermissions(ctx context.Context, b Backend, _ rbacListPermissionsInput) (rbacPermissionsOutput, string, error) {
	perms, err := b.GetAllPermissions(ctx)
	if err != nil {
		return rbacPermissionsOutput{}, "", err
	}
	out := rbacPermissionsOutput{Permissions: rbacPermissionViewsFrom(perms), Total: len(perms)}
	return out, fmt.Sprintf("Found %d permission(s).", len(perms)), nil
}

// rbacGetPermission handles rbac_get_permission.
func rbacGetPermission(ctx context.Context, b Backend, in rbacGetPermissionInput) (rbacPermissionView, string, error) {
	perm, err := b.GetPermissionByID(ctx, in.ID)
	if err != nil {
		return rbacPermissionView{}, "", err
	}
	return rbacPermissionViewFrom(perm), fmt.Sprintf("Permission #%d %q (code: %s).", perm.ID, perm.Name, perm.Code), nil
}

// rbacGetUserRoles handles rbac_get_user_roles.
func rbacGetUserRoles(ctx context.Context, b Backend, in rbacGetUserRolesInput) (rbacRolesOutput, string, error) {
	roles, err := b.GetUserRoles(ctx, in.UserID)
	if err != nil {
		return rbacRolesOutput{}, "", err
	}
	out := rbacRolesOutput{Roles: rbacRoleViewsFrom(roles), Total: len(roles)}
	return out, fmt.Sprintf("User #%d has %d role(s).", in.UserID, len(roles)), nil
}

// rbacGetUserPermissions handles rbac_get_user_permissions.
func rbacGetUserPermissions(ctx context.Context, b Backend, in rbacGetUserPermissionsInput) (rbacPermissionsOutput, string, error) {
	perms, err := b.GetUserPermissions(ctx, in.UserID)
	if err != nil {
		return rbacPermissionsOutput{}, "", err
	}
	out := rbacPermissionsOutput{Permissions: rbacPermissionViewsFrom(perms), Total: len(perms)}
	return out, fmt.Sprintf("User #%d has %d effective permission(s).", in.UserID, len(perms)), nil
}

// rbacGetRolePermissions handles rbac_get_role_permissions.
func rbacGetRolePermissions(ctx context.Context, b Backend, in rbacGetRolePermissionsInput) (rbacPermissionsOutput, string, error) {
	perms, err := b.GetRolePermissions(ctx, in.RoleID)
	if err != nil {
		return rbacPermissionsOutput{}, "", err
	}
	out := rbacPermissionsOutput{Permissions: rbacPermissionViewsFrom(perms), Total: len(perms)}
	return out, fmt.Sprintf("Role #%d grants %d permission(s).", in.RoleID, len(perms)), nil
}

// --- write handlers ---

// rbacCreateRole handles rbac_create_role.
func rbacCreateRole(ctx context.Context, b Backend, in rbacCreateRoleInput) (rbacRoleView, string, error) {
	role := &models.Role{
		Name:        in.Name,
		Description: in.Description,
		IsActive:    true,
	}
	if in.IsActive != nil {
		role.IsActive = *in.IsActive
	}

	if err := b.CreateRole(ctx, role); err != nil {
		return rbacRoleView{}, "", err
	}
	return rbacRoleViewFrom(role), fmt.Sprintf("Created role #%d %q.", role.ID, role.Name), nil
}

// rbacUpdateRole handles rbac_update_role. It loads the current role, applies the
// provided fields, and persists the result.
func rbacUpdateRole(ctx context.Context, b Backend, in rbacUpdateRoleInput) (rbacRoleView, string, error) {
	role, err := b.GetRoleByID(ctx, in.ID)
	if err != nil {
		return rbacRoleView{}, "", err
	}

	if in.Name != nil {
		role.Name = *in.Name
	}
	if in.Description != nil {
		role.Description = *in.Description
	}
	if in.IsActive != nil {
		role.IsActive = *in.IsActive
	}

	if err := b.UpdateRole(ctx, role); err != nil {
		return rbacRoleView{}, "", err
	}
	return rbacRoleViewFrom(role), fmt.Sprintf("Updated role #%d %q.", role.ID, role.Name), nil
}

// rbacDeleteRole handles rbac_delete_role.
func rbacDeleteRole(ctx context.Context, b Backend, in rbacDeleteRoleInput) (rbacDeleteOutput, string, error) {
	if err := b.DeleteRole(ctx, in.ID); err != nil {
		return rbacDeleteOutput{}, "", err
	}
	return rbacDeleteOutput{ID: in.ID, Deleted: true}, fmt.Sprintf("Deleted role #%d.", in.ID), nil
}

// rbacCreatePermission handles rbac_create_permission.
func rbacCreatePermission(ctx context.Context, b Backend, in rbacCreatePermissionInput) (rbacPermissionView, string, error) {
	perm := &models.Permission{
		Code:        in.Code,
		Name:        in.Name,
		Category:    in.Category,
		Description: in.Description,
	}

	if err := b.CreatePermission(ctx, perm); err != nil {
		return rbacPermissionView{}, "", err
	}
	return rbacPermissionViewFrom(perm), fmt.Sprintf("Created permission #%d %q (code: %s).", perm.ID, perm.Name, perm.Code), nil
}

// rbacUpdatePermission handles rbac_update_permission. It loads the current
// permission, applies the provided fields, and persists the result.
func rbacUpdatePermission(ctx context.Context, b Backend, in rbacUpdatePermissionInput) (rbacPermissionView, string, error) {
	perm, err := b.GetPermissionByID(ctx, in.ID)
	if err != nil {
		return rbacPermissionView{}, "", err
	}

	if in.Code != nil {
		perm.Code = *in.Code
	}
	if in.Name != nil {
		perm.Name = *in.Name
	}
	if in.Category != nil {
		perm.Category = *in.Category
	}
	if in.Description != nil {
		perm.Description = *in.Description
	}

	if err := b.UpdatePermission(ctx, perm); err != nil {
		return rbacPermissionView{}, "", err
	}
	return rbacPermissionViewFrom(perm), fmt.Sprintf("Updated permission #%d %q (code: %s).", perm.ID, perm.Name, perm.Code), nil
}

// rbacDeletePermission handles rbac_delete_permission.
func rbacDeletePermission(ctx context.Context, b Backend, in rbacDeletePermissionInput) (rbacDeleteOutput, string, error) {
	if err := b.DeletePermission(ctx, in.ID); err != nil {
		return rbacDeleteOutput{}, "", err
	}
	return rbacDeleteOutput{ID: in.ID, Deleted: true}, fmt.Sprintf("Deleted permission #%d.", in.ID), nil
}

// --- assign/remove handlers ---

// rbacAssignRoleToUser handles rbac_assign_role_to_user.
func rbacAssignRoleToUser(ctx context.Context, b Backend, in rbacAssignRoleToUserInput) (rbacAssignmentOutput, string, error) {
	if err := b.AssignRoleToUser(ctx, in.UserID, in.RoleID); err != nil {
		return rbacAssignmentOutput{}, "", err
	}
	return rbacAssignmentOutput{Success: true}, fmt.Sprintf("Assigned role #%d to user #%d.", in.RoleID, in.UserID), nil
}

// rbacRemoveRoleFromUser handles rbac_remove_role_from_user.
func rbacRemoveRoleFromUser(ctx context.Context, b Backend, in rbacRemoveRoleFromUserInput) (rbacAssignmentOutput, string, error) {
	if err := b.RemoveRoleFromUser(ctx, in.UserID, in.RoleID); err != nil {
		return rbacAssignmentOutput{}, "", err
	}
	return rbacAssignmentOutput{Success: true}, fmt.Sprintf("Removed role #%d from user #%d.", in.RoleID, in.UserID), nil
}

// rbacAssignPermissionToUser handles rbac_assign_permission_to_user.
func rbacAssignPermissionToUser(ctx context.Context, b Backend, in rbacAssignPermissionToUserInput) (rbacAssignmentOutput, string, error) {
	if err := b.AssignPermissionToUser(ctx, in.UserID, in.PermissionID); err != nil {
		return rbacAssignmentOutput{}, "", err
	}
	return rbacAssignmentOutput{Success: true}, fmt.Sprintf("Assigned permission #%d to user #%d.", in.PermissionID, in.UserID), nil
}

// rbacRemovePermissionFromUser handles rbac_remove_permission_from_user.
func rbacRemovePermissionFromUser(ctx context.Context, b Backend, in rbacRemovePermissionFromUserInput) (rbacAssignmentOutput, string, error) {
	if err := b.RemovePermissionFromUser(ctx, in.UserID, in.PermissionID); err != nil {
		return rbacAssignmentOutput{}, "", err
	}
	return rbacAssignmentOutput{Success: true}, fmt.Sprintf("Removed permission #%d from user #%d.", in.PermissionID, in.UserID), nil
}

// rbacAssignPermissionToRole handles rbac_assign_permission_to_role.
func rbacAssignPermissionToRole(ctx context.Context, b Backend, in rbacAssignPermissionToRoleInput) (rbacAssignmentOutput, string, error) {
	if err := b.AssignPermissionToRole(ctx, in.RoleID, in.PermissionID); err != nil {
		return rbacAssignmentOutput{}, "", err
	}
	return rbacAssignmentOutput{Success: true}, fmt.Sprintf("Assigned permission #%d to role #%d.", in.PermissionID, in.RoleID), nil
}

// rbacRemovePermissionFromRole handles rbac_remove_permission_from_role.
func rbacRemovePermissionFromRole(ctx context.Context, b Backend, in rbacRemovePermissionFromRoleInput) (rbacAssignmentOutput, string, error) {
	if err := b.RemovePermissionFromRole(ctx, in.RoleID, in.PermissionID); err != nil {
		return rbacAssignmentOutput{}, "", err
	}
	return rbacAssignmentOutput{Success: true}, fmt.Sprintf("Removed permission #%d from role #%d.", in.PermissionID, in.RoleID), nil
}
