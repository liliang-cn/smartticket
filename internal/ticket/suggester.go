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
	SuggestReplyStructured(ctx context.Context, in aiassist.SuggestInput) (aiassist.Draft, error)
}

// SetSuggester injects the AI reply assistant.
func (s *Service) SetSuggester(r ReplySuggester) { s.suggester = r }

// buildConversation loads ticket messages from the DB and converts them to the
// aiassist.Turn slice expected by the AI layer.
func (s *Service) buildConversation(ticketID uint) ([]aiassist.Turn, error) {
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
	return conv, nil
}

// mapAIError translates sentinel AI errors to app-layer conflict errors.
func mapAIError(err error) error {
	switch {
	case aiassist.IsDisabled(err):
		return errors.NewConflictError("AI suggested replies are turned off")
	case aiassist.IsNotConfigured(err):
		return errors.NewConflictError("no AI provider configured")
	default:
		return err
	}
}

// SuggestReplyDraft asks the AI assistant to produce a structured Draft for a
// ticket's next reply. Team-only.
func (s *Service) SuggestReplyDraft(actor authz.Actor, ticketID uint) (aiassist.Draft, error) {
	if !actor.IsTeam() {
		return aiassist.Draft{}, errors.NewForbiddenError("only team members can use AI suggestions")
	}
	if s.suggester == nil {
		return aiassist.Draft{}, errors.NewConflictError("no AI provider configured")
	}

	tkt, err := s.findTicketForActor(actor, ticketID)
	if err != nil {
		return aiassist.Draft{}, err
	}

	conv, _ := s.buildConversation(ticketID)

	draft, err := s.suggester.SuggestReplyStructured(context.Background(), aiassist.SuggestInput{
		Title:        tkt.Title,
		Description:  tkt.Description,
		CustomerName: tkt.RequesterName,
		Conversation: conv,
	})
	if err != nil {
		return aiassist.Draft{}, mapAIError(err)
	}
	return draft, nil
}

// SuggestReply asks the AI assistant to draft the agent's next reply for a
// ticket. Team-only. Maps AI-unavailability to a clear client error.
// Delegates to SuggestReplyDraft and returns the reply string.
func (s *Service) SuggestReply(actor authz.Actor, ticketID uint) (string, error) {
	draft, err := s.SuggestReplyDraft(actor, ticketID)
	if err != nil {
		return "", err
	}
	return draft.Reply, nil
}
