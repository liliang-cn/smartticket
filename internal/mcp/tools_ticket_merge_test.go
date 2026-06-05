package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/ticket"
)

func TestTicketMergeTool(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("ticket:write"))

	// MergeTickets ignores the actor in the mock (matches on source/target only).
	b.On("MergeTickets", uint(10), uint(20)).Return(nil)

	out, summary, err := ticketMerge(ctx, b, ticketMergeInput{SourceID: 10, TargetID: 20})
	require.NoError(t, err)
	assert.Equal(t, uint(10), out.SourceID)
	assert.Equal(t, uint(20), out.TargetID)
	assert.Equal(t, "merged", out.Status)
	assert.Contains(t, summary, "#10")
	assert.Contains(t, summary, "#20")
	b.AssertExpectations(t)
}

func TestTicketLinkCreateTool(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("ticket:write"))

	b.On("LinkTickets", uint(10), uint(20), "related").Return(
		&models.TicketLink{BaseModel: models.BaseModel{ID: 7}, SourceID: 10, TargetID: 20, Type: "related"},
		nil,
	)

	out, summary, err := ticketLinkCreate(ctx, b, ticketLinkCreateInput{
		SourceID: 10, TargetID: 20, LinkType: "related",
	})
	require.NoError(t, err)
	assert.Equal(t, uint(7), out.ID)
	assert.Equal(t, "related", out.Type)
	assert.Contains(t, summary, "#10")
	b.AssertExpectations(t)
}

func TestTicketLinkListTool(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("ticket:read"))

	lr := ticket.LinkResponse{ID: 7, SourceID: 10, TargetID: 20, Type: "related"}
	lr.OtherTicket.ID = 20
	lr.OtherTicket.Title = "Related issue"
	lr.OtherTicket.Status = "open"

	b.On("ListTicketLinks", uint(10)).Return([]ticket.LinkResponse{lr}, nil)

	out, summary, err := ticketLinkList(ctx, b, ticketLinkListInput{TicketID: 10})
	require.NoError(t, err)
	assert.Equal(t, uint(10), out.TicketID)
	assert.Equal(t, 1, out.Total)
	require.Len(t, out.Links, 1)
	assert.Equal(t, "related", out.Links[0].Type)
	assert.Equal(t, uint(20), out.Links[0].OtherTicket.ID)
	assert.Contains(t, summary, "#10")
	b.AssertExpectations(t)
}

func TestTicketLinkDeleteTool(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("ticket:write"))

	b.On("UnlinkTicket", uint(10), uint(7)).Return(nil)

	out, summary, err := ticketLinkDelete(ctx, b, ticketLinkDeleteInput{TicketID: 10, LinkID: 7})
	require.NoError(t, err)
	assert.Equal(t, uint(7), out.LinkID)
	assert.Equal(t, uint(10), out.TicketID)
	assert.Equal(t, "deleted", out.Status)
	assert.Contains(t, summary, "#7")
	b.AssertExpectations(t)
}

func TestTicketMergePermissionDenied(t *testing.T) {
	// ticket:write is required for merge and link create/delete.
	ctx := ctxWithSession(newTestSession("ticket:read")) // read only
	err := RequirePermission(ctx, "ticket:write")
	var permErr *PermissionError
	require.ErrorAs(t, err, &permErr)
	assert.Equal(t, "ticket:write", permErr.Code)
}
