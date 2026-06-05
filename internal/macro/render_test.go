package macro

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRender_CustomerName(t *testing.T) {
	ctx := RenderContext{CustomerName: "Alice"}
	out := Render("Hello {{customer.name}}!", ctx)
	require.Equal(t, "Hello Alice!", out)
}

func TestRender_AgentName(t *testing.T) {
	ctx := RenderContext{AgentName: "Bob"}
	out := Render("Regards, {{agent.name}}", ctx)
	require.Equal(t, "Regards, Bob", out)
}

func TestRender_TicketID(t *testing.T) {
	ctx := RenderContext{TicketID: "42"}
	out := Render("Ticket #{{ticket.id}} received.", ctx)
	require.Equal(t, "Ticket #42 received.", out)
}

func TestRender_TicketSubject(t *testing.T) {
	ctx := RenderContext{TicketSubject: "Printer broken"}
	out := Render("Re: {{ticket.subject}}", ctx)
	require.Equal(t, "Re: Printer broken", out)
}

func TestRender_UnknownVariableBlanked(t *testing.T) {
	out := Render("Value: {{unknown.var}}", RenderContext{})
	require.Equal(t, "Value: ", out)
}

func TestRender_NoVariables(t *testing.T) {
	body := "Plain text without variables."
	out := Render(body, RenderContext{})
	require.Equal(t, body, out)
}

func TestRender_MultipleVars(t *testing.T) {
	ctx := RenderContext{
		CustomerName:  "Alice",
		AgentName:     "Bob",
		TicketID:      "7",
		TicketSubject: "Login issue",
	}
	body := "Hi {{customer.name}}, ticket {{ticket.id}} ({{ticket.subject}}) assigned to {{agent.name}}."
	out := Render(body, ctx)
	require.Equal(t, "Hi Alice, ticket 7 (Login issue) assigned to Bob.", out)
}
