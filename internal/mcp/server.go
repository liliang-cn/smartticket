package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ----------------------------------------------------------------------------
// Tool implementation conventions (READ THIS before adding domain tools)
// ----------------------------------------------------------------------------
//
// Every MCP tool in this package follows the same pattern. Domain tasks fill in
// the register<Domain>Tools functions; the auth_whoami tool below is the
// canonical reference implementation.
//
// PREFER registerTool. The registerTool helper (tools_helpers.go) applies the
// cross-cutting concerns — RBAC enforcement, panic recovery, latency timing,
// per-call structured logging, and clean service-error mapping — uniformly so
// individual tools do not reimplement them. A domain task should normally only
// provide:
//   - In/Out structs (the tool's MCP schema, see (2) below);
//   - a business closure func(ctx, in In) (Out, summary string, err error).
// and register it via:
//
//     registerTool(s, "<domain>_<action>", "<description>", "<permission-code>",
//         func(ctx context.Context, in In) (Out, string, error) { ... })
//
// Inside the closure: do NOT call RequirePermission (the helper already did it
// from the permission arg), do NOT recover panics, do NOT map errors — just call
// the Backend and return (out, summary, err). The helper turns any returned
// error into a clean IsError result via mapServiceError, recovers panics into a
// tool error, and logs {tool, user, latency, outcome}. Resolve the acting user
// via SessionFromContext when a Backend method needs a userID.
//
// Only drop down to mcp.AddTool directly for special cases the helper cannot
// express (e.g. streaming, custom multi-content results, or tools that must
// inspect the raw *mcp.CallToolRequest).
//
//  1. NAMING. Tools are named "<domain>_<action>", e.g. "ticket_create",
//     "knowledge_list", "rbac_assign_role". Use snake_case.
//
//  2. INPUT/OUTPUT TYPES. Each tool declares its own MCP-specific Input and
//     Output structs. DO NOT reuse the service-layer DTOs directly as the tool
//     schema — translate between them inside the closure. Annotate fields with
//     the `json` tag (wire name) and the `jsonschema` tag (human-readable
//     description); the SDK infers the JSON Schema from these via AddTool.
//     Optional fields should be pointers or use omitempty so the schema marks
//     them non-required appropriately.
//
//  3. RBAC. Pass the required permission code as registerTool's permission
//     argument; the helper enforces it before the closure runs. Pass "" only for
//     identity tools (like auth_whoami) that manage their own session checks.
//
//  4. ERROR MAPPING. Just return the error from the closure; registerTool routes
//     it through mapServiceError, which produces a clean, non-leaking message and
//     never surfaces raw Go error text or internal details. When dropping down to
//     mcp.AddTool directly, call mapServiceError yourself and never return a
//     protocol-level error for business failures.
//
// ----------------------------------------------------------------------------

// serverName / serverVersion identify this MCP server to clients.
const (
	serverName    = "smartticket"
	serverVersion = "0.1.0"
)

// allToolsets is the canonical list of toolset names, one per domain. Passing an
// empty slice to NewMCPServer enables all of them.
var allToolsets = []string{
	"ticket",
	"knowledge",
	"product",
	"service",
	"sla",
	"importexport",
	"user",
	"rbac",
	"customer",
	"subscription",
	"notification",
	"llm",
	"branding",
	"attachment",
	"macro",
	"automation",
	"team",
	"survey",
	"ticketmerge",
}

// toolsetRegistry maps a toolset name to its registration function.
var toolsetRegistry = map[string]func(s *mcp.Server, b Backend){
	"ticket":       registerTicketTools,
	"knowledge":    registerKnowledgeTools,
	"product":      registerProductTools,
	"service":      registerServiceTools,
	"sla":          registerSLATools,
	"importexport": registerImportExportTools,
	"user":         registerUserTools,
	"rbac":         registerRBACTools,
	"customer":     registerCustomerTools,
	"subscription": registerSubscriptionTools,
	"notification": registerNotificationTools,
	"llm":          registerLLMTools,
	"branding":     registerBrandingTools,
	"attachment":   registerAttachmentTools,
	"macro":        registerMacroTools,
	"automation":   registerAutomationTools,
	"team":         registerTeamTools,
	"survey":       registerSurveyTools,
	"ticketmerge":  registerTicketMergeTools,
}

// NewMCPServer builds an MCP server exposing the SmartTicket toolsets. If
// toolsets is empty, all toolsets are registered; otherwise only the named ones
// (unknown names are ignored). The auth_whoami tool is always registered.
func NewMCPServer(b Backend, toolsets []string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)

	// auth_whoami is always available regardless of toolset selection.
	registerAuthTools(s, b)

	enabled := toolsets
	if len(enabled) == 0 {
		enabled = allToolsets
	}
	for _, name := range enabled {
		if reg, ok := toolsetRegistry[name]; ok {
			reg(s, b)
		}
	}

	return s
}

// toolError builds a *mcp.CallToolResult representing a tool-level error with a
// clean, user-facing message. Domain handlers should funnel surfaced errors
// through here to avoid leaking internal Go error text.
func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

// textResult builds a successful *mcp.CallToolResult with a plain-text summary.
// The structured Out value is populated separately by the SDK from the handler's
// returned output.
func textResult(summary string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: summary}},
	}
}

// ----------------------------------------------------------------------------
// auth_whoami — canonical reference tool implementation
// ----------------------------------------------------------------------------

// WhoAmIInput is the input schema for auth_whoami. It takes no arguments.
type WhoAmIInput struct{}

// WhoAmIOutput is the structured output of auth_whoami.
type WhoAmIOutput struct {
	UserID      uint     `json:"user_id" jsonschema:"the authenticated user's numeric ID"`
	Permissions []string `json:"permissions" jsonschema:"effective permission codes held by the user"`
}

// registerAuthTools registers connection/identity tools. auth_whoami is the
// canonical template every domain tool follows: it declares In/Out structs and a
// business closure, then defers all RBAC/recover/logging/error-mapping to
// registerTool. It passes an empty permission code because identity introspection
// requires only an authenticated session (which it checks itself), not a specific
// permission.
func registerAuthTools(s *mcp.Server, _ Backend) {
	registerTool(s,
		"auth_whoami",
		"Return the identity and effective permissions of the authenticated MCP session.",
		"", // no specific permission required; the closure validates the session.
		func(ctx context.Context, _ WhoAmIInput) (WhoAmIOutput, string, error) {
			session, ok := SessionFromContext(ctx)
			if !ok || session == nil {
				return WhoAmIOutput{}, "", ErrUnauthenticated
			}

			perms := make([]string, 0, len(session.Permissions))
			for code := range session.Permissions {
				perms = append(perms, code)
			}

			out := WhoAmIOutput{
				UserID:      session.UserID,
				Permissions: perms,
			}
			summary := fmt.Sprintf("Authenticated as user #%d with %d permission(s).", out.UserID, len(perms))
			return out, summary, nil
		},
	)
}
