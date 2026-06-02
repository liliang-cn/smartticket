package ticket

import (
	"context"
	"testing"

	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/email"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestEmail_CreatesNewTicket(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))

		err := svc.IngestEmail(context.Background(), email.InboundEmail{
			FromName:  "Jane Roe",
			FromEmail: "jane@acme.com",
			Subject:   "Cannot log in",
			Text:      "It says invalid password.",
		})
		require.NoError(t, err)

		var tkt models.Ticket
		require.NoError(t, db.DB.Where("requester_email = ?", "jane@acme.com").First(&tkt).Error)
		assert.Equal(t, "Cannot log in", tkt.Title)
		assert.Equal(t, "open", tkt.Status)
		assert.Equal(t, "email", tkt.Type)
		assert.Contains(t, tkt.Description, "invalid password")
	})
}

func TestIngestEmail_AppendsWhenRequesterMatches(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))
		require.NoError(t, svc.IngestEmail(context.Background(), email.InboundEmail{
			FromEmail: "jane@acme.com", Subject: "Help", Text: "first",
		}))

		var tkt models.Ticket
		require.NoError(t, db.DB.First(&tkt).Error)

		err := svc.IngestEmail(context.Background(), email.InboundEmail{
			FromEmail: "JANE@acme.com", // case-insensitive match
			Subject:   "Re: [" + tkt.TicketNumber + "] Help",
			Text:      "second message",
		})
		require.NoError(t, err)

		var msgs int64
		db.DB.Model(&models.Message{}).Where("ticket_id = ?", tkt.ID).Count(&msgs)
		assert.Equal(t, int64(1), msgs, "the reply is appended as a message")

		var tickets int64
		db.DB.Model(&models.Ticket{}).Count(&tickets)
		assert.Equal(t, int64(1), tickets, "no new ticket is opened")
	})
}

func TestIngestEmail_SpoofedSenderOpensNewTicket(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))
		require.NoError(t, svc.IngestEmail(context.Background(), email.InboundEmail{
			FromEmail: "jane@acme.com", Subject: "Help", Text: "first",
		}))

		var tkt models.Ticket
		require.NoError(t, db.DB.First(&tkt).Error)

		// A different sender referencing the ticket must NOT append to it.
		require.NoError(t, svc.IngestEmail(context.Background(), email.InboundEmail{
			FromEmail: "attacker@evil.com",
			Subject:   "Re: [" + tkt.TicketNumber + "] Help",
			Text:      "inject",
		}))

		var tickets int64
		db.DB.Model(&models.Ticket{}).Count(&tickets)
		assert.Equal(t, int64(2), tickets, "spoofed reply opens a separate ticket")

		var msgs int64
		db.DB.Model(&models.Message{}).Where("ticket_id = ?", tkt.ID).Count(&msgs)
		assert.Equal(t, int64(0), msgs, "the original ticket is untouched")
	})
}
