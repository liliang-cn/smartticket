package ticket

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/company/smartticket/internal/email"
	"github.com/company/smartticket/internal/models"
)

// Mailer sends outbound ticket replies by email. Implemented by
// internal/email.Service and injected via SetMailer; a nil mailer is a no-op.
type Mailer interface {
	SendTicketReply(ctx context.Context, to, ticketNumber, ticketTitle string, ticketID uint, body, authorName string)
}

// SetMailer injects the outbound email sender used by the reply hook.
func (s *Service) SetMailer(m Mailer) { s.mailer = m }

// ticketNumberRe matches the "TK-123" token embedded in reply subjects.
var ticketNumberRe = regexp.MustCompile(`TK-\d+`)

// IngestEmail routes an inbound email (from the webhook) into the ticket system:
// if the subject references an existing ticket (and the sender is that ticket's
// requester), it is appended as a reply; otherwise a new ticket is opened.
// Implements email.TicketSink.
func (s *Service) IngestEmail(ctx context.Context, in email.InboundEmail) error {
	from := strings.ToLower(strings.TrimSpace(in.FromEmail))
	if from == "" {
		return fmt.Errorf("inbound email has no sender")
	}
	subject := strings.TrimSpace(in.Subject)
	body := strings.TrimSpace(in.Text)
	if body == "" {
		body = "(empty email)"
	}

	// Append to an existing ticket when the subject carries its number AND the
	// sender is the original requester (guards against spoofed appends).
	if num := ticketNumberRe.FindString(subject); num != "" {
		var tkt models.Ticket
		if err := s.db.Where("ticket_number = ? AND is_deleted = ?", num, false).First(&tkt).Error; err == nil {
			if strings.EqualFold(strings.TrimSpace(tkt.RequesterEmail), from) {
				uid := s.userIDByEmail(from)
				msg := &models.Message{TicketID: tkt.ID, UserID: uid, Content: body, ContentType: "text"}
				if err := s.db.Create(msg).Error; err != nil {
					return fmt.Errorf("failed to append inbound message: %w", err)
				}
				s.recordEvent(tkt.ID, uid, "replied", "replied via email")
				if s.notifier != nil && tkt.AssignedTo != nil && *tkt.AssignedTo != uid {
					s.notifier.Notify(ctx, []uint{*tkt.AssignedTo}, "ticket_reply",
						fmt.Sprintf("New email reply on ticket #%d", tkt.ID), snippet(body), "ticket", tkt.ID)
				}
				return nil
			}
			// Sender is not the requester — fall through to a new ticket.
		}
	}

	return s.createTicketFromEmail(from, in.FromName, subject, body)
}

// createTicketFromEmail opens a fresh ticket from an unmatched inbound email.
func (s *Service) createTicketFromEmail(fromEmail, fromName, subject, body string) error {
	ticketNumber, err := s.generateTicketNumber()
	if err != nil {
		return fmt.Errorf("failed to generate ticket number: %w", err)
	}
	const priority, severity = "medium", "minor"
	slaDue, err := s.slaCalculator.CalculateSLADueDates(priority, severity, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to calculate SLA: %w", err)
	}

	title := subject
	if title == "" {
		title = "(no subject)"
	}
	name := strings.TrimSpace(fromName)
	if name == "" {
		name = fromEmail
	}

	// Associate the ticket with the customer org of a matching user, if any.
	var customerID *uint
	if u := s.userByEmail(fromEmail); u != nil {
		customerID = u.CustomerID
	}

	tkt := &models.Ticket{
		BaseModel:      models.BaseModel{CreatedAt: time.Now(), UpdatedAt: time.Now()},
		TicketNumber:   ticketNumber,
		Title:          clip(title, 255),
		Description:    body,
		Status:         "open",
		Priority:       priority,
		Severity:       severity,
		Type:           "email",
		CustomerID:     customerID,
		RequesterName:  clip(name, 255),
		RequesterEmail: clip(fromEmail, 255),
		DueDate:        &slaDue.ResponseDueDate,
		SLAStatus:      "within",
	}
	if err := s.db.Create(tkt).Error; err != nil {
		return fmt.Errorf("failed to create ticket from email: %w", err)
	}
	s.recordEvent(tkt.ID, 0, "created", "created from inbound email")
	return nil
}

func (s *Service) userByEmail(emailAddr string) *models.User {
	var u models.User
	if err := s.db.Where("email = ?", strings.ToLower(emailAddr)).First(&u).Error; err != nil {
		return nil
	}
	return &u
}

func (s *Service) userIDByEmail(emailAddr string) uint {
	if u := s.userByEmail(emailAddr); u != nil {
		return u.ID
	}
	return 0
}

func snippet(s string) string {
	if len(s) > 140 {
		return s[:140]
	}
	return s
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
