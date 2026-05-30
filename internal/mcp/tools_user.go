package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/user"
)

// ----------------------------------------------------------------------------
// Local output view
// ----------------------------------------------------------------------------
//
// auth.UserInfo has no slice/map fields (only scalars and a nullable *time.Time),
// so it is safe over the wire on its own; userView mirrors it as an MCP-local
// struct so the whole package uniformly returns MCP-local Output types rather
// than service-layer DTOs. The LastLoginAt pointer carries omitempty.
type userView struct {
	ID          uint       `json:"id" jsonschema:"the user's numeric ID"`
	Email       string     `json:"email" jsonschema:"the user's email address"`
	Username    string     `json:"username" jsonschema:"the user's username"`
	FirstName   string     `json:"first_name" jsonschema:"the user's given name"`
	LastName    string     `json:"last_name" jsonschema:"the user's family name"`
	Role        string     `json:"role" jsonschema:"the user's role"`
	IsActive    bool       `json:"is_active" jsonschema:"whether the account is active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty" jsonschema:"when the user last logged in, if ever"`
}

// userViewFrom converts an auth.UserInfo into the MCP-local view. Sensitive
// fields are never present on auth.UserInfo, so none are carried here either.
func userViewFrom(u *auth.UserInfo) userView {
	if u == nil {
		return userView{}
	}
	return userView{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Role:        u.Role,
		IsActive:    u.IsActive,
		LastLoginAt: u.LastLoginAt,
	}
}

// userViewsFrom converts a slice of auth.UserInfo into views.
func userViewsFrom(us []auth.UserInfo) []userView {
	if len(us) == 0 {
		return nil
	}
	views := make([]userView, len(us))
	for i := range us {
		views[i] = userViewFrom(&us[i])
	}
	return views
}

// registerUserTools registers the user-domain MCP tools.
// See server.go for the tool implementation conventions and auth_whoami template.
//
// All identifiers in this file are prefixed with "user" to avoid collisions with
// sibling domain files in the same package. The structured outputs use the
// MCP-local userView (translated from auth.UserInfo, which already omits
// sensitive fields such as password and password hash); the inputs are likewise
// MCP-specific structs translated into the service-layer DTOs.
func registerUserTools(s *mcp.Server, b Backend) {
	registerTool(s,
		"user_create",
		"Create a new user account with the given profile, role, and initial password.",
		"user:write",
		func(ctx context.Context, in userCreateInput) (userView, string, error) {
			return userCreate(ctx, b, in)
		},
	)

	registerTool(s,
		"user_get",
		"Fetch a single user by their numeric ID. Sensitive fields are never returned.",
		"user:read",
		func(ctx context.Context, in userGetInput) (userView, string, error) {
			return userGet(ctx, b, in)
		},
	)

	registerTool(s,
		"user_update",
		"Update an existing user's profile fields by numeric ID. Only provided fields are changed.",
		"user:write",
		func(ctx context.Context, in userUpdateInput) (userView, string, error) {
			return userUpdate(ctx, b, in)
		},
	)

	registerTool(s,
		"user_delete",
		"Soft-delete a user account by its numeric ID.",
		"user:write",
		func(ctx context.Context, in userDeleteInput) (userDeleteOutput, string, error) {
			return userDelete(ctx, b, in)
		},
	)

	registerTool(s,
		"user_list",
		"List users with pagination and filtering by search term, role, or active status.",
		"user:read",
		func(ctx context.Context, in userListInput) (userListOutput, string, error) {
			return userList(ctx, b, in)
		},
	)

	registerTool(s,
		"user_activate",
		"Activate a user account by its numeric ID.",
		"user:write",
		func(ctx context.Context, in userActivateInput) (userStatusOutput, string, error) {
			return userActivate(ctx, b, in)
		},
	)

	registerTool(s,
		"user_deactivate",
		"Deactivate a user account by its numeric ID.",
		"user:write",
		func(ctx context.Context, in userDeactivateInput) (userStatusOutput, string, error) {
			return userDeactivate(ctx, b, in)
		},
	)

	registerTool(s,
		"user_change_password",
		"Set a new password for a user account by its numeric ID (administrative reset).",
		"user:write",
		func(ctx context.Context, in userChangePasswordInput) (userChangePasswordOutput, string, error) {
			return userChangePassword(ctx, b, in)
		},
	)

	registerTool(s,
		"user_stats",
		"Return aggregate user statistics keyed by metric name.",
		"user:read",
		func(ctx context.Context, in userStatsInput) (userStatsOutput, string, error) {
			return userStats(ctx, b, in)
		},
	)
}

// ----------------------------------------------------------------------------
// Input / Output schemas
// ----------------------------------------------------------------------------

// userCreateInput is the MCP input schema for user_create. It mirrors the fields
// of user.CreateUserRequest.
type userCreateInput struct {
	Email       string `json:"email" jsonschema:"user email address (required, unique)"`
	Username    string `json:"username" jsonschema:"unique username, 3-50 chars (required)"`
	FirstName   string `json:"first_name" jsonschema:"user given name (required)"`
	LastName    string `json:"last_name" jsonschema:"user family name (required)"`
	Password    string `json:"password" jsonschema:"initial password meeting complexity rules (required)"`
	Role        string `json:"role" jsonschema:"role to assign: admin, engineer, support, customer, or sales (required)"`
	IsActive    bool   `json:"is_active,omitempty" jsonschema:"whether the account is active on creation"`
	Preferences string `json:"preferences,omitempty" jsonschema:"user preferences as a JSON-encoded string"`
}

// userGetInput is the MCP input schema for user_get.
type userGetInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the user to fetch"`
}

// userUpdateInput is the MCP input schema for user_update. All fields except ID
// are optional; only non-empty values are applied. IsActive is a pointer so an
// explicit false can be distinguished from "unset".
type userUpdateInput struct {
	ID          uint   `json:"id" jsonschema:"numeric ID of the user to update"`
	Email       string `json:"email,omitempty" jsonschema:"new email address"`
	Username    string `json:"username,omitempty" jsonschema:"new username, 3-50 chars"`
	FirstName   string `json:"first_name,omitempty" jsonschema:"new given name"`
	LastName    string `json:"last_name,omitempty" jsonschema:"new family name"`
	Role        string `json:"role,omitempty" jsonschema:"new role: admin, engineer, support, customer, or sales"`
	IsActive    *bool  `json:"is_active,omitempty" jsonschema:"new active status"`
	Preferences string `json:"preferences,omitempty" jsonschema:"new preferences as a JSON-encoded string"`
}

// userDeleteInput is the MCP input schema for user_delete.
type userDeleteInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the user to delete"`
}

// userDeleteOutput reports the outcome of a user_delete call.
type userDeleteOutput struct {
	ID      uint `json:"id" jsonschema:"the ID of the deleted user"`
	Deleted bool `json:"deleted" jsonschema:"true when the user was deleted"`
}

// userListInput is the MCP input schema for user_list. It exposes pagination and
// the supported filters.
type userListInput struct {
	Page     int    `json:"page,omitempty" jsonschema:"page number, 1-based (default 1)"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"number of items per page, 1-100 (default 20)"`
	Search   string `json:"search,omitempty" jsonschema:"search term matched against user fields"`
	Role     string `json:"role,omitempty" jsonschema:"filter by exact role"`
	IsActive *bool  `json:"is_active,omitempty" jsonschema:"filter by active status; omit to include both"`
}

// userListOutput is the structured output of user_list. Users are returned as
// auth.UserInfo values, which omit sensitive fields.
type userListOutput struct {
	Users    []userView `json:"users,omitempty" jsonschema:"the page of users"`
	Total    int64      `json:"total" jsonschema:"total number of users matching the filters"`
	Page     int        `json:"page" jsonschema:"the page number returned"`
	PageSize int        `json:"page_size" jsonschema:"the page size used"`
}

// userActivateInput is the MCP input schema for user_activate.
type userActivateInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the user to activate"`
}

// userDeactivateInput is the MCP input schema for user_deactivate.
type userDeactivateInput struct {
	ID uint `json:"id" jsonschema:"numeric ID of the user to deactivate"`
}

// userStatusOutput reports the resulting active status of an activate/deactivate
// call.
type userStatusOutput struct {
	ID       uint `json:"id" jsonschema:"the user ID"`
	IsActive bool `json:"is_active" jsonschema:"the user's active status after the operation"`
}

// userChangePasswordInput is the MCP input schema for user_change_password.
type userChangePasswordInput struct {
	ID          uint   `json:"id" jsonschema:"numeric ID of the user whose password is being changed"`
	NewPassword string `json:"new_password" jsonschema:"the new password meeting complexity rules (required)"`
}

// userChangePasswordOutput reports the outcome of a user_change_password call. It
// deliberately carries no password material.
type userChangePasswordOutput struct {
	ID      uint `json:"id" jsonschema:"the user ID"`
	Changed bool `json:"changed" jsonschema:"true when the password was changed"`
}

// userStatsInput is the MCP input schema for user_stats. It takes no arguments.
type userStatsInput struct{}

// userStatsOutput is the structured output of user_stats. Statistics are returned
// as a free-form map keyed by metric name.
type userStatsOutput struct {
	Stats map[string]interface{} `json:"stats,omitempty" jsonschema:"aggregate user statistics keyed by metric name"`
}

// ----------------------------------------------------------------------------
// Business closures (named functions for direct unit testing)
// ----------------------------------------------------------------------------

// userCreate translates the MCP input into a service request and creates the user
// via the Backend. The returned userView omits sensitive fields.
func userCreate(_ context.Context, b Backend, in userCreateInput) (userView, string, error) {
	req := &user.CreateUserRequest{
		Email:       in.Email,
		Username:    in.Username,
		FirstName:   in.FirstName,
		LastName:    in.LastName,
		Password:    in.Password,
		Role:        in.Role,
		IsActive:    in.IsActive,
		Preferences: in.Preferences,
	}

	resp, err := b.CreateUser(req)
	if err != nil {
		return userView{}, "", err
	}
	summary := fmt.Sprintf("Created user %q (#%d, role %s).", resp.Username, resp.ID, resp.Role)
	return userViewFrom(resp), summary, nil
}

// userGet fetches a single user by ID. The returned userView omits sensitive
// fields.
func userGet(_ context.Context, b Backend, in userGetInput) (userView, string, error) {
	resp, err := b.GetUser(in.ID)
	if err != nil {
		return userView{}, "", err
	}
	summary := fmt.Sprintf("User %q (#%d, role %s).", resp.Username, resp.ID, resp.Role)
	return userViewFrom(resp), summary, nil
}

// userUpdate applies the provided fields to an existing user.
func userUpdate(_ context.Context, b Backend, in userUpdateInput) (userView, string, error) {
	req := &user.UpdateUserRequest{
		Email:       in.Email,
		Username:    in.Username,
		FirstName:   in.FirstName,
		LastName:    in.LastName,
		Role:        in.Role,
		IsActive:    in.IsActive,
		Preferences: in.Preferences,
	}

	resp, err := b.UpdateUser(in.ID, req)
	if err != nil {
		return userView{}, "", err
	}
	summary := fmt.Sprintf("Updated user %q (#%d).", resp.Username, resp.ID)
	return userViewFrom(resp), summary, nil
}

// userDelete soft-deletes a user.
func userDelete(_ context.Context, b Backend, in userDeleteInput) (userDeleteOutput, string, error) {
	if err := b.DeleteUser(in.ID); err != nil {
		return userDeleteOutput{}, "", err
	}
	out := userDeleteOutput{ID: in.ID, Deleted: true}
	summary := fmt.Sprintf("Deleted user #%d.", in.ID)
	return out, summary, nil
}

// userList lists users with pagination and filtering.
func userList(_ context.Context, b Backend, in userListInput) (userListOutput, string, error) {
	req := &user.UserListRequest{
		Page:     in.Page,
		PageSize: in.PageSize,
		Search:   in.Search,
		Role:     in.Role,
		IsActive: in.IsActive,
	}

	resp, err := b.ListUsers(req)
	if err != nil {
		return userListOutput{}, "", err
	}

	out := userListOutput{
		Users:    userViewsFrom(resp.Data),
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if resp.Meta != nil {
		out.Total = resp.Meta.Total
		out.Page = resp.Meta.Page
		out.PageSize = resp.Meta.PageSize
	}
	summary := fmt.Sprintf("Returned %d of %d user(s).", len(out.Users), out.Total)
	return out, summary, nil
}

// userActivate activates a user account.
func userActivate(_ context.Context, b Backend, in userActivateInput) (userStatusOutput, string, error) {
	if err := b.ActivateUser(in.ID); err != nil {
		return userStatusOutput{}, "", err
	}
	out := userStatusOutput{ID: in.ID, IsActive: true}
	summary := fmt.Sprintf("Activated user #%d.", in.ID)
	return out, summary, nil
}

// userDeactivate deactivates a user account.
func userDeactivate(_ context.Context, b Backend, in userDeactivateInput) (userStatusOutput, string, error) {
	if err := b.DeactivateUser(in.ID); err != nil {
		return userStatusOutput{}, "", err
	}
	out := userStatusOutput{ID: in.ID, IsActive: false}
	summary := fmt.Sprintf("Deactivated user #%d.", in.ID)
	return out, summary, nil
}

// userChangePassword sets a new password for a user. The password material is
// never echoed back in the output.
func userChangePassword(_ context.Context, b Backend, in userChangePasswordInput) (userChangePasswordOutput, string, error) {
	if err := b.ChangeUserPassword(in.ID, in.NewPassword); err != nil {
		return userChangePasswordOutput{}, "", err
	}
	out := userChangePasswordOutput{ID: in.ID, Changed: true}
	summary := fmt.Sprintf("Changed password for user #%d.", in.ID)
	return out, summary, nil
}

// userStats returns aggregate user statistics.
func userStats(_ context.Context, b Backend, _ userStatsInput) (userStatsOutput, string, error) {
	stats, err := b.GetUserStats()
	if err != nil {
		return userStatsOutput{}, "", err
	}
	return userStatsOutput{Stats: stats}, "Fetched user statistics.", nil
}
