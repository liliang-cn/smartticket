package mcp

import "github.com/modelcontextprotocol/go-sdk/mcp"

// registerRBACTools registers the RBAC-domain MCP tools.
// See server.go for the tool implementation conventions and auth_whoami template.
func registerRBACTools(s *mcp.Server, b Backend) {
	// TODO: filled in by domain task (I9).
	_ = s
	_ = b
}
