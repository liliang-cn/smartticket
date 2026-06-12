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

// fakeDeptScoper is a test double for DeptScoper.
type fakeDeptScoper struct{ scope []uint }

func (f fakeDeptScoper) DeptScopeFor(uint) ([]uint, error) { return f.scope, nil }

// deptFixture holds all entities for department-scoping tests.
type deptFixture struct {
	service  *Service
	deptA    uint // department A
	deptB    uint // department B
	agentA   uint // agent assigned to dept A
	agentB   uint // agent assigned to dept B
	ticketA  uint // ticket assigned to agentA
	ticketB  uint // ticket assigned to agentB
	noAssign uint // ticket with no assignee (unassigned)
	custA    uint // customer org for isolation regression
	custB    uint // customer org for isolation regression
	ticketCA uint // ticket for customer A (no dept assignment)
	ticketCB uint // ticket for customer B (no dept assignment)
	actorCA  authz.Actor
	actorCB  authz.Actor
}

func setupDeptFixture(t *testing.T, db *database.Database) *deptFixture {
	t.Helper()

	// Create two departments.
	deptA := &models.Department{Name: "Engineering"}
	deptB := &models.Department{Name: "Support"}
	require.NoError(t, db.DB.Create(deptA).Error)
	require.NoError(t, db.DB.Create(deptB).Error)

	deptAID := deptA.ID
	deptBID := deptB.ID

	// Create two staff agents, one per department.
	agentA := &models.User{
		Email: "agent-a@example.com", Username: "agenta",
		FirstName: "Agent", LastName: "A", Role: "engineer", IsActive: true,
		DepartmentID: &deptAID,
	}
	agentB := &models.User{
		Email: "agent-b@example.com", Username: "agentb",
		FirstName: "Agent", LastName: "B", Role: "engineer", IsActive: true,
		DepartmentID: &deptBID,
	}
	require.NoError(t, db.DB.Create(agentA).Error)
	require.NoError(t, db.DB.Create(agentB).Error)

	// Create tickets: one assigned to each agent, one unassigned.
	ticketA := &models.Ticket{
		TicketNumber: generateTicketNumber(), Title: "Ticket A", Description: "desc",
		Status: "open", Priority: "medium", Severity: "minor",
		RequesterName: "R", RequesterEmail: "r@example.com",
		AssignedTo: &agentA.ID,
	}
	ticketB := &models.Ticket{
		TicketNumber: generateTicketNumber(), Title: "Ticket B", Description: "desc",
		Status: "open", Priority: "medium", Severity: "minor",
		RequesterName: "R", RequesterEmail: "r@example.com",
		AssignedTo: &agentB.ID,
	}
	noAssign := &models.Ticket{
		TicketNumber: generateTicketNumber(), Title: "Unassigned Ticket", Description: "desc",
		Status: "open", Priority: "medium", Severity: "minor",
		RequesterName: "R", RequesterEmail: "r@example.com",
	}
	require.NoError(t, db.DB.Create(ticketA).Error)
	require.NoError(t, db.DB.Create(ticketB).Error)
	require.NoError(t, db.DB.Create(noAssign).Error)

	// Create two customer orgs with one customer user and one ticket each (for regression).
	codeA, codeB := "DCUSTA", "DCUSTB"
	custA := &models.Customer{Name: "DeptCustA", Code: &codeA, IsActive: true}
	custB := &models.Customer{Name: "DeptCustB", Code: &codeB, IsActive: true}
	require.NoError(t, db.DB.Create(custA).Error)
	require.NoError(t, db.DB.Create(custB).Error)

	cidA := custA.ID
	cidB := custB.ID

	userCA := &models.User{
		Email: "ca@dcusta.com", Username: "userCA", Role: "customer",
		IsActive: true, CustomerID: &cidA,
	}
	userCB := &models.User{
		Email: "cb@dcustb.com", Username: "userCB", Role: "customer",
		IsActive: true, CustomerID: &cidB,
	}
	require.NoError(t, db.DB.Create(userCA).Error)
	require.NoError(t, db.DB.Create(userCB).Error)

	ticketCA := &models.Ticket{
		TicketNumber: generateTicketNumber(), Title: "Cust A Ticket", Description: "desc",
		Status: "open", Priority: "medium", Severity: "minor",
		CustomerID: &cidA, RequesterName: "CA", RequesterEmail: "ca@dcusta.com",
	}
	ticketCB := &models.Ticket{
		TicketNumber: generateTicketNumber(), Title: "Cust B Ticket", Description: "desc",
		Status: "open", Priority: "medium", Severity: "minor",
		CustomerID: &cidB, RequesterName: "CB", RequesterEmail: "cb@dcustb.com",
	}
	require.NoError(t, db.DB.Create(ticketCA).Error)
	require.NoError(t, db.DB.Create(ticketCB).Error)

	slaCalc := sla.NewCalculator(db.DB)
	service := NewService(db.DB, slaCalc)

	return &deptFixture{
		service:  service,
		deptA:    deptAID,
		deptB:    deptBID,
		agentA:   agentA.ID,
		agentB:   agentB.ID,
		ticketA:  ticketA.ID,
		ticketB:  ticketB.ID,
		noAssign: noAssign.ID,
		custA:    cidA,
		custB:    cidB,
		ticketCA: ticketCA.ID,
		ticketCB: ticketCB.ID,
		actorCA:  authz.Actor{UserID: userCA.ID, Role: authz.RoleCustomer, CustomerID: &cidA},
		actorCB:  authz.Actor{UserID: userCB.ID, Role: authz.RoleCustomer, CustomerID: &cidB},
	}
}

// (a) isolation OFF → a manager actor's ListTickets returns all staff tickets.
func TestDeptScope_IsolationOff_ManagerSeesAll(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupDeptFixture(t, db)
		// Isolation OFF (default), deptScoper returns deptA scope.
		f.service.SetDeptScoper(fakeDeptScoper{scope: []uint{f.deptA}})
		// departmentIsolationFn not set → OFF

		managerActor := authz.Actor{UserID: 99, Role: "engineer"} // non-admin team actor
		list, err := f.service.ListTickets(managerActor, 1, 100, map[string]interface{}{})
		require.NoError(t, err)
		// Should see all tickets (dept + customer tickets).
		assert.GreaterOrEqual(t, int(list.Total), 3, "isolation OFF: should see all staff-visible tickets")
		ids := make(map[uint]bool)
		for _, tr := range list.Data {
			ids[tr.ID] = true
		}
		assert.True(t, ids[f.ticketA], "ticket assigned to agentA must be visible")
		assert.True(t, ids[f.ticketB], "ticket assigned to agentB must be visible")
		assert.True(t, ids[f.noAssign], "unassigned ticket must be visible")
	})
}

// (b) isolation ON → only tickets assigned to users in the manager's DeptScope.
func TestDeptScope_IsolationOn_ManagerSeesOnlySubtree(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupDeptFixture(t, db)
		// DeptScoper returns deptA only; isolation is ON.
		f.service.SetDeptScoper(fakeDeptScoper{scope: []uint{f.deptA}})
		f.service.SetDepartmentIsolation(func() bool { return true })

		managerActor := authz.Actor{UserID: 99, Role: "engineer"}
		list, err := f.service.ListTickets(managerActor, 1, 100, map[string]interface{}{})
		require.NoError(t, err)

		ids := make(map[uint]bool)
		for _, tr := range list.Data {
			ids[tr.ID] = true
		}
		assert.True(t, ids[f.ticketA], "ticket assigned to agentA (deptA) should be visible")
		assert.False(t, ids[f.ticketB], "ticket assigned to agentB (deptB) must NOT be visible")
		// Unassigned ticket: assigned_to is NULL, which doesn't match the subquery → invisible.
		assert.False(t, ids[f.noAssign], "unassigned ticket must NOT be visible under isolation")
	})
}

// (c) ListTicketsForDepartment always forces department scoping, even when isolation is OFF.
func TestDeptScope_ListTicketsForDepartment_ForcesDeptScope(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupDeptFixture(t, db)
		// isolation OFF, but ListTicketsForDepartment should still apply dept isolation.
		f.service.SetDeptScoper(fakeDeptScoper{scope: []uint{f.deptA}})
		// isolation fn NOT set (= false)

		managerActor := authz.Actor{UserID: 99, Role: "engineer"}
		list, err := f.service.ListTicketsForDepartment(managerActor, 1, 100, map[string]interface{}{})
		require.NoError(t, err)

		ids := make(map[uint]bool)
		for _, tr := range list.Data {
			ids[tr.ID] = true
		}
		assert.True(t, ids[f.ticketA], "ticket assigned to agentA (deptA) should be visible")
		assert.False(t, ids[f.ticketB], "ticket assigned to agentB (deptB) must NOT be visible")
		assert.False(t, ids[f.noAssign], "unassigned ticket must NOT be visible in dept view")
	})
}

// (d) Admin actor always sees everything regardless of isolation setting.
func TestDeptScope_AdminSeesAll(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupDeptFixture(t, db)
		f.service.SetDeptScoper(fakeDeptScoper{scope: []uint{f.deptA}})
		f.service.SetDepartmentIsolation(func() bool { return true })

		admin := authz.Actor{UserID: 1, Role: authz.RoleAdmin}
		list, err := f.service.ListTickets(admin, 1, 100, map[string]interface{}{})
		require.NoError(t, err)

		ids := make(map[uint]bool)
		for _, tr := range list.Data {
			ids[tr.ID] = true
		}
		assert.True(t, ids[f.ticketA], "admin must see ticketA")
		assert.True(t, ids[f.ticketB], "admin must see ticketB")
		assert.True(t, ids[f.noAssign], "admin must see unassigned ticket")
	})
}

// (f) Under isolation, a plain member (manages no department) still sees their
// OWN department's tickets — not nothing. Regression for the empty-DeptScope bug.
func TestDeptScope_IsolationOn_MemberSeesOwnDepartment(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupDeptFixture(t, db)
		// agentA manages no department (deptScoper returns empty) but belongs to deptA.
		f.service.SetDeptScoper(fakeDeptScoper{scope: nil})
		f.service.SetDepartmentIsolation(func() bool { return true })

		memberActor := authz.Actor{UserID: f.agentA, Role: "engineer"} // belongs to deptA
		list, err := f.service.ListTickets(memberActor, 1, 100, map[string]interface{}{})
		require.NoError(t, err)

		ids := make(map[uint]bool)
		for _, tr := range list.Data {
			ids[tr.ID] = true
		}
		assert.True(t, ids[f.ticketA], "member should see their own department's tickets")
		assert.False(t, ids[f.ticketB], "member must NOT see another department's tickets")
	})
}

// (e) Customer isolation regression: customer actor sees only their customer's tickets.
func TestDeptScope_CustomerIsolationRegression(t *testing.T) {
	testutils.WithTestDatabase(t, func(t *testing.T, db *database.Database) {
		f := setupDeptFixture(t, db)
		// Even with dept isolation ON, customer actors must only see their customer.
		f.service.SetDeptScoper(fakeDeptScoper{scope: []uint{f.deptA}})
		f.service.SetDepartmentIsolation(func() bool { return true })

		listA, err := f.service.ListTickets(f.actorCA, 1, 100, map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), listA.Total, "customer A should see exactly 1 ticket")
		require.Len(t, listA.Data, 1)
		assert.Equal(t, f.ticketCA, listA.Data[0].ID)

		listB, err := f.service.ListTickets(f.actorCB, 1, 100, map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), listB.Total, "customer B should see exactly 1 ticket")
		require.Len(t, listB.Data, 1)
		assert.Equal(t, f.ticketCB, listB.Data[0].ID)
	})
}
