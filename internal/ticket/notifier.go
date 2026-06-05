package ticket

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/company/smartticket/internal/logger"
	"github.com/company/smartticket/internal/models"
)

// Notifier emits in-app notifications. Implemented by internal/notification.Service.
// It is injected via Service.SetNotifier so the ticket package never imports the
// notification package, and is always nil-safe (a nil notifier is a no-op).
type Notifier interface {
	Notify(ctx context.Context, userIDs []uint, ntype, title, body, refType string, refID uint)
}

// SetNotifier injects the in-app notifier used by the ticket-event hooks. Passing
// nil (or never calling this) disables notifications without affecting ticket ops.
func (s *Service) SetNotifier(n Notifier) { s.notifier = n }

// notifyMentions parses @handle tokens from body, resolves each handle to a
// user (case-insensitive match on username), and emits a mention notification
// to every found user. The author (authorID) is never notified of their own
// mention. This is best-effort: failures are logged, never propagated.
// Only called for internal messages (callers are responsible for the gate).
func (s *Service) notifyMentions(tkt *models.Ticket, body string, authorID uint) {
	if s.notifier == nil {
		return
	}
	handles := parseMentions(body)
	if len(handles) == 0 {
		return
	}

	// Resolve handles to user IDs in a single DB query (case-insensitive).
	// SQLite LOWER() is available; GORM uses the column name; we normalise to
	// lowercase for comparison.
	var users []models.User
	if err := s.db.
		Where("LOWER(username) IN ?", handles).
		Find(&users).Error; err != nil {
		logger.Warn("mention: failed to resolve handles", zap.Error(err))
		return
	}

	// Resolve the author's display name for the notification body (best-effort).
	authorDisplay := fmt.Sprintf("user #%d", authorID)
	if authorID != 0 {
		var author models.User
		if err := s.db.First(&author, authorID).Error; err == nil {
			authorDisplay = displayName(&author)
		}
	}

	// Build the recipient list, excluding the author.
	recipients := make([]uint, 0, len(users))
	for _, u := range users {
		if u.ID == authorID {
			continue
		}
		recipients = append(recipients, u.ID)
	}
	if len(recipients) == 0 {
		return
	}

	title := fmt.Sprintf("You were mentioned on ticket #%d", tkt.ID)
	body2 := fmt.Sprintf("%s mentioned you on ticket #%d: %s",
		authorDisplay, tkt.ID, truncate(strings.TrimSpace(body), 140))

	s.notifier.Notify(context.Background(), recipients, "ticket_mention", title, body2, "ticket", tkt.ID)
}

// truncate shortens s to at most n runes, appending "…" when truncated.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// customerRecipients returns the active user IDs for a customer organization,
// excluding the given user (typically the event's author so they are not
// notified of their own action).
func (s *Service) customerRecipients(customerID, exclude uint) []uint {
	var ids []uint
	if err := s.db.Model(&models.User{}).
		Where("customer_id = ? AND is_active = ?", customerID, true).
		Pluck("id", &ids).Error; err != nil {
		return nil
	}
	if exclude == 0 {
		return ids
	}
	out := ids[:0]
	for _, id := range ids {
		if id != exclude {
			out = append(out, id)
		}
	}
	return out
}
