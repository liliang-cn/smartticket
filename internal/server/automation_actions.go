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

// automationEffector implements automation.Effector using the ticket,
// notification, and AI services. Placing the adapter in the server package
// (which already imports all of them) avoids any import cycle.
//
// Import diagram (no cycle):
//
//	server → ticket, notification, aiassist  (all already imported by server)
//	internal/automation defines Effector — does NOT import ticket/notification
type automationEffector struct {
	svc       *ticket.Service
	notif     *notification.Service
	assistant *aiassist.Assistant // optional; may be nil
	db        *gorm.DB
}

func (e *automationEffector) Assign(ticketID uint, userID, teamID *uint) error {
	return e.svc.AssignAutomation(ticketID, userID, teamID)
}

func (e *automationEffector) AddTag(ticketID uint, tag string) error {
	return e.svc.AddTagAutomation(ticketID, tag)
}

func (e *automationEffector) SetField(ticketID uint, field, value string) error {
	return e.svc.SetFieldAutomation(ticketID, field, value)
}

// Notify creates an in-app notification for the ticket's assigned agent (best-effort).
func (e *automationEffector) Notify(ticketID uint, message string) error {
	var tkt models.Ticket
	if err := e.db.Where("id = ? AND is_deleted = ?", ticketID, false).First(&tkt).Error; err != nil {
		logger.Warn("automation effector: notify: ticket not found",
			zap.Uint("ticket_id", ticketID), zap.Error(err))
		return nil
	}
	if tkt.AssignedTo == nil {
		return nil // no assignee — silently skip
	}
	e.notif.Notify(context.Background(), []uint{*tkt.AssignedTo},
		"automation", fmt.Sprintf("Automation: %s", message), message, "ticket", ticketID)
	return nil
}

// SendEmail sends an email to the ticket's requester.
// This is a best-effort no-op placeholder; in a future iteration the email
// service can be injected directly here.
func (e *automationEffector) SendEmail(ticketID uint, subject, body string) error {
	logger.Info("automation: send_email action (no-op — wire email service to enable)",
		zap.Uint("ticket_id", ticketID),
		zap.String("subject", subject),
	)
	return nil
}

func (e *automationEffector) Escalate(ticketID uint) error {
	return e.svc.EscalateAutomation(ticketID)
}

func (e *automationEffector) AISuggest(ticketID uint) error {
	if e.assistant == nil {
		return nil
	}
	// Fire-and-forget: run AI suggestion in background so the event handler returns quickly.
	go func() {
		in, _, err := e.svc.LoadAISuggestInput(ticketID)
		if err != nil {
			logger.Warn("automation: ai_suggest: LoadAISuggestInput failed",
				zap.Uint("ticket_id", ticketID), zap.Error(err))
			return
		}
		_, _ = e.assistant.SuggestReplyStructured(context.Background(), in)
	}()
	return nil
}

func (e *automationEffector) AIAutoReply(ticketID uint) error {
	if e.assistant == nil {
		return nil
	}
	go func() {
		in, _, err := e.svc.LoadAISuggestInput(ticketID)
		if err != nil {
			logger.Warn("automation: ai_auto_reply: LoadAISuggestInput failed",
				zap.Uint("ticket_id", ticketID), zap.Error(err))
			return
		}
		draft, err := e.assistant.SuggestReplyStructured(context.Background(), in)
		if err != nil {
			logger.Warn("automation: ai_auto_reply: SuggestReplyStructured failed",
				zap.Uint("ticket_id", ticketID), zap.Error(err))
			return
		}
		if err := e.svc.PostAIMessage(ticketID, draft.Reply); err != nil {
			logger.Warn("automation: ai_auto_reply: PostAIMessage failed",
				zap.Uint("ticket_id", ticketID), zap.Error(err))
		}
	}()
	return nil
}

func (e *automationEffector) Close(ticketID uint) error {
	return e.svc.SetFieldAutomation(ticketID, "status", "closed")
}
