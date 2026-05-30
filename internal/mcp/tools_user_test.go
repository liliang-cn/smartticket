package mcp

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/user"
)

func TestUserCreate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write"))

	in := userCreateInput{
		Email:     "jane@example.com",
		Username:  "janedoe",
		FirstName: "Jane",
		LastName:  "Doe",
		Password:  "SecurePass123!",
		Role:      "support",
		IsActive:  true,
	}

	mb.On("CreateUser", mock.MatchedBy(func(req *user.CreateUserRequest) bool {
		return req.Email == "jane@example.com" &&
			req.Username == "janedoe" &&
			req.FirstName == "Jane" &&
			req.LastName == "Doe" &&
			req.Password == "SecurePass123!" &&
			req.Role == "support" &&
			req.IsActive
	})).Return(&auth.UserInfo{ID: 7, Username: "janedoe", Role: "support"}, nil)

	out, summary, err := userCreate(ctx, mb, in)
	assert.NoError(t, err)
	assert.Equal(t, uint(7), out.ID)
	assert.Equal(t, `Created user "janedoe" (#7, role support).`, summary)
	mb.AssertExpectations(t)
}

func TestUserCreateError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write"))

	wantErr := errors.New("email already exists")
	mb.On("CreateUser", mock.Anything).Return(nil, wantErr)

	_, _, err := userCreate(ctx, mb, userCreateInput{Email: "x@example.com"})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestUserGet(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:read"))

	mb.On("GetUser", uint(3)).Return(&auth.UserInfo{ID: 3, Username: "bob", Role: "admin"}, nil)

	out, summary, err := userGet(ctx, mb, userGetInput{ID: 3})
	assert.NoError(t, err)
	assert.Equal(t, uint(3), out.ID)
	assert.Equal(t, `User "bob" (#3, role admin).`, summary)
	mb.AssertExpectations(t)
}

func TestUserUpdate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write"))

	active := false
	in := userUpdateInput{
		ID:        9,
		FirstName: "Updated",
		Role:      "engineer",
		IsActive:  &active,
	}

	mb.On("UpdateUser", uint(9), mock.MatchedBy(func(req *user.UpdateUserRequest) bool {
		return req.FirstName == "Updated" &&
			req.Role == "engineer" &&
			req.IsActive != nil && !*req.IsActive
	})).Return(&auth.UserInfo{ID: 9, Username: "carol"}, nil)

	out, summary, err := userUpdate(ctx, mb, in)
	assert.NoError(t, err)
	assert.Equal(t, uint(9), out.ID)
	assert.Equal(t, `Updated user "carol" (#9).`, summary)
	mb.AssertExpectations(t)
}

func TestUserDelete(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write"))

	mb.On("DeleteUser", uint(5)).Return(nil)

	out, summary, err := userDelete(ctx, mb, userDeleteInput{ID: 5})
	assert.NoError(t, err)
	assert.Equal(t, uint(5), out.ID)
	assert.True(t, out.Deleted)
	assert.Equal(t, "Deleted user #5.", summary)
	mb.AssertExpectations(t)
}

func TestUserDeleteError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write"))

	wantErr := errors.New("user not found")
	mb.On("DeleteUser", uint(5)).Return(wantErr)

	_, _, err := userDelete(ctx, mb, userDeleteInput{ID: 5})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestUserList(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:read"))

	active := true
	in := userListInput{
		Page:     2,
		PageSize: 10,
		Search:   "jane",
		Role:     "support",
		IsActive: &active,
	}

	mb.On("ListUsers", mock.MatchedBy(func(req *user.UserListRequest) bool {
		return req.Page == 2 &&
			req.PageSize == 10 &&
			req.Search == "jane" &&
			req.Role == "support" &&
			req.IsActive != nil && *req.IsActive
	})).Return(&user.UserListResponse{
		Success: true,
		Data:    []auth.UserInfo{{ID: 1}, {ID: 2}},
		Meta:    &user.PaginationMeta{Page: 2, PageSize: 10, Total: 12},
	}, nil)

	out, summary, err := userList(ctx, mb, in)
	assert.NoError(t, err)
	assert.Len(t, out.Users, 2)
	assert.Equal(t, int64(12), out.Total)
	assert.Equal(t, 2, out.Page)
	assert.Equal(t, 10, out.PageSize)
	assert.Equal(t, "Returned 2 of 12 user(s).", summary)
	mb.AssertExpectations(t)
}

func TestUserListNilMeta(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:read"))

	mb.On("ListUsers", mock.Anything).Return(&user.UserListResponse{
		Data: []auth.UserInfo{},
	}, nil)

	out, summary, err := userList(ctx, mb, userListInput{})
	assert.NoError(t, err)
	assert.Empty(t, out.Users)
	assert.Equal(t, int64(0), out.Total)
	assert.Equal(t, "Returned 0 of 0 user(s).", summary)
	mb.AssertExpectations(t)
}

func TestUserActivate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write"))

	mb.On("ActivateUser", uint(4)).Return(nil)

	out, summary, err := userActivate(ctx, mb, userActivateInput{ID: 4})
	assert.NoError(t, err)
	assert.Equal(t, uint(4), out.ID)
	assert.True(t, out.IsActive)
	assert.Equal(t, "Activated user #4.", summary)
	mb.AssertExpectations(t)
}

func TestUserDeactivate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write"))

	mb.On("DeactivateUser", uint(4)).Return(nil)

	out, summary, err := userDeactivate(ctx, mb, userDeactivateInput{ID: 4})
	assert.NoError(t, err)
	assert.Equal(t, uint(4), out.ID)
	assert.False(t, out.IsActive)
	assert.Equal(t, "Deactivated user #4.", summary)
	mb.AssertExpectations(t)
}

func TestUserChangePassword(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write"))

	mb.On("ChangeUserPassword", uint(6), "NewSecret123!").Return(nil)

	out, summary, err := userChangePassword(ctx, mb, userChangePasswordInput{ID: 6, NewPassword: "NewSecret123!"})
	assert.NoError(t, err)
	assert.Equal(t, uint(6), out.ID)
	assert.True(t, out.Changed)
	assert.Equal(t, "Changed password for user #6.", summary)
	mb.AssertExpectations(t)
}

func TestUserChangePasswordError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write"))

	wantErr := errors.New("password validation failed")
	mb.On("ChangeUserPassword", uint(6), "weak").Return(wantErr)

	_, _, err := userChangePassword(ctx, mb, userChangePasswordInput{ID: 6, NewPassword: "weak"})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestUserStats(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:read"))

	stats := map[string]interface{}{"total_users": int64(10), "active_users": int64(8)}
	mb.On("GetUserStats").Return(stats, nil)

	out, summary, err := userStats(ctx, mb, userStatsInput{})
	assert.NoError(t, err)
	assert.Equal(t, stats, out.Stats)
	assert.Equal(t, "Fetched user statistics.", summary)
	mb.AssertExpectations(t)
}

// TestUserOutputsOmitSensitiveFields verifies that the JSON-serialized outputs of
// user_create and user_get never expose password or hash material.
func TestUserOutputsOmitSensitiveFields(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("user:write", "user:read"))

	info := &auth.UserInfo{
		ID:        11,
		Email:     "secure@example.com",
		Username:  "secure",
		FirstName: "Se",
		LastName:  "Cure",
		Role:      "admin",
		IsActive:  true,
	}
	mb.On("CreateUser", mock.Anything).Return(info, nil)
	mb.On("GetUser", uint(11)).Return(info, nil)

	createOut, _, err := userCreate(ctx, mb, userCreateInput{
		Email: "secure@example.com", Username: "secure", Password: "SecurePass123!", Role: "admin",
	})
	assert.NoError(t, err)

	getOut, _, err := userGet(ctx, mb, userGetInput{ID: 11})
	assert.NoError(t, err)

	for name, out := range map[string]interface{}{"create": createOut, "get": getOut} {
		raw, err := json.Marshal(out)
		assert.NoError(t, err)
		lower := strings.ToLower(string(raw))
		assert.NotContains(t, lower, "password", "%s output must not expose password", name)
		assert.NotContains(t, lower, "hash", "%s output must not expose hash", name)
	}

	mb.AssertExpectations(t)
}
