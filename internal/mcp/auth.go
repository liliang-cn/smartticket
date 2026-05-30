package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/company/smartticket/internal/auth"
	"github.com/company/smartticket/internal/services"
)

// ErrUnauthenticated is returned when no authenticated Session is present in the
// request context. Tool handlers map this to a structured MCP error.
var ErrUnauthenticated = errors.New("unauthenticated: no valid session on this connection")

// PermissionError indicates the current Session lacks a required permission.
// It is returned by RequirePermission and is intended to be surfaced to the
// client as a tool error (IsError) rather than a protocol error.
type PermissionError struct {
	Code string
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission denied: missing %q", e.Code)
}

// Authenticator validates connection credentials (JWT) and builds the Session
// (identity + effective permission set) used for per-tool RBAC.
type Authenticator struct {
	authService *auth.Service
	perms       *services.PermissionService
}

// NewAuthenticator creates an Authenticator backed by the auth and permission
// services.
func NewAuthenticator(authService *auth.Service, perms *services.PermissionService) *Authenticator {
	return &Authenticator{authService: authService, perms: perms}
}

// Authenticate validates the bearer token, resolves the user, loads their
// effective permissions, and returns a populated Session.
func (a *Authenticator) Authenticate(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, errors.New("missing credential token")
	}

	claims, err := a.authService.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid credential token: %w", err)
	}

	permList, err := a.perms.GetUserPermissions(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to load permissions: %w", err)
	}

	permMap := make(map[string]bool, len(permList))
	for _, p := range permList {
		permMap[p.Code] = true
	}

	return &Session{
		UserID:      claims.UserID,
		Permissions: permMap,
	}, nil
}

// RequirePermission enforces that the context carries an authenticated Session
// holding the named permission code. It returns ErrUnauthenticated when no
// session is present, or a *PermissionError when the session lacks the code.
// Tool handlers should call this before invoking the Backend.
func RequirePermission(ctx context.Context, code string) error {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return ErrUnauthenticated
	}
	if !session.Can(code) {
		return &PermissionError{Code: code}
	}
	return nil
}
