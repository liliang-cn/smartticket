package ticket

import (
	"context"

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
