package department

import (
	"fmt"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Department{}))
	return db
}

func staff(t *testing.T, db *gorm.DB, name string) models.User {
	t.Helper()
	u := models.User{Email: name + "@x.local", Username: name, PasswordHash: "-", Role: "engineer", IsActive: true}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func TestCreateRejectsCycle(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	root, err := svc.Create(CreateInput{Name: "root"})
	require.NoError(t, err)
	child, err := svc.Create(CreateInput{Name: "child", ParentID: &root.ID})
	require.NoError(t, err)
	err = svc.Update(root.ID, UpdateInput{ParentID: &child.ID})
	require.ErrorIs(t, err, ErrCycle)
}

func TestSupervisorOfWalksUp(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	mgrRoot := staff(t, db, "rootmgr")
	mgrChild := staff(t, db, "childmgr")
	agent := staff(t, db, "agent")

	root, _ := svc.Create(CreateInput{Name: "root", ManagerID: &mgrRoot.ID})
	child, _ := svc.Create(CreateInput{Name: "child", ParentID: &root.ID, ManagerID: &mgrChild.ID})
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", agent.ID).Update("department_id", child.ID).Error)
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", mgrChild.ID).Update("department_id", child.ID).Error)

	sup, err := svc.SupervisorOf(agent.ID)
	require.NoError(t, err)
	require.NotNil(t, sup)
	require.Equal(t, mgrChild.ID, sup.ID)

	sup2, err := svc.SupervisorOf(mgrChild.ID)
	require.NoError(t, err)
	require.NotNil(t, sup2)
	require.Equal(t, mgrRoot.ID, sup2.ID)

	require.NoError(t, db.Model(&models.User{}).Where("id = ?", mgrRoot.ID).Update("department_id", root.ID).Error)
	sup3, err := svc.SupervisorOf(mgrRoot.ID)
	require.NoError(t, err)
	require.Nil(t, sup3)
}

func TestDeptScopeForManagerSubtree(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	mgr := staff(t, db, "mgr")
	root, _ := svc.Create(CreateInput{Name: "root", ManagerID: &mgr.ID})
	child, _ := svc.Create(CreateInput{Name: "child", ParentID: &root.ID})
	grand, _ := svc.Create(CreateInput{Name: "grand", ParentID: &child.ID})
	other, _ := svc.Create(CreateInput{Name: "other"})

	scope, err := svc.DeptScopeFor(mgr.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{root.ID, child.ID, grand.ID}, scope)
	require.NotContains(t, scope, other.ID)

	plain := staff(t, db, "plain")
	scope2, err := svc.DeptScopeFor(plain.ID)
	require.NoError(t, err)
	require.Empty(t, scope2)
}
