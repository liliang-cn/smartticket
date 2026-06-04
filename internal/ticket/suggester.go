package ticket

import (
	"context"
	"strings"

	"github.com/company/smartticket/internal/aiassist"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
)

// ReplySuggester drafts an AI reply for a ticket. Implemented by
// internal/aiassist.Assistant and injected via SetSuggester; nil = unavailable.
type ReplySuggester interface {
	SuggestReply(ctx context.Context, in aiassist.SuggestInput) (string, error)
}

// SetSuggester injects the AI reply assistant.
func (s *Service) SetSuggester(r ReplySuggester) { s.suggester = r }

// SuggestReply asks the AI assistant to draft the agent's next reply for a
// ticket. Team-only. Maps AI-unavailability to a clear client error.
func (s *Service) SuggestReply(actor authz.Actor, ticketID uint) (string, error) {
	if !actor.IsTeam() {
		return "", errors.NewForbiddenError("only team members can use AI suggestions")
	}
	if s.suggester == nil {
		return "", errors.NewConflictError("no AI provider configured")
	}

	tkt, err := s.findTicketForActor(actor, ticketID)
	if err != nil {
		return "", err
	}

	var msgs []models.Message
	s.db.Where("ticket_id = ?", ticketID).Order("created_at asc").Preload("User").Find(&msgs)

	conv := make([]aiassist.Turn, 0, len(msgs))
	for i := range msgs {
		m := msgs[i]
		author, isCustomer := "", false
		if m.User != nil {
			author = strings.TrimSpace(m.User.FirstName + " " + m.User.LastName)
			isCustomer = m.User.Role == "customer"
		}
		conv = append(conv, aiassist.Turn{Author: author, IsCustomer: isCustomer, Content: m.Content})
	}

	draft, err := s.suggester.SuggestReply(context.Background(), aiassist.SuggestInput{
		Title:        tkt.Title,
		Description:  tkt.Description,
		CustomerName: tkt.RequesterName,
		Conversation: conv,
	})
	if err != nil {
		switch {
		case aiassist.IsDisabled(err):
			return "", errors.NewConflictError("AI suggested replies are turned off")
		case aiassist.IsNotConfigured(err):
			return "", errors.NewConflictError("no AI provider configured")
		default:
			return "", err
		}
	}
	return draft, nil
}
