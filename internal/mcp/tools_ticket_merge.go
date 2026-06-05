package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/ticket"
)

// ticketLinkResponse is the MCP-local view of a ticket.LinkResponse. The
// nested OtherTicket struct carries three scalar fields; no slice/map fields
// are present so it is safe to use as an MCP Output.
type ticketLinkResponse struct {
	ID          uint                     `json:"id" jsonschema:"the link record ID"`
	SourceID    uint                     `json:"source_id" jsonschema:"the source ticket ID"`
	TargetID    uint                     `json:"target_id" jsonschema:"the target ticket ID"`
	Type        string                   `json:"type" jsonschema:"link type: related, duplicate, or blocks"`
	OtherTicket ticketLinkOtherTicket    `json:"other_ticket" jsonschema:"summary of the other ticket in the link"`
}

type ticketLinkOtherTicket struct {
	ID     uint   `json:"id" jsonschema:"the other ticket's numeric ID"`
	Title  string `json:"title" jsonschema:"the other ticket's title"`
	Status string `json:"status" jsonschema:"the other ticket's status"`
}

func ticketLinkResponseFrom(r ticket.LinkResponse) ticketLinkResponse {
	return ticketLinkResponse{
		ID:       r.ID,
		SourceID: r.SourceID,
		TargetID: r.TargetID,
		Type:     r.Type,
		OtherTicket: ticketLinkOtherTicket{
			ID:     r.OtherTicket.ID,
			Title:  r.OtherTicket.Title,
			Status: r.OtherTicket.Status,
		},
	}
}

func ticketLinkResponsesFrom(rs []ticket.LinkResponse) []ticketLinkResponse {
	if len(rs) == 0 {
		return nil
	}
	out := make([]ticketLinkResponse, len(rs))
	for i, r := range rs {
		out[i] = ticketLinkResponseFrom(r)
	}
	return out
}

// ticketLinksOutput is the structured output for ticket_link_list.
type ticketLinksOutput struct {
	Links    []ticketLinkResponse `json:"links,omitempty" jsonschema:"the links associated with the ticket"`
	TicketID uint                 `json:"ticket_id" jsonschema:"the ticket these links belong to"`
	Total    int                  `json:"total" jsonschema:"total number of links"`
}

// ticketMergeOutput is the structured output for ticket_merge.
type ticketMergeOutput struct {
	SourceID uint   `json:"source_id" jsonschema:"the merged (now closed) ticket ID"`
	TargetID uint   `json:"target_id" jsonschema:"the ticket that absorbed the source"`
	Status   string `json:"status" jsonschema:"always 'merged'"`
}

// ticketLinkActionOutput is the structured output for ticket_link_delete.
type ticketLinkActionOutput struct {
	LinkID   uint   `json:"link_id" jsonschema:"the affected link ID"`
	TicketID uint   `json:"ticket_id" jsonschema:"the ticket the link belonged to"`
	Status   string `json:"status" jsonschema:"the result status"`
}

// ----------------------------------------------------------------------------
// Input types
// ----------------------------------------------------------------------------

type ticketMergeInput struct {
	SourceID uint `json:"source_id" jsonschema:"the ticket to merge and close (required)"`
	TargetID uint `json:"target_id" jsonschema:"the ticket to absorb the source (required)"`
}

type ticketLinkCreateInput struct {
	SourceID uint   `json:"source_id" jsonschema:"the source ticket ID (required)"`
	TargetID uint   `json:"target_id" jsonschema:"the target ticket ID (required)"`
	LinkType string `json:"link_type" jsonschema:"link type: related, duplicate, or blocks (required)"`
}

type ticketLinkListInput struct {
	TicketID uint `json:"ticket_id" jsonschema:"the ticket whose links to list (required)"`
}

type ticketLinkDeleteInput struct {
	TicketID uint `json:"ticket_id" jsonschema:"the ticket the link is associated with (required)"`
	LinkID   uint `json:"link_id" jsonschema:"the numeric ID of the link to delete (required)"`
}

// ----------------------------------------------------------------------------
// Registration
// ----------------------------------------------------------------------------

func registerTicketMergeTools(s *mcp.Server, b Backend) {
	registerTool(s, "ticket_merge",
		"Merge one ticket into another: reassigns all messages and attachments from the source to the target, then marks the source as merged (team members only).",
		"ticket:write",
		func(ctx context.Context, in ticketMergeInput) (ticketMergeOutput, string, error) {
			return ticketMerge(ctx, b, in)
		})

	registerTool(s, "ticket_link_create",
		"Create a directional link between two tickets. Link types: related, duplicate, blocks. Idempotent — returns the existing link if the same triple already exists.",
		"ticket:write",
		func(ctx context.Context, in ticketLinkCreateInput) (ticketLinkResponse, string, error) {
			return ticketLinkCreate(ctx, b, in)
		})

	registerTool(s, "ticket_link_list",
		"List all links (as source or target) for a given ticket, enriched with the other ticket's id, title, and status.",
		"ticket:read",
		func(ctx context.Context, in ticketLinkListInput) (ticketLinksOutput, string, error) {
			return ticketLinkList(ctx, b, in)
		})

	registerTool(s, "ticket_link_delete",
		"Delete a specific ticket link by link ID. The link must be associated with the given ticket_id (prevents arbitrary link deletion).",
		"ticket:write",
		func(ctx context.Context, in ticketLinkDeleteInput) (ticketLinkActionOutput, string, error) {
			return ticketLinkDelete(ctx, b, in)
		})
}

// ----------------------------------------------------------------------------
// Handlers
// ----------------------------------------------------------------------------

func ticketMerge(ctx context.Context, b Backend, in ticketMergeInput) (ticketMergeOutput, string, error) {
	actor := sessionActor(ctx)
	if err := b.MergeTickets(actor, in.SourceID, in.TargetID); err != nil {
		return ticketMergeOutput{}, "", err
	}
	out := ticketMergeOutput{SourceID: in.SourceID, TargetID: in.TargetID, Status: "merged"}
	return out, fmt.Sprintf("Merged ticket #%d into #%d.", in.SourceID, in.TargetID), nil
}

func ticketLinkCreate(ctx context.Context, b Backend, in ticketLinkCreateInput) (ticketLinkResponse, string, error) {
	actor := sessionActor(ctx)
	link, err := b.LinkTickets(actor, in.SourceID, in.TargetID, in.LinkType)
	if err != nil {
		return ticketLinkResponse{}, "", err
	}
	lr := ticketLinkResponse{
		ID:       link.ID,
		SourceID: link.SourceID,
		TargetID: link.TargetID,
		Type:     link.Type,
	}
	return lr, fmt.Sprintf("Linked ticket #%d → #%d (%s).", in.SourceID, in.TargetID, in.LinkType), nil
}

func ticketLinkList(ctx context.Context, b Backend, in ticketLinkListInput) (ticketLinksOutput, string, error) {
	actor := sessionActor(ctx)
	links, err := b.ListTicketLinks(actor, in.TicketID)
	if err != nil {
		return ticketLinksOutput{}, "", err
	}
	out := ticketLinksOutput{
		Links:    ticketLinkResponsesFrom(links),
		TicketID: in.TicketID,
		Total:    len(links),
	}
	return out, fmt.Sprintf("Listed %d link(s) for ticket #%d.", len(links), in.TicketID), nil
}

func ticketLinkDelete(ctx context.Context, b Backend, in ticketLinkDeleteInput) (ticketLinkActionOutput, string, error) {
	actor := sessionActor(ctx)
	if err := b.UnlinkTicket(actor, in.TicketID, in.LinkID); err != nil {
		return ticketLinkActionOutput{}, "", err
	}
	out := ticketLinkActionOutput{LinkID: in.LinkID, TicketID: in.TicketID, Status: "deleted"}
	return out, fmt.Sprintf("Deleted link #%d from ticket #%d.", in.LinkID, in.TicketID), nil
}
