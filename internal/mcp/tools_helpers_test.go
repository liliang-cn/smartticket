package mcp

import (
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/company/smartticket/internal/errors"
)

// textContent extracts the text of the first TextContent in a tool result.
func textContent(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, r.Content)
	tc, ok := r.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected first content to be *mcp.TextContent")
	return tc.Text
}

// assertPlainError returns a bare errors.New value (no AppError wrapping).
func assertPlainError(msg string) error { return errors.New(msg) }

func TestMapServiceError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.Nil(t, mapServiceError(nil))
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := mapServiceError(ErrUnauthenticated)
		require.NotNil(t, r)
		assert.True(t, r.IsError)
		assert.Equal(t, "not authenticated", textContent(t, r))
	})

	t.Run("permission error", func(t *testing.T) {
		r := mapServiceError(&PermissionError{Code: "ticket:delete"})
		require.NotNil(t, r)
		assert.True(t, r.IsError)
		assert.Contains(t, textContent(t, r), "ticket:delete")
	})

	t.Run("not found surfaces message", func(t *testing.T) {
		r := mapServiceError(apperrors.NewNotFoundError("ticket"))
		require.NotNil(t, r)
		assert.True(t, r.IsError)
		assert.Equal(t, "Ticket not found", textContent(t, r))
	})

	t.Run("validation surfaces message", func(t *testing.T) {
		r := mapServiceError(apperrors.NewValidationError("title is required"))
		assert.Equal(t, "title is required", textContent(t, r))
	})

	t.Run("conflict surfaces message", func(t *testing.T) {
		r := mapServiceError(apperrors.NewConflictError("already assigned"))
		assert.Equal(t, "already assigned", textContent(t, r))
	})

	t.Run("forbidden surfaces message", func(t *testing.T) {
		r := mapServiceError(apperrors.NewForbiddenError("nope"))
		assert.Equal(t, "nope", textContent(t, r))
	})

	t.Run("internal error does not leak", func(t *testing.T) {
		// Internal AppError: its Message must not be surfaced verbatim.
		r := mapServiceError(apperrors.NewInternalError("db exploded with secret connstring", nil))
		assert.Equal(t, "the operation could not be completed", textContent(t, r))
	})

	t.Run("unknown error does not leak", func(t *testing.T) {
		r := mapServiceError(assertPlainError("raw go error with /etc/secret path"))
		assert.Equal(t, "the operation could not be completed", textContent(t, r))
	})
}
