package authz

import (
	"fmt"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func scopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Ticket{}))
	return db
}

func countScoped(t *testing.T, db *gorm.DB, actor Actor, opts ScopeOptions) int64 {
	t.Helper()
	q := Scope(db, db.Model(&models.Ticket{}), actor, opts)
	var n int64
	require.NoError(t, q.Count(&n).Error)
	return n
}

func TestScopeCustomerIsolation(t *testing.T) {
	db := scopeTestDB(t)
	cid := uint(7)
	require.NoError(t, db.Create(&models.Ticket{TicketNumber: "TK-1", CustomerID: &cid, Title: "a"}).Error)
	other := uint(8)
	require.NoError(t, db.Create(&models.Ticket{TicketNumber: "TK-2", CustomerID: &other, Title: "b"}).Error)
	opts := ScopeOptions{CustomerColumn: "customer_id", AssigneeColumn: "assigned_to"}

	cust := Actor{UserID: 1, Role: RoleCustomer, CustomerID: &cid}
	require.Equal(t, int64(1), countScoped(t, db, cust, opts))

	require.Equal(t, int64(0), countScoped(t, db, Actor{UserID: 2, Role: RoleCustomer}, opts))

	require.Equal(t, int64(2), countScoped(t, db, Actor{UserID: 3, Role: RoleAdmin}, opts))
}

func TestScopeDepartmentIsolation(t *testing.T) {
	db := scopeTestDB(t)
	d1, d2 := uint(10), uint(20)
	u1 := models.User{Email: "u1@x", Username: "u1", PasswordHash: "-", Role: "engineer", DepartmentID: &d1}
	u2 := models.User{Email: "u2@x", Username: "u2", PasswordHash: "-", Role: "engineer", DepartmentID: &d2}
	require.NoError(t, db.Create(&u1).Error)
	require.NoError(t, db.Create(&u2).Error)
	require.NoError(t, db.Create(&models.Ticket{TicketNumber: "TK-1", Title: "t1", AssignedTo: &u1.ID}).Error)
	require.NoError(t, db.Create(&models.Ticket{TicketNumber: "TK-2", Title: "t2", AssignedTo: &u2.ID}).Error)

	mgr := Actor{UserID: 99, Role: RoleEngineer, DeptScope: []uint{d1}}

	require.Equal(t, int64(2), countScoped(t, db, mgr, ScopeOptions{CustomerColumn: "customer_id", AssigneeColumn: "assigned_to", DepartmentIsolation: false}))
	require.Equal(t, int64(1), countScoped(t, db, mgr, ScopeOptions{CustomerColumn: "customer_id", AssigneeColumn: "assigned_to", DepartmentIsolation: true}))
}
