package ticket

import (
	"context"
	"testing"

	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSupervisors is a stub SupervisorResolver returning a fixed supervisor.
type fakeSupervisors struct{ sup *models.User }

func (f fakeSupervisors) SupervisorOf(uint) (*models.User, error) { return f.sup, nil }

// notifyCall records arguments passed to a single Notify invocation.
type notifyCall struct {
	userIDs []uint
	ntype   string
}

// capturingNotifier captures all Notify calls for assertion.
type capturingNotifier struct{ calls []notifyCall }

func (c *capturingNotifier) Notify(_ context.Context, ids []uint, ntype, _, _, _ string, _ uint) {
	c.calls = append(c.calls, notifyCall{ids, ntype})
}

func TestEscalateAutomation_NotifiesSupervisor(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))

		// Create the assignee and their supervisor.
		assignee := createTestUser(t, db)
		supervisor := createTestUser(t, db)

		// Create a ticket assigned to the assignee with priority "medium".
		assigneeID := assignee.ID
		tkt := &models.Ticket{
			TicketNumber:   generateTicketNumber(),
			Title:          "Escalation test",
			Description:    "needs escalation",
			Status:         "open",
			Priority:       "medium",
			Severity:       "minor",
			Category:       "technical",
			RequesterName:  "Tester",
			RequesterEmail: "tester@example.com",
			IsDeleted:      false,
			AssignedTo:     &assigneeID,
		}
		require.NoError(t, db.DB.Create(tkt).Error)

		notifier := &capturingNotifier{}
		svc.SetNotifier(notifier)
		svc.SetSupervisors(fakeSupervisors{sup: supervisor})

		// Act.
		require.NoError(t, svc.EscalateAutomation(tkt.ID))

		// Priority must have been bumped to "high".
		var updated models.Ticket
		require.NoError(t, db.DB.First(&updated, tkt.ID).Error)
		assert.Equal(t, "high", updated.Priority, "priority should be bumped from medium to high")

		// Notifier must have received exactly one call of type "ticket_escalated"
		// targeting the supervisor.
		require.Len(t, notifier.calls, 1, "exactly one notification expected")
		assert.Equal(t, "ticket_escalated", notifier.calls[0].ntype)
		assert.Equal(t, []uint{supervisor.ID}, notifier.calls[0].userIDs)
	})
}

func TestEscalateAutomation_NoAssignee_NoPanic_NilNotify(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))

		supervisor := createTestUser(t, db)
		notifier := &capturingNotifier{}
		svc.SetNotifier(notifier)
		svc.SetSupervisors(fakeSupervisors{sup: supervisor})

		// Ticket with no assignee.
		tkt := &models.Ticket{
			TicketNumber:   generateTicketNumber(),
			Title:          "Unassigned escalation test",
			Description:    "no one assigned",
			Status:         "open",
			Priority:       "low",
			Severity:       "minor",
			Category:       "technical",
			RequesterName:  "Tester",
			RequesterEmail: "tester@example.com",
			IsDeleted:      false,
			AssignedTo:     nil,
		}
		require.NoError(t, db.DB.Create(tkt).Error)

		// Must not panic, must not error.
		require.NoError(t, svc.EscalateAutomation(tkt.ID))

		// Priority still bumped.
		var updated models.Ticket
		require.NoError(t, db.DB.First(&updated, tkt.ID).Error)
		assert.Equal(t, "medium", updated.Priority, "priority should be bumped from low to medium")

		// No supervisor notification because there is no assignee.
		assert.Empty(t, notifier.calls, "no notifications expected when ticket has no assignee")
	})
}
