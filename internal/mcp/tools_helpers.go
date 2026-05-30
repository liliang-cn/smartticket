package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/logger"
)

// mcpLogger returns the package-scoped structured logger used for per-call tool
// logging. It reuses the repository's global zap logger so MCP logs land in the
// same sink as the rest of the application.
func mcpLogger() *zap.Logger {
	return logger.GetGlobalLogger().Named("mcp")
}

// registerTool registers an MCP tool with the cross-cutting concerns the design
// (§6) requires applied uniformly: RBAC enforcement, panic recovery, latency
// timing, structured per-call logging, and clean service-error mapping.
//
// Domain tasks should prefer this helper over calling mcp.AddTool directly. A
// task supplies:
//   - name/description: the tool identity and human-readable summary;
//   - permission: the RBAC permission code required to invoke the tool. Pass ""
//     to skip the permission check (still requires an authenticated session is
//     unnecessary — use "" only for identity tools like auth_whoami that handle
//     their own session checks);
//   - fn: the business closure. It receives the typed input and returns the
//     typed output, a friendly text summary, and an error. The closure must not
//     concern itself with RBAC, recover, logging, or error mapping — the helper
//     handles all of those.
//
// On success the helper returns textResult(summary) plus the typed Out so the
// SDK populates StructuredContent automatically. On error (including RBAC denial
// and recovered panics) it returns a *mcp.CallToolResult with IsError set and a
// clean, non-leaking message; the handler never returns a protocol-level error.
func registerTool[In, Out any](
	s *mcp.Server,
	name, description, permission string,
	fn func(ctx context.Context, in In) (Out, string, error),
) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        name,
		Description: description,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (result *mcp.CallToolResult, out Out, err error) {
		start := time.Now()

		// outcome is captured by the deferred logger; default to "error" so a
		// recovered panic (which jumps over the explicit assignment) is logged
		// as an error.
		outcome := "error"
		var userID uint
		if session, ok := SessionFromContext(ctx); ok && session != nil {
			userID = session.UserID
		}

		// ① / ② Panic recovery: convert any panic in the business closure (or
		// permission check) into a tool error result instead of crashing the
		// process. The named returns let us overwrite the result safely.
		defer func() {
			if r := recover(); r != nil {
				var zero Out
				result = toolError("internal error while executing tool")
				out = zero
				err = nil
				mcpLogger().Error("tool handler panic",
					zap.String("tool", name),
					zap.Uint("user", userID),
					zap.Any("panic", r),
					zap.Stack("stack"),
				)
			}
		}()

		// ① RBAC: enforce the required permission before doing any work. An
		// empty permission code means the tool opts out of the check (e.g.
		// auth_whoami).
		if permission != "" {
			if perr := RequirePermission(ctx, permission); perr != nil {
				var zero Out
				logToolCall(name, userID, time.Since(start), "denied")
				return mapServiceError(perr), zero, nil
			}
		}

		// ④ Invoke the business closure.
		o, summary, ferr := fn(ctx, in)
		latency := time.Since(start)

		// ⑤ Map any business/service error to a clean tool error result.
		if ferr != nil {
			var zero Out
			logToolCall(name, userID, latency, "error")
			return mapServiceError(ferr), zero, nil
		}

		// ⑥ Success: log and return the summary + structured output.
		outcome = "ok"
		logToolCall(name, userID, latency, outcome)
		return textResult(summary), o, nil
	})
}

// logToolCall emits one structured log line per tool invocation, recording the
// tool name, acting user, latency, and outcome (ok/error/denied).
func logToolCall(tool string, userID uint, latency time.Duration, outcome string) {
	mcpLogger().Info("tool call",
		zap.String("tool", tool),
		zap.Uint("user", userID),
		zap.Duration("latency", latency),
		zap.String("outcome", outcome),
	)
}

// mapServiceError converts an error returned by the Backend/service layer (or by
// RBAC enforcement) into a *mcp.CallToolResult carrying a clean, user-facing
// message. It deliberately avoids leaking raw Go error text or internal details:
//   - auth.go's ErrUnauthenticated and *PermissionError map to stable auth
//     messages;
//   - internal/errors AppError values map by ErrorCode to NotFound / Validation /
//     Conflict / Forbidden / Unauthorized phrasing, surfacing only AppError.Message;
//   - anything else collapses to a generic internal-error message.
//
// registerTool funnels every surfaced error through here.
func mapServiceError(err error) *mcp.CallToolResult {
	if err == nil {
		return nil
	}

	// MCP auth errors (auth.go).
	if errors.Is(err, ErrUnauthenticated) {
		return toolError("not authenticated")
	}
	var permErr *PermissionError
	if errors.As(err, &permErr) {
		return toolError(fmt.Sprintf("permission denied: missing %q", permErr.Code))
	}

	// Structured application errors (internal/errors). Surface only the curated
	// Message, never Details/Cause/StackTrace.
	if appErr, ok := apperrors.IsAppError(err); ok {
		switch appErr.Code {
		case apperrors.ErrCodeNotFound, apperrors.ErrCodeFileNotFound:
			return toolError(appErr.Message)
		case apperrors.ErrCodeValidation, apperrors.ErrCodeInvalidInput,
			apperrors.ErrCodeMissingField, apperrors.ErrCodeInvalidFormat,
			apperrors.ErrCodeValueTooLarge, apperrors.ErrCodeValueTooSmall:
			return toolError(appErr.Message)
		case apperrors.ErrCodeConflict, apperrors.ErrCodeAlreadyExists,
			apperrors.ErrCodeStaleResource, apperrors.ErrCodeResourceLocked:
			return toolError(appErr.Message)
		case apperrors.ErrCodeForbidden, apperrors.ErrCodeOperationNotAllowed,
			apperrors.ErrCodeAccountLocked:
			return toolError(appErr.Message)
		case apperrors.ErrCodeUnauthorized, apperrors.ErrCodeInvalidToken,
			apperrors.ErrCodeExpiredToken, apperrors.ErrCodeMissingAuth,
			apperrors.ErrCodeInvalidCredentials, apperrors.ErrCodeInvalidPassword:
			return toolError(appErr.Message)
		default:
			// Internal/system/unknown application errors: surface a stable,
			// non-leaking message rather than internal details.
			return toolError("the operation could not be completed")
		}
	}

	// Unknown error type: never echo raw text back to the client.
	return toolError("the operation could not be completed")
}
