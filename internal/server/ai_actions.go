package server

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/aiassist"
	"github.com/company/smartticket/internal/logger"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/notification"
	"github.com/company/smartticket/internal/ticket"
)

// ticketAIActions implements aiassist.TicketActions by delegating to the
// ticket and notification services.  Placing this adapter in the server
// package (which already imports both ticket and aiassist) avoids a new
// intermediate package and prevents import cycles.
//
// Import diagram (no cycle):
//   server → ticket → aiassist   (ticket/suggester.go already imports aiassist)
//   server → aiassist            (server directly imports aiassist)
//   aiassist does NOT import ticket — it only holds the TicketActions interface
type ticketAIActions struct {
	svc  *ticket.Service
	notif *notification.Service
	db   *gorm.DB
}

// PostAIReply appends an AI-authored public reply.  The ticket service
// publishes EventMessageCreated with Source:"ai", which the AutoResolver's
// loop guard catches so the reply is never re-processed.
func (a *ticketAIActions) PostAIReply(ticketID uint, body string) error {
	return a.svc.PostAIMessage(ticketID, body)
}

// CountAIReplies returns the number of AI-authored messages on the ticket.
func (a *ticketAIActions) CountAIReplies(ticketID uint) (int, error) {
	return a.svc.CountAIMessages(ticketID)
}

// UpdateClassification updates the ticket's category, priority, and tags.
func (a *ticketAIActions) UpdateClassification(ticketID uint, category, priority string, tags []string) error {
	return a.svc.UpdateTicketClassification(ticketID, category, priority, tags)
}

// SetSummary stores a generated summary on the ticket.
func (a *ticketAIActions) SetSummary(ticketID uint, summary string) error {
	return a.svc.SetTicketSummary(ticketID, summary)
}

// LoadContext assembles the AI input context for a ticket and reports whether
// the last message is from a customer (i.e. someone is waiting for a reply).
func (a *ticketAIActions) LoadContext(ticketID uint) (aiassist.SuggestInput, bool, error) {
	return a.svc.LoadAISuggestInput(ticketID)
}

// NotifyAgentsSuggestion creates an in-app notification for the assigned agent
// summarising the AI suggestion.  If the ticket has no assigned agent, the
// notification is silently skipped (logged at debug level).
func (a *ticketAIActions) NotifyAgentsSuggestion(ticketID uint, draft aiassist.Draft) error {
	// Load the ticket to find the assigned agent.
	var tkt models.Ticket
	if err := a.db.Where("id = ? AND is_deleted = ?", ticketID, false).First(&tkt).Error; err != nil {
		logger.Warn("ai_actions: failed to load ticket for notification",
			zap.Uint("ticket_id", ticketID), zap.Error(err))
		return nil // best-effort; never break the caller
	}

	if tkt.AssignedTo == nil || *tkt.AssignedTo == 0 {
		logger.Debug("ai_actions: ticket has no assigned agent; skipping suggestion notification",
			zap.Uint("ticket_id", ticketID))
		return nil
	}

	confidence := int(draft.Confidence * 100)
	title := fmt.Sprintf("AI suggestion for ticket #%d (confidence %d%%)", ticketID, confidence)
	body := draft.Reply
	if len(body) > 200 {
		body = body[:200] + "…"
	}
	if draft.NeedsClarification {
		title = fmt.Sprintf("AI needs clarification on ticket #%d", ticketID)
	}

	a.notif.Notify(context.Background(), []uint{*tkt.AssignedTo},
		"ai_suggestion", title, body, "ticket", ticketID)
	return nil
}
