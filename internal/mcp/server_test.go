package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionCan(t *testing.T) {
	s := newTestSession("ticket:read", "ticket:create")
	assert.True(t, s.Can("ticket:read"))
	assert.True(t, s.Can("ticket:create"))
	assert.False(t, s.Can("ticket:delete"))

	var nilSession *Session
	assert.False(t, nilSession.Can("anything"))
}

func TestSessionContextRoundTrip(t *testing.T) {
	s := newTestSession("x")
	ctx := ctxWithSession(s)

	got, ok := SessionFromContext(ctx)
	assert.True(t, ok)
	assert.Same(t, s, got)

	_, ok = SessionFromContext(context.Background())
	assert.False(t, ok)
}

func TestRequirePermission(t *testing.T) {
	t.Run("no session", func(t *testing.T) {
		err := RequirePermission(context.Background(), "ticket:read")
		assert.ErrorIs(t, err, ErrUnauthenticated)
	})

	t.Run("missing permission", func(t *testing.T) {
		ctx := ctxWithSession(newTestSession("ticket:read"))
		err := RequirePermission(ctx, "ticket:delete")
		var permErr *PermissionError
		assert.True(t, errors.As(err, &permErr))
		assert.Equal(t, "ticket:delete", permErr.Code)
	})

	t.Run("has permission", func(t *testing.T) {
		ctx := ctxWithSession(newTestSession("ticket:read"))
		assert.NoError(t, RequirePermission(ctx, "ticket:read"))
	})
}

func TestNewMCPServer(t *testing.T) {
	mb := &MockBackend{}

	t.Run("all toolsets when empty", func(t *testing.T) {
		s := NewMCPServer(mb, nil)
		assert.NotNil(t, s)
	})

	t.Run("subset of toolsets", func(t *testing.T) {
		s := NewMCPServer(mb, []string{"ticket", "rbac"})
		assert.NotNil(t, s)
	})

	t.Run("unknown toolset ignored", func(t *testing.T) {
		s := NewMCPServer(mb, []string{"does-not-exist"})
		assert.NotNil(t, s)
	})
}
