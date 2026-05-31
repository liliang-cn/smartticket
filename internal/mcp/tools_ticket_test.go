package mcp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/company/smartticket/internal/ticket"
)

func TestTicketCreate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:write"))

	in := ticketCreateInput{
		Title:          "Cannot login",
		Description:    "User is unable to log in",
		Priority:       "high",
		RequesterName:  "Alice",
		RequesterEmail: "alice@example.com",
	}

	mb.On("CreateTicket", uint(1), mock.MatchedBy(func(req *ticket.CreateTicketRequest) bool {
		return req.Title == "Cannot login" &&
			req.Description == "User is unable to log in" &&
			req.Priority == "high" &&
			req.RequesterName == "Alice" &&
			req.RequesterEmail == "alice@example.com"
	})).Return(&ticket.TicketResponse{ID: 42, TicketNumber: "TK-42"}, nil)

	out, summary, err := ticketCreate(ctx, mb, in)
	assert.NoError(t, err)
	assert.Equal(t, uint(42), out.ID)
	assert.Equal(t, "TK-42", out.TicketNumber)
	assert.Equal(t, "created ticket #42 (TK-42)", summary)
	mb.AssertExpectations(t)
}

func TestTicketCreateUnauthenticated(t *testing.T) {
	mb := &MockBackend{}
	// No session in context.
	_, _, err := ticketCreate(t.Context(), mb, ticketCreateInput{Title: "x"})
	assert.ErrorIs(t, err, ErrUnauthenticated)
	mb.AssertNotCalled(t, "CreateTicket", mock.Anything, mock.Anything)
}

func TestTicketCreateError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:write"))

	wantErr := errors.New("boom")
	mb.On("CreateTicket", uint(1), mock.Anything).Return(nil, wantErr)

	_, _, err := ticketCreate(ctx, mb, ticketCreateInput{Title: "x"})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestTicketGet(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:read"))

	mb.On("GetTicket", uint(7)).Return(&ticket.TicketResponse{ID: 7, TicketNumber: "TK-7"}, nil)

	out, summary, err := ticketGet(ctx, mb, ticketGetInput{ID: 7})
	assert.NoError(t, err)
	assert.Equal(t, uint(7), out.ID)
	assert.Equal(t, "fetched ticket #7 (TK-7)", summary)
	mb.AssertExpectations(t)
}

func TestTicketList(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:read"))

	in := ticketListInput{
		Page:       2,
		PageSize:   10,
		Status:     "open",
		Priority:   "high",
		Category:   "bug",
		AssignedTo: 5,
		Search:     "login",
	}

	wantFilters := map[string]interface{}{
		"status":      "open",
		"priority":    "high",
		"category":    "bug",
		"assigned_to": uint(5),
		"search":      "login",
	}

	mb.On("ListTickets", 2, 10, wantFilters).Return(&ticket.TicketListResponse{
		Data:  []ticket.TicketResponse{{ID: 1}, {ID: 2}},
		Total: 2,
		Page:  2,
	}, nil)

	out, summary, err := ticketList(ctx, mb, in)
	assert.NoError(t, err)
	assert.Len(t, out.Data, 2)
	assert.Equal(t, "listed 2 of 2 ticket(s) (page 2)", summary)
	mb.AssertExpectations(t)
}

func TestTicketListEmptyFilters(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:read"))

	mb.On("ListTickets", 0, 0, map[string]interface{}{}).Return(&ticket.TicketListResponse{
		Data:  []ticket.TicketResponse{},
		Total: 0,
		Page:  1,
	}, nil)

	_, summary, err := ticketList(ctx, mb, ticketListInput{})
	assert.NoError(t, err)
	assert.Equal(t, "listed 0 of 0 ticket(s) (page 1)", summary)
	mb.AssertExpectations(t)
}

func TestTicketUpdate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:write"))

	in := ticketUpdateInput{
		ID:     9,
		Status: "resolved",
		Title:  "Updated title",
	}

	mb.On("UpdateTicket", uint(9), uint(1), mock.MatchedBy(func(req *ticket.UpdateTicketRequest) bool {
		return req.Status == "resolved" && req.Title == "Updated title"
	})).Return(&ticket.TicketResponse{ID: 9, TicketNumber: "TK-9"}, nil)

	out, summary, err := ticketUpdate(ctx, mb, in)
	assert.NoError(t, err)
	assert.Equal(t, uint(9), out.ID)
	assert.Equal(t, "updated ticket #9 (TK-9)", summary)
	mb.AssertExpectations(t)
}

func TestTicketUpdateUnauthenticated(t *testing.T) {
	mb := &MockBackend{}
	_, _, err := ticketUpdate(t.Context(), mb, ticketUpdateInput{ID: 1})
	assert.ErrorIs(t, err, ErrUnauthenticated)
	mb.AssertNotCalled(t, "UpdateTicket", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketDelete(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:write"))

	mb.On("DeleteTicket", uint(3)).Return(nil)

	out, summary, err := ticketDelete(ctx, mb, ticketDeleteInput{ID: 3})
	assert.NoError(t, err)
	assert.Equal(t, uint(3), out.ID)
	assert.True(t, out.Deleted)
	assert.Equal(t, "deleted ticket #3", summary)
	mb.AssertExpectations(t)
}

func TestTicketDeleteError(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:write"))

	wantErr := errors.New("not found")
	mb.On("DeleteTicket", uint(3)).Return(wantErr)

	_, _, err := ticketDelete(ctx, mb, ticketDeleteInput{ID: 3})
	assert.ErrorIs(t, err, wantErr)
	mb.AssertExpectations(t)
}

func TestTicketAssign(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:write"))

	mb.On("AssignTicket", uint(4), uint(6)).Return(nil)

	out, summary, err := ticketAssign(ctx, mb, ticketAssignInput{ID: 4, AssignedTo: 6})
	assert.NoError(t, err)
	assert.Equal(t, uint(4), out.ID)
	assert.Equal(t, uint(6), out.AssignedTo)
	assert.Equal(t, "assigned ticket #4 to user #6", summary)
	mb.AssertExpectations(t)
}

func TestTicketStats(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:read"))

	stats := map[string]interface{}{"total_tickets": int64(12), "open_tickets": int64(3)}
	mb.On("GetTicketStats").Return(stats, nil)

	out, summary, err := ticketStats(ctx, mb, ticketStatsInput{})
	assert.NoError(t, err)
	assert.Equal(t, stats, out.Stats)
	assert.Equal(t, "fetched ticket statistics", summary)
	mb.AssertExpectations(t)
}

func TestTicketMessageCreate(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:write"))

	in := ticketMessageCreateInput{
		TicketID:    42,
		Content:     "Working on it now",
		ContentType: "text",
		IsInternal:  true,
	}

	mb.On("CreateMessage", uint(42), uint(1), mock.MatchedBy(func(req *ticket.CreateMessageRequest) bool {
		return req.Content == "Working on it now" &&
			req.ContentType == "text" &&
			req.IsInternal == true
	})).Return(&ticket.MessageResponse{ID: 7, TicketID: 42, UserID: 1, Content: "Working on it now", ContentType: "text", IsInternal: true}, nil)

	out, summary, err := ticketMessageCreate(ctx, mb, in)
	assert.NoError(t, err)
	assert.Equal(t, uint(7), out.ID)
	assert.Equal(t, uint(42), out.TicketID)
	assert.Equal(t, uint(1), out.UserID)
	assert.True(t, out.IsInternal)
	assert.Equal(t, "posted reply #7 on ticket #42", summary)
	mb.AssertExpectations(t)
}

func TestTicketMessageCreateUnauthenticated(t *testing.T) {
	mb := &MockBackend{}
	_, _, err := ticketMessageCreate(t.Context(), mb, ticketMessageCreateInput{TicketID: 1, Content: "x"})
	assert.ErrorIs(t, err, ErrUnauthenticated)
	mb.AssertNotCalled(t, "CreateMessage", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketMessageList(t *testing.T) {
	mb := &MockBackend{}
	ctx := ctxWithSession(newTestSession("ticket:read"))

	mb.On("ListMessages", uint(42)).Return([]ticket.MessageResponse{
		{ID: 1, TicketID: 42, UserID: 1, Content: "first"},
		{ID: 2, TicketID: 42, UserID: 2, Content: "second"},
	}, nil)

	out, summary, err := ticketMessageList(ctx, mb, ticketMessageListInput{TicketID: 42})
	assert.NoError(t, err)
	assert.Equal(t, 2, out.Total)
	assert.Len(t, out.Data, 2)
	assert.Equal(t, uint(1), out.Data[0].ID)
	assert.Equal(t, "second", out.Data[1].Content)
	assert.Equal(t, "listed 2 message(s) on ticket #42", summary)
	mb.AssertExpectations(t)
}
