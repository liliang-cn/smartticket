// Package macro implements canned responses (macros) for the SmartTicket
// helpdesk. A macro is a reusable text template with optional {{variable}}
// placeholders and side-effect actions (set status, add tag, etc.).
package macro

import "regexp"

// varRe matches {{anything}} placeholders in macro body text.
var varRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// RenderContext carries the runtime values substituted into a macro body.
type RenderContext struct {
	CustomerName  string
	AgentName     string
	TicketID      string
	TicketSubject string
}

// Render substitutes known {{variable}} placeholders in body with values from
// ctx. Unknown placeholders are replaced with an empty string.
func Render(body string, ctx RenderContext) string {
	return varRe.ReplaceAllStringFunc(body, func(match string) string {
		// match is the full "{{name}}" token; extract inner key.
		inner := match[2 : len(match)-2]
		switch inner {
		case "customer.name":
			return ctx.CustomerName
		case "agent.name":
			return ctx.AgentName
		case "ticket.id":
			return ctx.TicketID
		case "ticket.subject":
			return ctx.TicketSubject
		default:
			return ""
		}
	})
}
