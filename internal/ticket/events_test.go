package ticket

import (
	"testing"

	"github.com/company/smartticket/internal/automation"
	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBus_MessageCreatedEventFired(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		slaCalc := sla.NewCalculator(db.DB)
		svc := NewService(db.DB, slaCalc)

		// Wire a bus and record events.
		bus := automation.NewBus()
		var got []automation.Event
		bus.Subscribe(automation.EventMessageCreated, func(e automation.Event) {
			got = append(got, e)
		})
		svc.SetBus(bus)

		// Create a ticket, then a message.
		user := createTestUser(t, db)
		actor := authz.Actor{Role: authz.RoleAdmin, UserID: user.ID}

		tkt, err := svc.CreateTicket(actor, user.ID, &CreateTicketRequest{
			Title:          "Event test ticket",
			Description:    "Testing event emission",
			Priority:       "medium",
			Severity:       "minor",
			RequesterName:  "Test User",
			RequesterEmail: "test@example.com",
		})
		require.NoError(t, err)

		_, err = svc.CreateMessage(actor, tkt.ID, user.ID, &CreateMessageRequest{
			Content:     "Hello, world",
			ContentType: "text",
		})
		require.NoError(t, err)

		require.Len(t, got, 1)
		assert.Equal(t, automation.EventMessageCreated, got[0].Type)
		assert.Equal(t, tkt.ID, got[0].TicketID)
		assert.Equal(t, user.ID, got[0].ActorID)
	})
}

func TestEventBus_TicketCreatedEventFired(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		slaCalc := sla.NewCalculator(db.DB)
		svc := NewService(db.DB, slaCalc)

		bus := automation.NewBus()
		var got []automation.Event
		bus.Subscribe(automation.EventTicketCreated, func(e automation.Event) {
			got = append(got, e)
		})
		svc.SetBus(bus)

		user := createTestUser(t, db)
		actor := authz.Actor{Role: authz.RoleAdmin, UserID: user.ID}

		tkt, err := svc.CreateTicket(actor, user.ID, &CreateTicketRequest{
			Title:          "Create event test",
			Description:    "desc",
			Priority:       "low",
			Severity:       "trivial",
			RequesterName:  "Alice",
			RequesterEmail: "alice@example.com",
		})
		require.NoError(t, err)

		require.Len(t, got, 1)
		assert.Equal(t, automation.EventTicketCreated, got[0].Type)
		assert.Equal(t, tkt.ID, got[0].TicketID)
	})
}

func TestEventBus_TicketResolvedEventFired(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		slaCalc := sla.NewCalculator(db.DB)
		svc := NewService(db.DB, slaCalc)

		bus := automation.NewBus()
		var got []automation.Event
		bus.Subscribe(automation.EventTicketResolved, func(e automation.Event) {
			got = append(got, e)
		})
		svc.SetBus(bus)

		user := createTestUser(t, db)
		tkt := createTestTicket(t, db, user.ID)

		_, err := svc.UpdateTicket(teamActor, tkt.ID, user.ID, &UpdateTicketRequest{
			Status: "resolved",
		})
		require.NoError(t, err)

		require.Len(t, got, 1)
		assert.Equal(t, automation.EventTicketResolved, got[0].Type)
		assert.Equal(t, tkt.ID, got[0].TicketID)
	})
}

func TestSetBus_NilSafe(t *testing.T) {
	// Ensure existing constructor + no SetBus still works (no panic).
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		slaCalc := sla.NewCalculator(db.DB)
		svc := NewService(db.DB, slaCalc) // bus is nil

		user := createTestUser(t, db)
		actor := authz.Actor{Role: authz.RoleAdmin, UserID: user.ID}

		tkt, err := svc.CreateTicket(actor, user.ID, &CreateTicketRequest{
			Title:          "No bus test",
			Description:    "desc",
			Priority:       "low",
			Severity:       "trivial",
			RequesterName:  "Bob",
			RequesterEmail: "bob@example.com",
		})
		require.NoError(t, err)
		assert.NotNil(t, tkt)
	})
}
