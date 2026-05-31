package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/company/smartticket/internal/branding"
	"github.com/company/smartticket/internal/llm"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/subscription"
	"github.com/company/smartticket/internal/ticket"
)

// --- Subscription ---

func TestSubscriptionCreateAndList(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("subscription:write", "subscription:read"))

	b.On("CreateSubscription", mock.MatchedBy(func(r *subscription.CreateSubscriptionRequest) bool {
		return r.CustomerID == 2 && r.ProductID == 3 && r.NodeCount == 5
	})).Return(&subscription.SubscriptionResponse{ID: 9, CustomerID: 2, ProductID: 3, Status: "active"}, nil)

	out, summary, err := subscriptionCreate(ctx, b, subscriptionCreateInput{CustomerID: 2, ProductID: 3, NodeCount: 5})
	require.NoError(t, err)
	assert.Equal(t, uint(9), out.ID)
	assert.Contains(t, summary, "#9")

	b.On("ListSubscriptions", mock.Anything).
		Return([]subscription.SubscriptionResponse{{ID: 9, Status: "active"}}, 1, nil)
	lout, _, err := subscriptionList(ctx, b, subscriptionListInput{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), lout.Total)
	require.Len(t, lout.Subscriptions, 1)
	b.AssertExpectations(t)
}

// --- Notification (acts on the session user) ---

func TestNotificationListAndCount(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession()) // UserID 1, no perms needed

	b.On("ListNotifications", uint(1), false, 1, 20).
		Return([]models.Notification{{BaseModel: models.BaseModel{ID: 1}, Title: "Reply", Type: "ticket_reply"}}, 1, nil)
	out, _, err := notificationList(ctx, b, notificationListInput{})
	require.NoError(t, err)
	require.Len(t, out.Notifications, 1)
	assert.Equal(t, "Reply", out.Notifications[0].Title)

	b.On("UnreadNotificationCount", uint(1)).Return(3, nil)
	cout, summary, err := notificationUnreadCount(ctx, b)
	require.NoError(t, err)
	assert.Equal(t, int64(3), cout.Unread)
	assert.Contains(t, summary, "3")
	b.AssertExpectations(t)
}

func TestNotificationRequiresSession(t *testing.T) {
	b := new(MockBackend)
	_, _, err := notificationUnreadCount(context.Background(), b)
	require.ErrorIs(t, err, ErrUnauthenticated)
}

// --- LLM ---

func TestLLMListCreateTest(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("llm:read", "llm:write"))

	b.On("ListLLMProviders").Return([]models.LLMProvider{
		{BaseModel: models.BaseModel{ID: 1}, Name: "OpenAI", ProviderType: "openai", TaskTypes: `["chat"]`, APIKey: "ENC"},
	}, nil)
	lout, _, err := llmList(ctx, b)
	require.NoError(t, err)
	require.Len(t, lout.Providers, 1)
	assert.True(t, lout.Providers[0].HasAPIKey)
	assert.Equal(t, []string{"chat"}, lout.Providers[0].TaskTypes)

	b.On("CreateLLMProvider", mock.MatchedBy(func(in llm.CreateProviderInput) bool {
		return in.Name == "DeepSeek" && len(in.TaskTypes) == 1 && in.TaskTypes[0] == "chat"
	})).Return(&models.LLMProvider{BaseModel: models.BaseModel{ID: 2}, Name: "DeepSeek"}, nil)
	cout, _, err := llmCreate(ctx, b, llmProviderInput{Name: "DeepSeek", ProviderType: "deepseek", APIEndpoint: "https://x", Model: "v3", TaskTypes: []string{"chat"}})
	require.NoError(t, err)
	assert.Equal(t, uint(2), cout.ID)

	b.On("TestLLMProvider", mock.Anything, uint(2)).Return(llm.TestResult{ChatOK: true, LatencyMS: 12}, nil)
	tout, _, err := llmTest(ctx, b, llmIDInput{ID: 2})
	require.NoError(t, err)
	assert.True(t, tout.ChatOK)
	b.AssertExpectations(t)
}

// --- Branding ---

func TestBrandingGetAndUpdate(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("settings:write"))

	b.On("GetBranding").Return(&models.Branding{AppName: "SmartTicket", PrimaryColor: "#f59e0b"}, nil)
	gout, _, err := brandingGet(ctx, b)
	require.NoError(t, err)
	assert.Equal(t, "SmartTicket", gout.AppName)
	assert.False(t, gout.HasLogo)

	name := "Acme Desk"
	b.On("UpdateBranding", mock.MatchedBy(func(r *branding.UpdateRequest) bool {
		return r.AppName != nil && *r.AppName == "Acme Desk"
	})).Return(&models.Branding{AppName: "Acme Desk", PrimaryColor: "#3b82f6"}, nil)
	uout, _, err := brandingUpdate(ctx, b, brandingUpdateInput{AppName: &name})
	require.NoError(t, err)
	assert.Equal(t, "Acme Desk", uout.AppName)
	b.AssertExpectations(t)
}

// --- Attachment ---

func TestAttachmentList(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("ticket:read"))

	b.On("ListAttachments", uint(5)).Return([]models.Attachment{
		{BaseModel: models.BaseModel{ID: 1}, TicketID: 5, OriginalName: "log.txt", FileSize: 42},
	}, nil)
	out, summary, err := attachmentList(ctx, b, attachmentListInput{TicketID: 5})
	require.NoError(t, err)
	require.Len(t, out.Attachments, 1)
	assert.Equal(t, "log.txt", out.Attachments[0].OriginalName)
	assert.Contains(t, summary, "#5")
	b.AssertExpectations(t)
}

func TestTicketSLAAndEvents(t *testing.T) {
	b := new(MockBackend)
	ctx := ctxWithSession(newTestSession("ticket:read"))

	b.On("GetTicketSLA", uint(4)).Return(&ticket.TicketSLAResponse{
		TicketID: 4, Priority: "high", Severity: "minor", Source: "default",
		PolicyName: "Default policy (by priority)", ResponseMinutes: 30, ResolutionMinutes: 120,
		SLAStatus: "within",
	}, nil)
	sla, _, err := ticketSLA(ctx, b, ticketGetInput{ID: 4})
	require.NoError(t, err)
	assert.Equal(t, "Default policy (by priority)", sla.PolicyName)
	assert.Equal(t, 30, sla.ResponseMinutes)

	b.On("ListTicketEvents", uint(4)).Return([]ticket.TicketEventResponse{
		{ID: 1, Action: "status", Summary: "changed status: open → in_progress", ActorName: "Admin"},
	}, nil)
	ev, _, err := ticketEvents(ctx, b, ticketGetInput{ID: 4})
	require.NoError(t, err)
	require.Len(t, ev.Events, 1)
	assert.Equal(t, "status", ev.Events[0].Action)
	b.AssertExpectations(t)
}
