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
//  1. NAMING. Tools are named "<domain>_<action>", e.g. "ticket_create",
//     "knowledge_list", "rbac_assign_role". Use snake_case.
//
//  2. INPUT/OUTPUT TYPES. Each tool declares its own MCP-specific Input and
//     Output structs. DO NOT reuse the service-layer DTOs directly as the tool
//     schema — translate between them inside the handler. Annotate fields with
//     the `json` tag (wire name) and the `jsonschema` tag (human-readable
//     description); the SDK infers the JSON Schema from these via AddTool.
//     Optional fields should be pointers or use omitempty so the schema marks
//     them non-required appropriately.
//
//  3. HANDLER SIGNATURE. Use the typed handler form:
//
//        func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error)
//
//     Register it with mcp.AddTool(s, &mcp.Tool{Name: ..., Description: ...}, handler).
//     You may return a nil *mcp.CallToolResult and let the SDK synthesize the
//     result from the Out value (it fills StructuredContent and a JSON text
//     summary automatically). To provide a friendlier text summary, return a
//     *mcp.CallToolResult whose Content carries a *mcp.TextContent.
//
//  4. RBAC FIRST. The handler must call RequirePermission(ctx, "<code>") before
//     touching the Backend. Resolve the acting user via SessionFromContext when
//     a Backend method needs a userID.
//
//  5. ERROR MAPPING. Translate Backend/service errors into tool errors, never
//     protocol errors: return them via toolError(...) which sets IsError and
//     packs a clean message into Content. Do NOT leak raw Go error text or
//     internal details (wrap with a stable, user-facing message). Returning a
//     non-nil error from the handler also produces a tool error automatically,
//     but prefer toolError for control over the surfaced message.
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
// template every domain tool follows.
func registerAuthTools(s *mcp.Server, _ Backend) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "auth_whoami",
		Description: "Return the identity and effective permissions of the authenticated MCP session.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ WhoAmIInput) (*mcp.CallToolResult, WhoAmIOutput, error) {
		// Step 1: resolve the session. auth_whoami needs no specific permission,
		// but it still requires an authenticated connection.
		session, ok := SessionFromContext(ctx)
		if !ok || session == nil {
			return toolError("not authenticated"), WhoAmIOutput{}, nil
		}

		// Step 2: build the structured output.
		perms := make([]string, 0, len(session.Permissions))
		for code := range session.Permissions {
			perms = append(perms, code)
		}

		out := WhoAmIOutput{
			UserID:      session.UserID,
			Permissions: perms,
		}

		// Step 3: return a friendly text summary alongside the structured output.
		summary := fmt.Sprintf("Authenticated as user #%d with %d permission(s).", out.UserID, len(perms))
		return textResult(summary), out, nil
	})
}
