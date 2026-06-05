package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/macro"
	"github.com/company/smartticket/internal/models"
)

// macroResponse is the MCP-local view of a models.Macro. All slice/pointer
// fields carry omitempty so a zero value omits the field rather than emitting
// JSON null, keeping the output schema clean.
type macroResponse struct {
	ID         uint      `json:"id" jsonschema:"the macro's numeric ID"`
	Title      string    `json:"title" jsonschema:"the macro title"`
	Category   string    `json:"category,omitempty" jsonschema:"optional category"`
	Body       string    `json:"body" jsonschema:"the macro body template"`
	Actions    string    `json:"actions,omitempty" jsonschema:"optional JSON array of side-effect actions"`
	Shared     bool      `json:"shared" jsonschema:"whether the macro is visible to all team members"`
	OwnerID    uint      `json:"owner_id" jsonschema:"the user ID who owns this macro"`
	UsageCount int       `json:"usage_count" jsonschema:"number of times this macro has been applied"`
	CreatedAt  time.Time `json:"created_at" jsonschema:"when the macro was created"`
	UpdatedAt  time.Time `json:"updated_at" jsonschema:"when the macro was last updated"`
}

func macroResponseFrom(m *models.Macro) macroResponse {
	if m == nil {
		return macroResponse{}
	}
	return macroResponse{
		ID:         m.ID,
		Title:      m.Title,
		Category:   m.Category,
		Body:       m.Body,
		Actions:    m.Actions,
		Shared:     m.Shared,
		OwnerID:    m.OwnerID,
		UsageCount: m.UsageCount,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func macroResponsesFrom(ms []models.Macro) []macroResponse {
	if len(ms) == 0 {
		return nil
	}
	out := make([]macroResponse, len(ms))
	for i := range ms {
		out[i] = macroResponseFrom(&ms[i])
	}
	return out
}

// macroListOutput is the structured output for macro_list.
type macroListOutput struct {
	Macros []macroResponse `json:"macros,omitempty" jsonschema:"the visible macros"`
	Total  int             `json:"total" jsonschema:"total number of returned macros"`
}

// macroApplyOutput is the structured output for macro_apply.
type macroApplyOutput struct {
	Rendered string        `json:"rendered" jsonschema:"the macro body after placeholder substitution"`
	Actions  []macro.Action `json:"actions,omitempty" jsonschema:"the side-effect actions parsed from the macro"`
}

// ----------------------------------------------------------------------------
// Input types
// ----------------------------------------------------------------------------

type macroListInput struct{}

type macroGetInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the macro to retrieve"`
}

type macroCreateInput struct {
	Title    string `json:"title" jsonschema:"the macro title (required)"`
	Category string `json:"category,omitempty" jsonschema:"optional category label"`
	Body     string `json:"body" jsonschema:"the macro body template with optional {{variable}} placeholders (required)"`
	Actions  string `json:"actions,omitempty" jsonschema:"optional JSON array of side-effect actions e.g. [{\"type\":\"set_status\",\"params\":{\"status\":\"resolved\"}}]"`
	Shared   *bool  `json:"shared,omitempty" jsonschema:"whether the macro is visible to all team members (defaults true)"`
}

type macroUpdateInput struct {
	ID       uint    `json:"id" jsonschema:"the numeric ID of the macro to update"`
	Title    *string `json:"title,omitempty" jsonschema:"new title"`
	Category *string `json:"category,omitempty" jsonschema:"new category"`
	Body     *string `json:"body,omitempty" jsonschema:"new body template"`
	Actions  *string `json:"actions,omitempty" jsonschema:"new JSON actions array"`
	Shared   *bool   `json:"shared,omitempty" jsonschema:"new shared flag"`
}

type macroDeleteInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the macro to delete"`
}

type macroApplyInput struct {
	MacroID       uint   `json:"macro_id" jsonschema:"the numeric ID of the macro to apply"`
	TicketID      uint   `json:"ticket_id,omitempty" jsonschema:"optional ticket ID — used to populate {{ticket.id}} and {{ticket.subject}} placeholders"`
	TicketSubject string `json:"ticket_subject,omitempty" jsonschema:"optional ticket subject for placeholder substitution"`
	CustomerName  string `json:"customer_name,omitempty" jsonschema:"optional customer name for {{customer.name}} placeholder"`
	AgentName     string `json:"agent_name,omitempty" jsonschema:"optional agent name for {{agent.name}} placeholder"`
}

// ----------------------------------------------------------------------------
// Registration
// ----------------------------------------------------------------------------

func registerMacroTools(s *mcp.Server, b Backend) {
	registerTool(s, "macro_list",
		"List all macros visible to the authenticated user (shared macros plus the user's own private macros).",
		"macro:read",
		func(ctx context.Context, _ macroListInput) (macroListOutput, string, error) {
			return macroList(ctx, b)
		})

	registerTool(s, "macro_get",
		"Retrieve a single macro by numeric ID (visibility-checked).",
		"macro:read",
		func(ctx context.Context, in macroGetInput) (macroResponse, string, error) {
			return macroGet(ctx, b, in)
		})

	registerTool(s, "macro_create",
		"Create a new macro owned by the authenticated user.",
		"macro:write",
		func(ctx context.Context, in macroCreateInput) (macroResponse, string, error) {
			return macroCreate(ctx, b, in)
		})

	registerTool(s, "macro_update",
		"Update the fields of an existing macro. Only provided fields are changed.",
		"macro:write",
		func(ctx context.Context, in macroUpdateInput) (macroResponse, string, error) {
			return macroUpdate(ctx, b, in)
		})

	registerTool(s, "macro_delete",
		"Delete a macro by numeric ID (ownership or admin required for private macros).",
		"macro:write",
		func(ctx context.Context, in macroDeleteInput) (deleteOutput, string, error) {
			return macroDelete(ctx, b, in)
		})

	registerTool(s, "macro_apply",
		"Apply a macro: render its body with context variables and return the rendered text plus parsed side-effect actions.",
		"macro:write",
		func(ctx context.Context, in macroApplyInput) (macroApplyOutput, string, error) {
			return macroApply(ctx, b, in)
		})
}

// ----------------------------------------------------------------------------
// Handlers
// ----------------------------------------------------------------------------

func macroList(ctx context.Context, b Backend) (macroListOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return macroListOutput{}, "", ErrUnauthenticated
	}
	ms, err := b.ListMacros(session.UserID)
	if err != nil {
		return macroListOutput{}, "", err
	}
	out := macroListOutput{Macros: macroResponsesFrom(ms), Total: len(ms)}
	return out, fmt.Sprintf("Listed %d macro(s).", len(ms)), nil
}

func macroGet(ctx context.Context, b Backend, in macroGetInput) (macroResponse, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return macroResponse{}, "", ErrUnauthenticated
	}
	m, err := b.GetMacro(session.UserID, in.ID)
	if err != nil {
		return macroResponse{}, "", err
	}
	return macroResponseFrom(m), fmt.Sprintf("Retrieved macro #%d (%s).", m.ID, m.Title), nil
}

func macroCreate(ctx context.Context, b Backend, in macroCreateInput) (macroResponse, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return macroResponse{}, "", ErrUnauthenticated
	}
	req := macro.CreateRequest{
		Title:    in.Title,
		Category: in.Category,
		Body:     in.Body,
		Actions:  in.Actions,
		Shared:   in.Shared,
	}
	m, err := b.CreateMacro(session.UserID, req)
	if err != nil {
		return macroResponse{}, "", err
	}
	return macroResponseFrom(m), fmt.Sprintf("Created macro #%d (%s).", m.ID, m.Title), nil
}

func macroUpdate(ctx context.Context, b Backend, in macroUpdateInput) (macroResponse, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return macroResponse{}, "", ErrUnauthenticated
	}
	req := macro.UpdateRequest{
		Title:    in.Title,
		Category: in.Category,
		Body:     in.Body,
		Actions:  in.Actions,
		Shared:   in.Shared,
	}
	m, err := b.UpdateMacro(session.UserID, in.ID, req)
	if err != nil {
		return macroResponse{}, "", err
	}
	return macroResponseFrom(m), fmt.Sprintf("Updated macro #%d (%s).", m.ID, m.Title), nil
}

func macroDelete(ctx context.Context, b Backend, in macroDeleteInput) (deleteOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return deleteOutput{}, "", ErrUnauthenticated
	}
	if err := b.DeleteMacro(session.UserID, in.ID); err != nil {
		return deleteOutput{}, "", err
	}
	return deleteOutput{ID: in.ID, Deleted: true}, fmt.Sprintf("Deleted macro #%d.", in.ID), nil
}

func macroApply(ctx context.Context, b Backend, in macroApplyInput) (macroApplyOutput, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return macroApplyOutput{}, "", ErrUnauthenticated
	}

	// Build the RenderContext from the tool input. The caller provides what they
	// know; unset fields produce empty substitutions (consistent with Render).
	ticketIDStr := ""
	if in.TicketID > 0 {
		ticketIDStr = fmt.Sprintf("%d", in.TicketID)
	}
	rctx := macro.RenderContext{
		CustomerName:  in.CustomerName,
		AgentName:     in.AgentName,
		TicketID:      ticketIDStr,
		TicketSubject: in.TicketSubject,
	}

	rendered, actions, err := b.ApplyMacro(in.MacroID, session.UserID, rctx)
	if err != nil {
		return macroApplyOutput{}, "", err
	}
	return macroApplyOutput{Rendered: rendered, Actions: actions},
		fmt.Sprintf("Applied macro #%d.", in.MacroID), nil
}
