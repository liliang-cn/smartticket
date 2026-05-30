package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/ticket"
)

// registerTicketTools registers the ticket-domain MCP tools.
// See server.go for the tool implementation conventions and auth_whoami template.
func registerTicketTools(s *mcp.Server, b Backend) {
	registerTool(s,
		"ticket_create",
		"Create a new support ticket.",
		"ticket:write",
		func(ctx context.Context, in ticketCreateInput) (ticket.TicketResponse, string, error) {
			return ticketCreate(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_get",
		"Fetch a single ticket by its numeric ID.",
		"ticket:read",
		func(ctx context.Context, in ticketGetInput) (ticket.TicketResponse, string, error) {
			return ticketGet(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_list",
		"List tickets with optional filtering and pagination.",
		"ticket:read",
		func(ctx context.Context, in ticketListInput) (ticket.TicketListResponse, string, error) {
			return ticketList(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_update",
		"Update an existing ticket's fields.",
		"ticket:write",
		func(ctx context.Context, in ticketUpdateInput) (ticket.TicketResponse, string, error) {
			return ticketUpdate(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_delete",
		"Soft-delete a ticket by its numeric ID.",
		"ticket:write",
		func(ctx context.Context, in ticketDeleteInput) (ticketDeleteOutput, string, error) {
			return ticketDelete(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_assign",
		"Assign a ticket to a user.",
		"ticket:write",
		func(ctx context.Context, in ticketAssignInput) (ticketAssignOutput, string, error) {
			return ticketAssign(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_stats",
		"Return aggregate ticket statistics (counts by status, priority, overdue, etc.).",
		"ticket:read",
		func(ctx context.Context, in ticketStatsInput) (ticketStatsOutput, string, error) {
			return ticketStats(ctx, b, in)
		},
	)
}

// ----------------------------------------------------------------------------
// ticket_create
// ----------------------------------------------------------------------------

// ticketCreateInput is the MCP input schema for ticket_create.
type ticketCreateInput struct {
	Title          string `json:"title" jsonschema:"the ticket title (required)"`
	Description    string `json:"description" jsonschema:"the ticket description (required)"`
	Priority       string `json:"priority,omitempty" jsonschema:"priority: one of low, medium, high, critical (defaults to medium)"`
	Severity       string `json:"severity,omitempty" jsonschema:"severity: one of trivial, minor, major, critical (defaults to minor)"`
	Category       string `json:"category,omitempty" jsonschema:"optional category label"`
	Type           string `json:"type,omitempty" jsonschema:"optional ticket type"`
	ProductID      *uint  `json:"product_id,omitempty" jsonschema:"optional associated product ID"`
	ServiceID      *uint  `json:"service_id,omitempty" jsonschema:"optional associated service ID"`
	RequesterName  string `json:"requester_name" jsonschema:"the requester's name (required)"`
	RequesterEmail string `json:"requester_email" jsonschema:"the requester's email address (required)"`
	Tags           string `json:"tags,omitempty" jsonschema:"optional tags as a JSON array string"`
	CustomFields   string `json:"custom_fields,omitempty" jsonschema:"optional custom fields as a JSON object string"`
}

// ticketCreate creates a ticket on behalf of the acting session user.
func ticketCreate(ctx context.Context, b Backend, in ticketCreateInput) (ticket.TicketResponse, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return ticket.TicketResponse{}, "", ErrUnauthenticated
	}

	req := &ticket.CreateTicketRequest{
		Title:          in.Title,
		Description:    in.Description,
		Priority:       in.Priority,
		Severity:       in.Severity,
		Category:       in.Category,
		Type:           in.Type,
		ProductID:      in.ProductID,
		ServiceID:      in.ServiceID,
		RequesterName:  in.RequesterName,
		RequesterEmail: in.RequesterEmail,
		Tags:           in.Tags,
		CustomFields:   in.CustomFields,
	}

	resp, err := b.CreateTicket(session.UserID, req)
	if err != nil {
		return ticket.TicketResponse{}, "", err
	}
	return *resp, fmt.Sprintf("created ticket #%d (%s)", resp.ID, resp.TicketNumber), nil
}

// ----------------------------------------------------------------------------
// ticket_get
// ----------------------------------------------------------------------------

// ticketGetInput is the MCP input schema for ticket_get.
type ticketGetInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the ticket to fetch"`
}

// ticketGet fetches a single ticket by ID.
func ticketGet(_ context.Context, b Backend, in ticketGetInput) (ticket.TicketResponse, string, error) {
	resp, err := b.GetTicket(in.ID)
	if err != nil {
		return ticket.TicketResponse{}, "", err
	}
	return *resp, fmt.Sprintf("fetched ticket #%d (%s)", resp.ID, resp.TicketNumber), nil
}

// ----------------------------------------------------------------------------
// ticket_list
// ----------------------------------------------------------------------------

// ticketListInput is the MCP input schema for ticket_list.
type ticketListInput struct {
	Page       int    `json:"page,omitempty" jsonschema:"1-based page number (defaults to 1)"`
	PageSize   int    `json:"page_size,omitempty" jsonschema:"page size, max 100 (defaults to 20)"`
	Status     string `json:"status,omitempty" jsonschema:"filter by status: open, in_progress, resolved, closed, cancelled"`
	Priority   string `json:"priority,omitempty" jsonschema:"filter by priority: low, medium, high, critical"`
	Category   string `json:"category,omitempty" jsonschema:"filter by category"`
	AssignedTo uint   `json:"assigned_to,omitempty" jsonschema:"filter by the user ID a ticket is assigned to"`
	Search     string `json:"search,omitempty" jsonschema:"free-text search over title, description, requester name and email"`
}

// ticketList lists tickets with optional filters and pagination.
func ticketList(_ context.Context, b Backend, in ticketListInput) (ticket.TicketListResponse, string, error) {
	filters := map[string]interface{}{}
	if in.Status != "" {
		filters["status"] = in.Status
	}
	if in.Priority != "" {
		filters["priority"] = in.Priority
	}
	if in.Category != "" {
		filters["category"] = in.Category
	}
	if in.AssignedTo > 0 {
		filters["assigned_to"] = in.AssignedTo
	}
	if in.Search != "" {
		filters["search"] = in.Search
	}

	resp, err := b.ListTickets(in.Page, in.PageSize, filters)
	if err != nil {
		return ticket.TicketListResponse{}, "", err
	}
	return *resp, fmt.Sprintf("listed %d of %d ticket(s) (page %d)", len(resp.Data), resp.Total, resp.Page), nil
}

// ----------------------------------------------------------------------------
// ticket_update
// ----------------------------------------------------------------------------

// ticketUpdateInput is the MCP input schema for ticket_update.
type ticketUpdateInput struct {
	ID             uint   `json:"id" jsonschema:"the numeric ID of the ticket to update"`
	Title          string `json:"title,omitempty" jsonschema:"new title"`
	Description    string `json:"description,omitempty" jsonschema:"new description"`
	Status         string `json:"status,omitempty" jsonschema:"new status: open, in_progress, resolved, closed, cancelled"`
	Priority       string `json:"priority,omitempty" jsonschema:"new priority: low, medium, high, critical"`
	Severity       string `json:"severity,omitempty" jsonschema:"new severity: trivial, minor, major, critical"`
	Category       string `json:"category,omitempty" jsonschema:"new category"`
	Type           string `json:"type,omitempty" jsonschema:"new ticket type"`
	ProductID      *uint  `json:"product_id,omitempty" jsonschema:"new associated product ID"`
	ServiceID      *uint  `json:"service_id,omitempty" jsonschema:"new associated service ID"`
	AssignedTo     *uint  `json:"assigned_to,omitempty" jsonschema:"user ID to assign the ticket to"`
	RequesterName  string `json:"requester_name,omitempty" jsonschema:"new requester name"`
	RequesterEmail string `json:"requester_email,omitempty" jsonschema:"new requester email"`
	Tags           string `json:"tags,omitempty" jsonschema:"tags as a JSON array string"`
	CustomFields   string `json:"custom_fields,omitempty" jsonschema:"custom fields as a JSON object string"`
}

// ticketUpdate updates a ticket on behalf of the acting session user.
func ticketUpdate(ctx context.Context, b Backend, in ticketUpdateInput) (ticket.TicketResponse, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return ticket.TicketResponse{}, "", ErrUnauthenticated
	}

	req := &ticket.UpdateTicketRequest{
		Title:          in.Title,
		Description:    in.Description,
		Status:         in.Status,
		Priority:       in.Priority,
		Severity:       in.Severity,
		Category:       in.Category,
		Type:           in.Type,
		ProductID:      in.ProductID,
		ServiceID:      in.ServiceID,
		AssignedTo:     in.AssignedTo,
		RequesterName:  in.RequesterName,
		RequesterEmail: in.RequesterEmail,
		Tags:           in.Tags,
		CustomFields:   in.CustomFields,
	}

	resp, err := b.UpdateTicket(in.ID, session.UserID, req)
	if err != nil {
		return ticket.TicketResponse{}, "", err
	}
	return *resp, fmt.Sprintf("updated ticket #%d (%s)", resp.ID, resp.TicketNumber), nil
}

// ----------------------------------------------------------------------------
// ticket_delete
// ----------------------------------------------------------------------------

// ticketDeleteInput is the MCP input schema for ticket_delete.
type ticketDeleteInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the ticket to delete"`
}

// ticketDeleteOutput is the structured output of ticket_delete.
type ticketDeleteOutput struct {
	ID      uint `json:"id" jsonschema:"the numeric ID of the deleted ticket"`
	Deleted bool `json:"deleted" jsonschema:"whether the ticket was deleted"`
}

// ticketDelete soft-deletes a ticket by ID.
func ticketDelete(_ context.Context, b Backend, in ticketDeleteInput) (ticketDeleteOutput, string, error) {
	if err := b.DeleteTicket(in.ID); err != nil {
		return ticketDeleteOutput{}, "", err
	}
	return ticketDeleteOutput{ID: in.ID, Deleted: true}, fmt.Sprintf("deleted ticket #%d", in.ID), nil
}

// ----------------------------------------------------------------------------
// ticket_assign
// ----------------------------------------------------------------------------

// ticketAssignInput is the MCP input schema for ticket_assign.
type ticketAssignInput struct {
	ID         uint `json:"id" jsonschema:"the numeric ID of the ticket to assign"`
	AssignedTo uint `json:"assigned_to" jsonschema:"the user ID to assign the ticket to"`
}

// ticketAssignOutput is the structured output of ticket_assign.
type ticketAssignOutput struct {
	ID         uint `json:"id" jsonschema:"the numeric ID of the assigned ticket"`
	AssignedTo uint `json:"assigned_to" jsonschema:"the user ID the ticket was assigned to"`
}

// ticketAssign assigns a ticket to a user.
func ticketAssign(_ context.Context, b Backend, in ticketAssignInput) (ticketAssignOutput, string, error) {
	if err := b.AssignTicket(in.ID, in.AssignedTo); err != nil {
		return ticketAssignOutput{}, "", err
	}
	return ticketAssignOutput{ID: in.ID, AssignedTo: in.AssignedTo},
		fmt.Sprintf("assigned ticket #%d to user #%d", in.ID, in.AssignedTo), nil
}

// ----------------------------------------------------------------------------
// ticket_stats
// ----------------------------------------------------------------------------

// ticketStatsInput is the MCP input schema for ticket_stats. It takes no arguments.
type ticketStatsInput struct{}

// ticketStatsOutput is the structured output of ticket_stats. Statistics are
// returned as a free-form map mirroring the service layer's stats shape.
type ticketStatsOutput struct {
	Stats map[string]interface{} `json:"stats" jsonschema:"aggregate ticket statistics keyed by metric name"`
}

// ticketStats returns aggregate ticket statistics.
func ticketStats(_ context.Context, b Backend, _ ticketStatsInput) (ticketStatsOutput, string, error) {
	stats, err := b.GetTicketStats()
	if err != nil {
		return ticketStatsOutput{}, "", err
	}
	return ticketStatsOutput{Stats: stats}, "fetched ticket statistics", nil
}
