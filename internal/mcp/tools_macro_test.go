package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/company/smartticket/internal/macro"
	"github.com/company/smartticket/internal/models"
)

func TestMacroCreateAndList(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("macro:read", "macro:write"))
	shared := true

	b.On("CreateMacro", uint(1), macro.CreateRequest{
		Title:  "Greeting",
		Body:   "Hello {{customer.name}}!",
		Shared: &shared,
	}).Return(&models.Macro{
		BaseModel: models.BaseModel{ID: 5},
		Title:     "Greeting",
		Body:      "Hello {{customer.name}}!",
		Shared:    true,
		OwnerID:   1,
	}, nil)

	out, summary, err := macroCreate(ctx, b, macroCreateInput{
		Title:  "Greeting",
		Body:   "Hello {{customer.name}}!",
		Shared: &shared,
	})
	require.NoError(t, err)
	assert.Equal(t, uint(5), out.ID)
	assert.Equal(t, "Greeting", out.Title)
	assert.Contains(t, summary, "#5")

	b.On("ListMacros", uint(1)).Return([]models.Macro{
		{BaseModel: models.BaseModel{ID: 5}, Title: "Greeting", Shared: true},
		{BaseModel: models.BaseModel{ID: 6}, Title: "Closing", Shared: false, OwnerID: 1},
	}, nil)

	lout, _, err := macroList(ctx, b)
	require.NoError(t, err)
	assert.Equal(t, 2, lout.Total)
	require.Len(t, lout.Macros, 2)
	b.AssertExpectations(t)
}

func TestMacroGetTool(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("macro:read"))

	b.On("GetMacro", uint(1), uint(5)).Return(&models.Macro{
		BaseModel: models.BaseModel{ID: 5},
		Title:     "Greeting",
		Body:      "Hello!",
	}, nil)

	out, summary, err := macroGet(ctx, b, macroGetInput{ID: 5})
	require.NoError(t, err)
	assert.Equal(t, uint(5), out.ID)
	assert.Equal(t, "Greeting", out.Title)
	assert.Contains(t, summary, "#5")
	b.AssertExpectations(t)
}

func TestMacroApplyTool(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("macro:write"))

	rctx := macro.RenderContext{
		CustomerName:  "Acme",
		AgentName:     "Alice",
		TicketID:      "42",
		TicketSubject: "Login issue",
	}
	b.On("ApplyMacro", uint(5), uint(1), rctx).Return(
		"Hello Acme! (Alice)",
		[]macro.Action{{Type: "set_status", Params: map[string]string{"status": "resolved"}}},
		nil,
	)

	out, summary, err := macroApply(ctx, b, macroApplyInput{
		MacroID:       5,
		TicketID:      42,
		TicketSubject: "Login issue",
		CustomerName:  "Acme",
		AgentName:     "Alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello Acme! (Alice)", out.Rendered)
	require.Len(t, out.Actions, 1)
	assert.Equal(t, "set_status", out.Actions[0].Type)
	assert.Contains(t, summary, "#5")
	b.AssertExpectations(t)
}

func TestMacroDeleteTool(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("macro:write"))

	b.On("DeleteMacro", uint(1), uint(7)).Return(nil)

	out, summary, err := macroDelete(ctx, b, macroDeleteInput{ID: 7})
	require.NoError(t, err)
	assert.Equal(t, uint(7), out.ID)
	assert.True(t, out.Deleted)
	assert.Contains(t, summary, "#7")
	b.AssertExpectations(t)
}

func TestMacroRequiresSession(t *testing.T) {
	// Handlers that call SessionFromContext should return ErrUnauthenticated when
	// the context carries no session (nil session stored by ctxWithSession(nil)).
	b := new(MockBackend)
	_, _, err := macroList(ctxWithSession(nil), b)
	require.ErrorIs(t, err, ErrUnauthenticated)
}
