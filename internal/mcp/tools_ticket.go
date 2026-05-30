package mcp

import "github.com/modelcontextprotocol/go-sdk/mcp"

// registerTicketTools registers the ticket-domain MCP tools.
// See server.go for the tool implementation conventions and auth_whoami template.
func registerTicketTools(s *mcp.Server, b Backend) {
	// TODO: filled in by domain task (I2).
	_ = s
	_ = b
}
