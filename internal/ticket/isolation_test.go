package ticket

import (
	"testing"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/database"
	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"github.com/company/smartticket/internal/sla"
	"github.com/company/smartticket/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolationFixture builds two customers (A and B), one customer-role user each,
// and one ticket per customer. It returns the service and the customer IDs.
type isolationFixture struct {
	service   *Service
	custA     uint
	custB     uint
	ticketA   uint
	ticketB   uint
	actorA    authz.Actor
	actorB    authz.Actor
	teamActor authz.Actor
	userAID   uint
}

func setupIsolation(t *testing.T, db *database.Database) *isolationFixture {
	t.Helper()
	slaCalc := sla.NewCalculator(db.DB)
	service := NewService(db.DB, slaCalc)

	codeA, codeB := "CUSTA", "CUSTB"
	custA := &models.Customer{Name: "Customer A", Code: &codeA, IsActive: true}
	custB := &models.Customer{Name: "Customer B", Code: &codeB, IsActive: true}
	require.NoError(t, db.DB.Create(custA).Error)
	require.NoError(t, db.DB.Create(custB).Error)

	cidA := custA.ID
	cidB := custB.ID

	userA := &models.User{
		Email: "a@custa.com", Username: "usera", FirstName: "A", LastName: "User",
		PasswordHash: "$2a$10$dummy.hash.for.testing", Role: "customer", IsActive: true, CustomerID: &cidA,
	}
	userB := &models.User{
		Email: "b@custb.com", Username: "userb", FirstName: "B", LastName: "User",
		PasswordHash: "$2a$10$dummy.hash.for.testing", Role: "customer", IsActive: true, CustomerID: &cidB,
	}
	require.NoError(t, db.DB.Create(userA).Error)
	require.NoError(t, db.DB.Create(userB).Error)

	ticketA := &models.Ticket{
		TicketNumber: generateTicketNumber(), Title: "A ticket", Description: "desc",
		Status: "open", Priority: "medium", Severity: "minor", CustomerID: &cidA,
		RequesterName: "A", RequesterEmail: "a@custa.com",
	}
	ticketB := &models.Ticket{
		TicketNumber: generateTicketNumber(), Title: "B ticket", Description: "desc",
		Status: "open", Priority: "medium", Severity: "minor", CustomerID: &cidB,
		RequesterName: "B", RequesterEmail: "b@custb.com",
	}
	require.NoError(t, db.DB.Create(ticketA).Error)
	require.NoError(t, db.DB.Create(ticketB).Error)

	return &isolationFixture{
		service:   service,
		custA:     cidA,
		custB:     cidB,
		ticketA:   ticketA.ID,
		ticketB:   ticketB.ID,
		actorA:    authz.Actor{UserID: userA.ID, Role: authz.RoleCustomer, CustomerID: &cidA},
		actorB:    authz.Actor{UserID: userB.ID, Role: authz.RoleCustomer, CustomerID: &cidB},
		teamActor: authz.Actor{UserID: 1, Role: authz.RoleAdmin},
		userAID:   userA.ID,
	}
}

func TestTicketIsolation_CustomerListsOnlyOwnTickets(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupIsolation(t, db)

		listA, err := f.service.ListTickets(f.actorA, 1, 20, map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), listA.Total)
		require.Len(t, listA.Data, 1)
		assert.Equal(t, f.ticketA, listA.Data[0].ID)

		listB, err := f.service.ListTickets(f.actorB, 1, 20, map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), listB.Total)
		require.Len(t, listB.Data, 1)
		assert.Equal(t, f.ticketB, listB.Data[0].ID)
	})
}

func TestTicketIsolation_TeamSeesAllTickets(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupIsolation(t, db)

		list, err := f.service.ListTickets(f.teamActor, 1, 20, map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, int64(2), list.Total)
		assert.Len(t, list.Data, 2)
	})
}

func TestTicketIsolation_CrossCustomerGetReturnsNotFound(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupIsolation(t, db)

		// Actor A can read its own ticket.
		own, err := f.service.GetTicket(f.actorA, f.ticketA)
		require.NoError(t, err)
		assert.Equal(t, f.ticketA, own.ID)

		// Actor A reading B's ticket -> NotFound (not Forbidden; no disclosure).
		_, err = f.service.GetTicket(f.actorA, f.ticketB)
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeNotFound, appErr.Code)
	})
}

func TestTicketIsolation_CustomerCreateForcesOwnCustomerID(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupIsolation(t, db)

		// Customer A tries to file a ticket on behalf of customer B; it must be
		// forced back to A's own customer.
		req := &CreateTicketRequest{
			Title:          "Filed by A",
			Description:    "should land on A",
			Priority:       "medium",
			Severity:       "minor",
			CustomerID:     &f.custB, // attempted spoof
			RequesterName:  "A User",
			RequesterEmail: "a@custa.com",
		}
		created, err := f.service.CreateTicket(f.actorA, f.userAID, req)
		require.NoError(t, err)

		var stored models.Ticket
		require.NoError(t, db.DB.First(&stored, created.ID).Error)
		require.NotNil(t, stored.CustomerID)
		assert.Equal(t, f.custA, *stored.CustomerID)
	})
}

func TestTicketIsolation_CustomerCannotAssign(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupIsolation(t, db)

		err := f.service.AssignTicket(f.actorA, f.ticketA, 1)
		require.Error(t, err)
		var appErr *apperrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperrors.ErrCodeForbidden, appErr.Code)
	})
}

func TestTicketIsolation_TeamCanAssign(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupIsolation(t, db)

		err := f.service.AssignTicket(f.teamActor, f.ticketA, f.userAID)
		require.NoError(t, err)

		got, err := f.service.GetTicket(f.teamActor, f.ticketA)
		require.NoError(t, err)
		require.NotNil(t, got.AssignedTo)
		assert.Equal(t, f.userAID, *got.AssignedTo)
	})
}
