package ticket

import (
	"context"
	"testing"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedNotify records a single Notify call.
type capturedNotify struct {
	userIDs []uint
	ntype   string
	title   string
	body    string
	refType string
	refID   uint
}

// fakeNotifier captures Notify calls for assertions.
type fakeNotifier struct{ calls []capturedNotify }

func (f *fakeNotifier) Notify(_ context.Context, userIDs []uint, ntype, title, body, refType string, refID uint) {
	f.calls = append(f.calls, capturedNotify{userIDs, ntype, title, body, refType, refID})
}

func TestCreateMessage_NotifiesCustomerOnPublicReplyButNotInternalNote(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		fake := &fakeNotifier{}
		svc := NewService(db.DB, sla.NewCalculator(db.DB))
		svc.SetNotifier(fake)

		code := "NC"
		cust := &models.Customer{Name: "C", Code: &code, IsActive: true}
		require.NoError(t, db.DB.Create(cust).Error)
		cid := cust.ID

		// A customer-side user who should receive the reply notification.
		custUser := &models.User{Email: "c@x.io", Username: "cuser", PasswordHash: "x", Role: authz.RoleCustomer, IsActive: true, CustomerID: &cid}
		require.NoError(t, db.DB.Create(custUser).Error)

		tk := &models.Ticket{TicketNumber: generateTicketNumber(), Title: "T", Status: "open", Priority: "medium", Severity: "minor", CustomerID: &cid}
		require.NoError(t, db.DB.Create(tk).Error)

		teamActor := authz.Actor{UserID: 999, Role: authz.RoleAdmin}

		// Public reply from team -> customer-side user gets notified.
		_, err := svc.CreateMessage(teamActor, tk.ID, 999, &CreateMessageRequest{Content: "public reply"})
		require.NoError(t, err)
		require.Len(t, fake.calls, 1)
		assert.Equal(t, "ticket_reply", fake.calls[0].ntype)
		assert.Equal(t, tk.ID, fake.calls[0].refID)
		assert.Equal(t, "ticket", fake.calls[0].refType)
		assert.Contains(t, fake.calls[0].userIDs, custUser.ID)

		// Internal note from team -> NO notification to the customer.
		_, err = svc.CreateMessage(teamActor, tk.ID, 999, &CreateMessageRequest{Content: "secret", IsInternal: true})
		require.NoError(t, err)
		assert.Len(t, fake.calls, 1, "internal notes must never notify customers")
	})
}

func TestCreateMessage_CustomerReplyNotifiesAssignedAgent(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		fake := &fakeNotifier{}
		svc := NewService(db.DB, sla.NewCalculator(db.DB))
		svc.SetNotifier(fake)

		code := "NA"
		cust := &models.Customer{Name: "C", Code: &code, IsActive: true}
		require.NoError(t, db.DB.Create(cust).Error)
		cid := cust.ID

		agent := &models.User{Email: "a@x.io", Username: "agent", PasswordHash: "x", Role: authz.RoleEngineer, IsActive: true}
		require.NoError(t, db.DB.Create(agent).Error)
		agentID := agent.ID

		tk := &models.Ticket{TicketNumber: generateTicketNumber(), Title: "T", Status: "open", Priority: "medium", Severity: "minor", CustomerID: &cid, AssignedTo: &agentID}
		require.NoError(t, db.DB.Create(tk).Error)

		custActor := authz.Actor{UserID: 7, Role: authz.RoleCustomer, CustomerID: &cid}
		_, err := svc.CreateMessage(custActor, tk.ID, 7, &CreateMessageRequest{Content: "hi"})
		require.NoError(t, err)
		require.Len(t, fake.calls, 1)
		assert.Equal(t, "ticket_reply", fake.calls[0].ntype)
		assert.Equal(t, []uint{agentID}, fake.calls[0].userIDs)
	})
}
