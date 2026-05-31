package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/ticket"
)

// ----------------------------------------------------------------------------
// Local output views
// ----------------------------------------------------------------------------
//
// The MCP tools must not reuse ticket.TicketResponse directly as their Output:
// its Tags ([]string) and CustomFields (map[string]interface{}) fields lack
// `omitempty`, so when they are nil they marshal to JSON null, which the go-sdk
// rejects against the inferred array/object output schema (failing both the
// success path on real nil data and every error path, which returns a zero Out).
// ticketResponse is a flat, MCP-local view whose slice/map/pointer fields all
// carry `omitempty`, so a nil value is omitted rather than emitted as null.

// ticketResponse is the cycle-safe, schema-safe MCP view of a ticket.
type ticketResponse struct {
	ID              uint                   `json:"id" jsonschema:"the ticket's numeric ID"`
	TicketNumber    string                 `json:"ticket_number" jsonschema:"the human-readable ticket number"`
	Title           string                 `json:"title" jsonschema:"the ticket title"`
	Description     string                 `json:"description" jsonschema:"the ticket description"`
	Status          string                 `json:"status" jsonschema:"the ticket status"`
	Priority        string                 `json:"priority" jsonschema:"the ticket priority"`
	Severity        string                 `json:"severity" jsonschema:"the ticket severity"`
	Category        string                 `json:"category,omitempty" jsonschema:"the ticket category"`
	Type            string                 `json:"type,omitempty" jsonschema:"the ticket type"`
	ProductID       *uint                  `json:"product_id,omitempty" jsonschema:"associated product ID, if any"`
	ServiceID       *uint                  `json:"service_id,omitempty" jsonschema:"associated service ID, if any"`
	CustomerID      *uint                  `json:"customer_id,omitempty" jsonschema:"the owning customer organization's ID, if any"`
	CustomerName    string                 `json:"customer_name,omitempty" jsonschema:"the owning customer organization's name, if any"`
	AssignedTo      *uint                  `json:"assigned_to,omitempty" jsonschema:"the user ID the ticket is assigned to, if any"`
	RequesterName   string                 `json:"requester_name" jsonschema:"the requester's name"`
	RequesterEmail  string                 `json:"requester_email" jsonschema:"the requester's email address"`
	Tags            []string               `json:"tags,omitempty" jsonschema:"the ticket tags"`
	CustomFields    map[string]interface{} `json:"custom_fields,omitempty" jsonschema:"the ticket custom fields"`
	IsDeleted       bool                   `json:"is_deleted" jsonschema:"whether the ticket is soft-deleted"`
	CreatedAt       time.Time              `json:"created_at" jsonschema:"when the ticket was created"`
	UpdatedAt       time.Time              `json:"updated_at" jsonschema:"when the ticket was last updated"`
	ResolutionTime  *time.Time             `json:"resolution_time,omitempty" jsonschema:"when the ticket resolution time was recorded, if any"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty" jsonschema:"when the ticket was resolved, if any"`
	DueDate         *time.Time             `json:"due_date,omitempty" jsonschema:"the ticket's SLA due date, if any"`
	SLAStatus       string                 `json:"sla_status,omitempty" jsonschema:"the ticket's SLA status"`
	MessageCount    int64                  `json:"message_count" jsonschema:"number of messages on the ticket"`
	AttachmentCount int64                  `json:"attachment_count" jsonschema:"number of attachments on the ticket"`
}

// ticketResponseFrom converts a service-layer ticket.TicketResponse into the
// schema-safe MCP view. The embedded *UserInfo association is dropped (its ID is
// already carried by AssignedTo).
func ticketResponseFrom(r *ticket.TicketResponse) ticketResponse {
	if r == nil {
		return ticketResponse{}
	}
	return ticketResponse{
		ID:              r.ID,
		TicketNumber:    r.TicketNumber,
		Title:           r.Title,
		Description:     r.Description,
		Status:          r.Status,
		Priority:        r.Priority,
		Severity:        r.Severity,
		Category:        r.Category,
		Type:            r.Type,
		ProductID:       r.ProductID,
		ServiceID:       r.ServiceID,
		CustomerID:      r.CustomerID,
		CustomerName:    r.CustomerName,
		AssignedTo:      r.AssignedTo,
		RequesterName:   r.RequesterName,
		RequesterEmail:  r.RequesterEmail,
		Tags:            r.Tags,
		CustomFields:    r.CustomFields,
		IsDeleted:       r.IsDeleted,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		ResolutionTime:  r.ResolutionTime,
		ResolvedAt:      r.ResolvedAt,
		DueDate:         r.DueDate,
		SLAStatus:       r.SLAStatus,
		MessageCount:    r.MessageCount,
		AttachmentCount: r.AttachmentCount,
	}
}

// ticketListResponse is the schema-safe MCP view of a paginated ticket list. The
// Data slice carries omitempty so a nil page marshals to an omitted field rather
// than JSON null.
type ticketListResponse struct {
	Data       []ticketResponse `json:"data,omitempty" jsonschema:"the page of tickets"`
	Total      int64            `json:"total" jsonschema:"total number of matching tickets"`
	Page       int              `json:"page" jsonschema:"the 1-based page number returned"`
	PageSize   int              `json:"page_size" jsonschema:"the page size used"`
	TotalPages int              `json:"total_pages" jsonschema:"total number of pages available"`
}

// ticketListResponseFrom converts a service-layer ticket.TicketListResponse into
// the schema-safe MCP view.
func ticketListResponseFrom(r *ticket.TicketListResponse) ticketListResponse {
	if r == nil {
		return ticketListResponse{}
	}
	out := ticketListResponse{
		Total:      r.Total,
		Page:       r.Page,
		PageSize:   r.PageSize,
		TotalPages: r.TotalPages,
	}
	if len(r.Data) > 0 {
		out.Data = make([]ticketResponse, len(r.Data))
		for i := range r.Data {
			out.Data[i] = ticketResponseFrom(&r.Data[i])
		}
	}
	return out
}

// registerTicketTools registers the ticket-domain MCP tools.
// See server.go for the tool implementation conventions and auth_whoami template.
func registerTicketTools(s *mcp.Server, b Backend) {
	registerTool(s,
		"ticket_create",
		"Create a new support ticket.",
		"ticket:write",
		func(ctx context.Context, in ticketCreateInput) (ticketResponse, string, error) {
			return ticketCreate(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_get",
		"Fetch a single ticket by its numeric ID.",
		"ticket:read",
		func(ctx context.Context, in ticketGetInput) (ticketResponse, string, error) {
			return ticketGet(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_list",
		"List tickets with optional filtering and pagination.",
		"ticket:read",
		func(ctx context.Context, in ticketListInput) (ticketListResponse, string, error) {
			return ticketList(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_update",
		"Update an existing ticket's fields.",
		"ticket:write",
		func(ctx context.Context, in ticketUpdateInput) (ticketResponse, string, error) {
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

	registerTool(s,
		"ticket_message_create",
		"Post a reply (message) on a ticket.",
		"ticket:write",
		func(ctx context.Context, in ticketMessageCreateInput) (messageResponse, string, error) {
			return ticketMessageCreate(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_message_list",
		"List the messages on a ticket.",
		"ticket:read",
		func(ctx context.Context, in ticketMessageListInput) (messageListOutput, string, error) {
			return ticketMessageList(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_sla",
		"Get the SLA policy governing a ticket: the matched rule/template, response and resolution targets, due date and within/breached status.",
		"ticket:read",
		func(ctx context.Context, in ticketGetInput) (ticketSLAOutput, string, error) {
			return ticketSLA(ctx, b, in)
		},
	)

	registerTool(s,
		"ticket_events",
		"List a ticket's activity log: creation, status/priority/severity changes, assignment, replies and notes (oldest first).",
		"ticket:read",
		func(ctx context.Context, in ticketGetInput) (ticketEventsOutput, string, error) {
			return ticketEvents(ctx, b, in)
		},
	)
}

// ----------------------------------------------------------------------------
// ticket_sla / ticket_events schemas + closures
// ----------------------------------------------------------------------------

type ticketSLAOutput struct {
	TicketID          uint       `json:"ticket_id" jsonschema:"the ticket ID"`
	Priority          string     `json:"priority" jsonschema:"the ticket priority"`
	Severity          string     `json:"severity" jsonschema:"the ticket severity"`
	Source            string     `json:"source" jsonschema:"'rule' (a matching SLA rule) or 'default'"`
	PolicyName        string     `json:"policy_name" jsonschema:"the governing SLA template/rule name, or 'Default policy'"`
	ResponseMinutes   int        `json:"response_minutes" jsonschema:"response target in minutes"`
	ResolutionMinutes int        `json:"resolution_minutes" jsonschema:"resolution target in minutes"`
	BusinessOnly      bool       `json:"business_only" jsonschema:"whether targets count business hours only"`
	DueDate           *time.Time `json:"due_date,omitempty" jsonschema:"the ticket's due date, if set"`
	SLAStatus         string     `json:"sla_status,omitempty" jsonschema:"within, warning, or breached"`
}

type ticketEventView struct {
	ID        uint      `json:"id" jsonschema:"event ID"`
	Action    string    `json:"action" jsonschema:"event type: created, status, priority, severity, assigned, replied, note"`
	Summary   string    `json:"summary" jsonschema:"human-readable description of the event"`
	ActorName string    `json:"actor_name,omitempty" jsonschema:"who performed the action"`
	ActorRole string    `json:"actor_role,omitempty" jsonschema:"the actor's role"`
	CreatedAt time.Time `json:"created_at" jsonschema:"when it happened"`
}

type ticketEventsOutput struct {
	Events []ticketEventView `json:"events,omitempty" jsonschema:"the activity log, oldest first"`
	Total  int               `json:"total" jsonschema:"number of events"`
}

func ticketSLA(ctx context.Context, b Backend, in ticketGetInput) (ticketSLAOutput, string, error) {
	r, err := b.GetTicketSLA(sessionActor(ctx), in.ID)
	if err != nil {
		return ticketSLAOutput{}, "", err
	}
	out := ticketSLAOutput{
		TicketID: r.TicketID, Priority: r.Priority, Severity: r.Severity,
		Source: r.Source, PolicyName: r.PolicyName,
		ResponseMinutes: r.ResponseMinutes, ResolutionMinutes: r.ResolutionMinutes,
		BusinessOnly: r.BusinessOnly, DueDate: r.DueDate, SLAStatus: r.SLAStatus,
	}
	return out, fmt.Sprintf("Ticket #%d SLA: %s (status %s).", r.TicketID, r.PolicyName, r.SLAStatus), nil
}

func ticketEvents(ctx context.Context, b Backend, in ticketGetInput) (ticketEventsOutput, string, error) {
	evs, err := b.ListTicketEvents(sessionActor(ctx), in.ID)
	if err != nil {
		return ticketEventsOutput{}, "", err
	}
	views := make([]ticketEventView, 0, len(evs))
	for i := range evs {
		e := &evs[i]
		views = append(views, ticketEventView{
			ID: e.ID, Action: e.Action, Summary: e.Summary,
			ActorName: e.ActorName, ActorRole: e.ActorRole, CreatedAt: e.CreatedAt,
		})
	}
	return ticketEventsOutput{Events: views, Total: len(views)},
		fmt.Sprintf("Ticket #%d has %d activity event(s).", in.ID, len(views)), nil
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
func ticketCreate(ctx context.Context, b Backend, in ticketCreateInput) (ticketResponse, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return ticketResponse{}, "", ErrUnauthenticated
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

	resp, err := b.CreateTicket(session.Actor(), session.UserID, req)
	if err != nil {
		return ticketResponse{}, "", err
	}
	return ticketResponseFrom(resp), fmt.Sprintf("created ticket #%d (%s)", resp.ID, resp.TicketNumber), nil
}

// ----------------------------------------------------------------------------
// ticket_get
// ----------------------------------------------------------------------------

// ticketGetInput is the MCP input schema for ticket_get.
type ticketGetInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the ticket to fetch"`
}

// ticketGet fetches a single ticket by ID.
func ticketGet(ctx context.Context, b Backend, in ticketGetInput) (ticketResponse, string, error) {
	resp, err := b.GetTicket(sessionActor(ctx), in.ID)
	if err != nil {
		return ticketResponse{}, "", err
	}
	return ticketResponseFrom(resp), fmt.Sprintf("fetched ticket #%d (%s)", resp.ID, resp.TicketNumber), nil
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
func ticketList(ctx context.Context, b Backend, in ticketListInput) (ticketListResponse, string, error) {
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

	resp, err := b.ListTickets(sessionActor(ctx), in.Page, in.PageSize, filters)
	if err != nil {
		return ticketListResponse{}, "", err
	}
	return ticketListResponseFrom(resp), fmt.Sprintf("listed %d of %d ticket(s) (page %d)", len(resp.Data), resp.Total, resp.Page), nil
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
func ticketUpdate(ctx context.Context, b Backend, in ticketUpdateInput) (ticketResponse, string, error) {
	session, ok := SessionFromContext(ctx)
	if !ok || session == nil {
		return ticketResponse{}, "", ErrUnauthenticated
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

	resp, err := b.UpdateTicket(session.Actor(), in.ID, session.UserID, req)
	if err != nil {
		return ticketResponse{}, "", err
	}
	return ticketResponseFrom(resp), fmt.Sprintf("updated ticket #%d (%s)", resp.ID, resp.TicketNumber), nil
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
func ticketDelete(ctx context.Context, b Backend, in ticketDeleteInput) (ticketDeleteOutput, string, error) {
	if err := b.DeleteTicket(sessionActor(ctx), in.ID); err != nil {
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
func ticketAssign(ctx context.Context, b Backend, in ticketAssignInput) (ticketAssignOutput, string, error) {
	if err := b.AssignTicket(sessionActor(ctx), in.ID, in.AssignedTo); err != nil {
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
	Stats map[string]interface{} `json:"stats,omitempty" jsonschema:"aggregate ticket statistics keyed by metric name"`
}

// ticketStats returns aggregate ticket statistics.
func ticketStats(ctx context.Context, b Backend, _ ticketStatsInput) (ticketStatsOutput, string, error) {
	stats, err := b.GetTicketStats(sessionActor(ctx))
	if err != nil {
		return ticketStatsOutput{}, "", err
	}
	return ticketStatsOutput{Stats: stats}, "fetched ticket statistics", nil
}
