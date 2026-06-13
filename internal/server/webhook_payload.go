package server

import (
	"encoding/json"
	"time"

	"github.com/company/smartticket/internal/automation"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// buildWebhookPayload constructs a JSON string describing the event. It loads
// the relevant ticket from the database for richer context; on any DB error it
// falls back to a minimal payload containing only the ticket ID.
func buildWebhookPayload(db *gorm.DB, ev automation.Event) string {
	out := map[string]any{"event": string(ev.Type), "occurred_at": time.Now().Unix()}
	var tkt models.Ticket
	if err := db.Where("id = ?", ev.TicketID).First(&tkt).Error; err == nil {
		out["data"] = map[string]any{
			"id":            tkt.ID,
			"ticket_number": tkt.TicketNumber,
			"title":         tkt.Title,
			"status":        tkt.Status,
			"priority":      tkt.Priority,
			"assigned_to":   tkt.AssignedTo,
		}
	} else {
		out["data"] = map[string]any{"ticket_id": ev.TicketID}
	}
	b, _ := json.Marshal(out)
	return string(b)
}
