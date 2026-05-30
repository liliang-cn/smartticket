package ticket

import (
	"testing"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessages_CustomerIsolationAndInternalHiding(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		svc := NewService(db.DB, sla.NewCalculator(db.DB))

		codeA, codeB := "MA", "MB"
		custA := &models.Customer{Name: "A", Code: &codeA, IsActive: true}
		custB := &models.Customer{Name: "B", Code: &codeB, IsActive: true}
		require.NoError(t, db.DB.Create(custA).Error)
		require.NoError(t, db.DB.Create(custB).Error)
		cidA := custA.ID

		// Ticket belongs to customer A.
		tk := &models.Ticket{TicketNumber: generateTicketNumber(), Title: "T", Status: "open", Priority: "medium", Severity: "minor", CustomerID: &cidA}
		require.NoError(t, db.DB.Create(tk).Error)
		// Two messages: one public, one internal.
		require.NoError(t, db.DB.Create(&models.Message{TicketID: tk.ID, Content: "public", ContentType: "text"}).Error)
		require.NoError(t, db.DB.Create(&models.Message{TicketID: tk.ID, Content: "internal note", ContentType: "text", IsInternal: true}).Error)

		teamActor := authz.Actor{Role: authz.RoleAdmin}
		custAActor := authz.Actor{Role: authz.RoleCustomer, CustomerID: &cidA}
		cidB := custB.ID
		custBActor := authz.Actor{Role: authz.RoleCustomer, CustomerID: &cidB}

		// Team sees both messages (incl. internal).
		teamMsgs, err := svc.ListMessages(teamActor, tk.ID)
		require.NoError(t, err)
		assert.Len(t, teamMsgs, 2)

		// Customer A sees only the public message (internal hidden).
		custMsgs, err := svc.ListMessages(custAActor, tk.ID)
		require.NoError(t, err)
		require.Len(t, custMsgs, 1)
		assert.Equal(t, "public", custMsgs[0].Content)
		assert.False(t, custMsgs[0].IsInternal)

		// Customer B cannot access customer A's ticket messages -> NotFound.
		_, err = svc.ListMessages(custBActor, tk.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Customer-created message can never be internal.
		msg, err := svc.CreateMessage(custAActor, tk.ID, 0, &CreateMessageRequest{Content: "hi", IsInternal: true})
		require.NoError(t, err)
		assert.False(t, msg.IsInternal, "customer messages must not be internal")

		// Customer B cannot post to customer A's ticket.
		_, err = svc.CreateMessage(custBActor, tk.ID, 0, &CreateMessageRequest{Content: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
