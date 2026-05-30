package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/auth"
	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/services"
	"github.com/company/smartticket/internal/ticket"
)

// ----------------------------------------------------------------------------
// End-to-end transport integration: a real go-sdk client connected to
// NewMCPServer over an in-memory transport. These tests exercise the full
// chain — tool registration, JSON-Schema-driven argument decoding, session
// propagation, RBAC enforcement, and result marshaling — that the per-tool unit
// tests deliberately skip by calling the closures directly.
// ----------------------------------------------------------------------------

// connectClient starts NewMCPServer(backend, nil) over an in-memory transport,
// driving the server with baseCtx (which carries the session, if any), and
// returns a connected client session. The server goroutine and the session are
// torn down via t.Cleanup.
func connectClient(t *testing.T, baseCtx context.Context, backend Backend) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	srv := NewMCPServer(backend, nil)

	// Run the server with the session-bearing context. Server.Run blocks until
	// the connection closes or the context is cancelled.
	runCtx, cancelRun := context.WithCancel(baseCtx)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Run(runCtx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	cs, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err, "client should connect to the in-memory MCP server")

	t.Cleanup(func() {
		_ = cs.Close()
		cancelRun()
		select {
		case <-serverErr:
		case <-time.After(2 * time.Second):
		}
	})

	return cs
}

// TestIntegrationToolCallSuccess verifies the happy path end to end: a session
// holding ticket:read calls ticket_get, the backend returns data, and the
// client observes a non-error result with the expected text summary and
// structured content.
//
// Crucially, the backend returns a TicketResponse whose Tags ([]string) and
// CustomFields (map) are nil. The MCP tool's local ticketResponse view marks
// those fields omitempty, so they are omitted rather than emitted as JSON null;
// this exercises the SDK's output-schema validation over a real transport and
// locks in the fix for the array/object-vs-null protocol error.
func TestIntegrationToolCallSuccess(t *testing.T) {
	mb := &MockBackend{}
	mb.On("GetTicket", uint(42)).
		Return(&ticket.TicketResponse{
			ID:           42,
			TicketNumber: "TK-42",
			Tags:         nil, // nil slice — must not marshal to JSON null over the wire.
			CustomFields: nil, // nil map — must not marshal to JSON null over the wire.
		}, nil)

	baseCtx := WithSession(context.Background(), newTestSession("ticket:read"))
	cs := connectClient(t, baseCtx, mb)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "ticket_get",
		Arguments: map[string]any{"id": 42},
	})
	require.NoError(t, err, "CallTool must not return a protocol-level error (nil slice/map output must not trip output-schema validation)")
	assert.False(t, res.IsError, "successful tool call should not be flagged as error: %+v", res.Content)

	// Text summary content.
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "first content element should be text")
	assert.Equal(t, "fetched ticket #42 (TK-42)", text.Text)

	// Structured content is populated by the SDK from the typed Out value.
	require.NotNil(t, res.StructuredContent)
	structured, ok := res.StructuredContent.(map[string]any)
	require.True(t, ok, "structured content should marshal to a JSON object")
	assert.EqualValues(t, 42, structured["id"])
	assert.Equal(t, "TK-42", structured["ticket_number"])
	// The omitempty slice/map fields are absent rather than null.
	_, hasTags := structured["tags"]
	assert.False(t, hasTags, "nil Tags should be omitted from structured output, not present as null")
	_, hasCustomFields := structured["custom_fields"]
	assert.False(t, hasCustomFields, "nil CustomFields should be omitted from structured output, not present as null")

	mb.AssertExpectations(t)
}

// TestIntegrationRBACDenied verifies that a session lacking the required
// permission is rejected by the registerTool RBAC gate before the backend is
// touched: the client sees IsError with a "permission denied" message, and the
// backend method is never invoked.
//
// This case deliberately targets ticket_create, whose output is a struct with
// slice/map fields (Tags/CustomFields). On the RBAC-denied path the handler
// returns a zero Out, whose nil slice/map would marshal to JSON null and trip
// the SDK's output-schema validation — unless those fields carry omitempty. So
// this test also regression-locks the error-path half of the null-field fix.
func TestIntegrationRBACDenied(t *testing.T) {
	mb := &MockBackend{}
	// No expectations registered: the backend must not be called.

	// Session has ticket:read but ticket_create requires ticket:write.
	baseCtx := WithSession(context.Background(), newTestSession("ticket:read"))
	cs := connectClient(t, baseCtx, mb)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_create",
		Arguments: map[string]any{
			"title":           "denied",
			"description":     "should never reach the backend",
			"requester_name":  "Nobody",
			"requester_email": "nobody@example.com",
		},
	})
	require.NoError(t, err, "RBAC denial is a tool error, not a protocol error (zero Out with nil slice/map must not trip output-schema validation)")
	assert.True(t, res.IsError, "RBAC-denied call should be flagged as a tool error")

	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "permission denied")
	assert.Contains(t, text.Text, "ticket:write")

	mb.AssertNotCalled(t, "CreateTicket", mock.Anything, mock.Anything)
}

// TestIntegrationAuthWhoami verifies the identity tool returns the acting
// user's ID and permission set without requiring any specific permission.
func TestIntegrationAuthWhoami(t *testing.T) {
	mb := &MockBackend{}

	baseCtx := WithSession(context.Background(), newTestSession("ticket:read", "knowledge:read"))
	cs := connectClient(t, baseCtx, mb)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "auth_whoami",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	assert.False(t, res.IsError, "whoami should succeed for any authenticated session: %+v", res.Content)

	require.NotNil(t, res.StructuredContent)
	structured, ok := res.StructuredContent.(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, 1, structured["user_id"])

	perms, ok := structured["permissions"].([]any)
	require.True(t, ok, "permissions should be a JSON array")
	got := make([]string, 0, len(perms))
	for _, p := range perms {
		got = append(got, p.(string))
	}
	assert.ElementsMatch(t, []string{"ticket:read", "knowledge:read"}, got)
}

// TestIntegrationUnauthenticated verifies that calling a tool over a connection
// whose base context carries no session yields a tool error (the RBAC gate
// detects the missing session) and the backend is not touched.
//
// As with the RBAC-denied case, this targets the struct-returning ticket_create
// tool so the zero Out (nil slice/map) on the error path is exercised against
// the SDK's output-schema validation, regression-locking the omitempty fix.
func TestIntegrationUnauthenticated(t *testing.T) {
	mb := &MockBackend{}

	// No WithSession on the base context.
	cs := connectClient(t, context.Background(), mb)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_create",
		Arguments: map[string]any{
			"title":           "no session",
			"description":     "should never reach the backend",
			"requester_name":  "Nobody",
			"requester_email": "nobody@example.com",
		},
	})
	require.NoError(t, err, "unauthenticated zero Out with nil slice/map must not trip output-schema validation")
	assert.True(t, res.IsError, "unauthenticated call should be a tool error")

	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	// With no session, the RBAC gate returns ErrUnauthenticated, which
	// mapServiceError surfaces as "not authenticated".
	assert.Equal(t, "not authenticated", text.Text)

	mb.AssertNotCalled(t, "CreateTicket", mock.Anything, mock.Anything)
}

// TestIntegrationBackendErrorPath verifies the business-error path of a
// struct-returning tool over a real transport: the session is authorized and the
// backend is reached, but the backend returns an error. registerTool returns a
// zero ticketResponse Out (nil Tags/CustomFields) alongside an IsError result.
// The omitempty fields keep the zero Out from tripping the SDK's output-schema
// validation, so the client observes a clean tool error rather than a
// protocol-level "type null, want array/object" failure.
func TestIntegrationBackendErrorPath(t *testing.T) {
	mb := &MockBackend{}
	mb.On("GetTicket", uint(404)).
		Return((*ticket.TicketResponse)(nil), apperrors.NewNotFoundError("Ticket"))

	baseCtx := WithSession(context.Background(), newTestSession("ticket:read"))
	cs := connectClient(t, baseCtx, mb)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "ticket_get",
		Arguments: map[string]any{"id": 404},
	})
	require.NoError(t, err, "a backend error is a tool error; the zero Out with nil slice/map must not trip output-schema validation")
	assert.True(t, res.IsError, "backend error should be flagged as a tool error")

	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "Ticket")

	mb.AssertExpectations(t)
}

// ----------------------------------------------------------------------------
// HTTP auth middleware: reject requests without a valid bearer credential
// (401) and never let them reach the wrapped MCP handler; pass valid ones
// through with the Session injected into the request context.
// ----------------------------------------------------------------------------

// newTestDB builds an in-memory SQLite DB migrated with the auth/RBAC models
// needed to mint and validate a real JWT.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Permission{},
		&models.Role{},
		&models.RolePermission{},
		&models.UserPermission{},
		&models.UserRole{},
	))
	return db
}

// rejectingAuthenticator returns an Authenticator backed by a real auth.Service
// with a JWT secret but no usable data. It rejects any malformed/forged token at
// the JWT-parse stage (before touching the DB), which is all the 401 reject
// cases need.
func rejectingAuthenticator(t *testing.T) *Authenticator {
	t.Helper()
	db := newTestDB(t)
	authSvc := auth.NewService(db, "integration-test-secret", time.Hour, 24*time.Hour, "smartticket-test")
	perms := services.NewPermissionService(db)
	return NewAuthenticator(authSvc, perms)
}

func TestAuthHTTPMiddleware_MissingHeader(t *testing.T) {
	authn := rejectingAuthenticator(t)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	authHTTPMiddleware(authn, next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called, "missing-credential request must not reach the MCP handler")
}

func TestAuthHTTPMiddleware_BadToken(t *testing.T) {
	authn := rejectingAuthenticator(t)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	authHTTPMiddleware(authn, next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called, "forged-token request must not reach the MCP handler")
}

// TestAuthHTTPMiddleware_ValidToken mints a real JWT for a live user and asserts
// the middleware passes the request through to next with the Session injected.
func TestAuthHTTPMiddleware_ValidToken(t *testing.T) {
	db := newTestDB(t)
	authSvc := auth.NewService(db, "integration-test-secret", time.Hour, 24*time.Hour, "smartticket-test")
	perms := services.NewPermissionService(db)
	authn := NewAuthenticator(authSvc, perms)

	// Seed a user and obtain a genuine access token via the login flow.
	const email, password = "agent@example.com", "password123"
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	u := &models.User{
		Email:        email,
		Username:     "agent",
		FirstName:    "Test",
		LastName:     "Agent",
		PasswordHash: string(hashed),
		IsActive:     true,
	}
	require.NoError(t, db.Create(u).Error)

	login, err := authSvc.Login(&auth.LoginRequest{Email: email, Password: password}, "127.0.0.1", "test")
	require.NoError(t, err)
	require.NotEmpty(t, login.Tokens.AccessToken)

	var gotSession *Session
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, ok := SessionFromContext(r.Context())
		require.True(t, ok, "session must be injected into the request context")
		gotSession = s
		w.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+login.Tokens.AccessToken)
	authHTTPMiddleware(authn, next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code, "valid credential should pass through to next")
	require.NotNil(t, gotSession, "next handler should have observed the injected session")
	assert.Equal(t, u.ID, gotSession.UserID)
}
