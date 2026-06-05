package team

import (
	"fmt"
	"testing"
	"time"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Team{}, &models.TeamMember{}, &models.User{}))
	return NewService(db)
}

func createUser(t *testing.T, svc *Service) *models.User {
	t.Helper()
	u := &models.User{
		Email:        fmt.Sprintf("u%d@test.io", time.Now().UnixNano()),
		Username:     fmt.Sprintf("u%d", time.Now().UnixNano()),
		PasswordHash: "x",
		IsActive:     true,
		Role:         "engineer",
	}
	require.NoError(t, svc.db.Create(u).Error)
	return u
}

func TestCreateTeam(t *testing.T) {
	svc := newTestService(t)
	team, err := svc.CreateTeam(&CreateRequest{Name: "Platform", Description: "Core platform"})
	require.NoError(t, err)
	require.NotZero(t, team.ID)
	assert.Equal(t, "Platform", team.Name)
	assert.Equal(t, "Core platform", team.Description)
}

func TestCreateTeam_DuplicateNameRejected(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.CreateTeam(&CreateRequest{Name: "Alpha"})
	require.NoError(t, err)
	_, err = svc.CreateTeam(&CreateRequest{Name: "Alpha"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGetTeam_NotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.GetTeam(9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateTeam(t *testing.T) {
	svc := newTestService(t)
	created, err := svc.CreateTeam(&CreateRequest{Name: "Old"})
	require.NoError(t, err)

	newName := "New"
	updated, err := svc.UpdateTeam(created.ID, &UpdateRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "New", updated.Name)
}

func TestDeleteTeam(t *testing.T) {
	svc := newTestService(t)
	created, err := svc.CreateTeam(&CreateRequest{Name: "ToDelete"})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteTeam(created.ID))
	_, err = svc.GetTeam(created.ID)
	require.Error(t, err)
}

func TestDeleteTeam_NotFound(t *testing.T) {
	svc := newTestService(t)
	err := svc.DeleteTeam(9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListTeams(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.CreateTeam(&CreateRequest{Name: "Zeta"})
	require.NoError(t, err)
	_, err = svc.CreateTeam(&CreateRequest{Name: "Alpha"})
	require.NoError(t, err)

	teams, err := svc.ListTeams()
	require.NoError(t, err)
	require.Len(t, teams, 2)
	// Should be sorted alphabetically
	assert.Equal(t, "Alpha", teams[0].Name)
	assert.Equal(t, "Zeta", teams[1].Name)
}

func TestAddMember(t *testing.T) {
	svc := newTestService(t)
	team, err := svc.CreateTeam(&CreateRequest{Name: "T1"})
	require.NoError(t, err)
	user := createUser(t, svc)

	require.NoError(t, svc.AddMember(team.ID, user.ID))
	members, err := svc.ListMembers(team.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, user.ID, members[0].ID)
}

func TestAddMember_Idempotent(t *testing.T) {
	svc := newTestService(t)
	team, err := svc.CreateTeam(&CreateRequest{Name: "T2"})
	require.NoError(t, err)
	user := createUser(t, svc)

	require.NoError(t, svc.AddMember(team.ID, user.ID))
	// Adding the same user again should succeed (no error, no duplicate row).
	require.NoError(t, svc.AddMember(team.ID, user.ID))
	members, err := svc.ListMembers(team.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
}

func TestRemoveMember(t *testing.T) {
	svc := newTestService(t)
	team, err := svc.CreateTeam(&CreateRequest{Name: "T3"})
	require.NoError(t, err)
	user := createUser(t, svc)

	require.NoError(t, svc.AddMember(team.ID, user.ID))
	require.NoError(t, svc.RemoveMember(team.ID, user.ID))

	members, err := svc.ListMembers(team.ID)
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestRemoveMember_NotMember_IsNoOp(t *testing.T) {
	svc := newTestService(t)
	team, err := svc.CreateTeam(&CreateRequest{Name: "T4"})
	require.NoError(t, err)
	// Removing a user who is not in the team is not an error.
	require.NoError(t, svc.RemoveMember(team.ID, 9999))
}

func TestListMembers_TeamNotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.ListMembers(9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAddMember_TeamNotFound(t *testing.T) {
	svc := newTestService(t)
	user := createUser(t, svc)
	err := svc.AddMember(9999, user.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAddMember_UserNotFound(t *testing.T) {
	svc := newTestService(t)
	team, err := svc.CreateTeam(&CreateRequest{Name: "T5"})
	require.NoError(t, err)
	err = svc.AddMember(team.ID, 9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteTeam_RemovesMemberships(t *testing.T) {
	svc := newTestService(t)
	team, err := svc.CreateTeam(&CreateRequest{Name: "T6"})
	require.NoError(t, err)
	u1 := createUser(t, svc)
	u2 := createUser(t, svc)
	require.NoError(t, svc.AddMember(team.ID, u1.ID))
	require.NoError(t, svc.AddMember(team.ID, u2.ID))

	require.NoError(t, svc.DeleteTeam(team.ID))

	// All TeamMember rows for this team should be gone.
	var count int64
	svc.db.Model(&models.TeamMember{}).Where("team_id = ?", team.ID).Count(&count)
	assert.Zero(t, count)
}
