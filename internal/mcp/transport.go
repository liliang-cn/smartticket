package mcp

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RunStdio serves the MCP server over stdin/stdout. The connection is
// authenticated once, up front, using the supplied token; the resulting Session
// is injected into the context that drives the whole stdio session.
func RunStdio(ctx context.Context, s *mcp.Server, authn *Authenticator, token string) error {
	if token == "" {
		return errors.New("a credential token is required for stdio transport (use --token or SMARTTICKET_MCP_TOKEN)")
	}

	session, err := authn.Authenticate(ctx, token)
	if err != nil {
		return err
	}

	ctx = WithSession(ctx, session)
	return s.Run(ctx, &mcp.StdioTransport{})
}

// RunHTTP serves the MCP server over Streamable HTTP at addr. Each request is
// authenticated from its Authorization: Bearer header; the resulting Session is
// stored in the request context so it flows to tool handlers. Requests without a
// valid bearer credential receive 401. The server shuts down gracefully on
// SIGINT/SIGTERM or when ctx is cancelled.
func RunHTTP(ctx context.Context, s *mcp.Server, authn *Authenticator, addr string) error {
	// The same server instance handles every session.
	mcpHandler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return s
	}, nil)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           authHTTPMiddleware(authn, mcpHandler),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown on signal or context cancellation.
	shutdownCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-shutdownCtx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutCtx)
	}
}

// authHTTPMiddleware extracts and validates the bearer token, injecting the
// resulting Session into the request context before delegating to next. The MCP
// handler derives its handler context from r.Context(), so the Session reaches
// tool handlers' ctx. Requests without a valid bearer credential receive 401 and
// never reach next.
func authHTTPMiddleware(authn *Authenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			http.Error(w, "missing or malformed Authorization header", http.StatusUnauthorized)
			return
		}
		session, err := authn.Authenticate(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithSession(r.Context(), session)))
	})
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header,
// returning "" if absent or malformed.
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
