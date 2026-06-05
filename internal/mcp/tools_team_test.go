package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/company/smartticket/internal/team"
)

func TestTeamCreateAndList(t *testing.T) {
	b := new(MockBackend)

	b.On("CreateTeam", &team.CreateRequest{Name: "Support", Description: "Front-line support"}).
		Return(&team.TeamResponse{ID: 2, Name: "Support", Description: "Front-line support"}, nil)

	out, summary, err := teamCreate(b, teamCreateInput{Name: "Support", Description: "Front-line support"})
	require.NoError(t, err)
	assert.Equal(t, uint(2), out.ID)
	assert.Equal(t, "Support", out.Name)
	assert.Contains(t, summary, "#2")

	b.On("ListTeams").Return([]team.TeamResponse{
		{ID: 1, Name: "Engineering"},
		{ID: 2, Name: "Support", Description: "Front-line support"},
	}, nil)

	lout, _, err := teamList(b)
	require.NoError(t, err)
	assert.Equal(t, 2, lout.Total)
	require.Len(t, lout.Teams, 2)
	b.AssertExpectations(t)
}

func TestTeamGetTool(t *testing.T) {
	b := new(MockBackend)

	b.On("GetTeam", uint(1)).Return(&team.TeamResponse{ID: 1, Name: "Engineering"}, nil)

	out, summary, err := teamGet(b, teamGetInput{ID: 1})
	require.NoError(t, err)
	assert.Equal(t, uint(1), out.ID)
	assert.Equal(t, "Engineering", out.Name)
	assert.Contains(t, summary, "#1")
	b.AssertExpectations(t)
}

func TestTeamDeleteTool(t *testing.T) {
	b := new(MockBackend)

	b.On("DeleteTeam", uint(2)).Return(nil)

	out, summary, err := teamDelete(b, teamDeleteInput{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, uint(2), out.ID)
	assert.Equal(t, "deleted", out.Status)
	assert.Contains(t, summary, "#2")
	b.AssertExpectations(t)
}

func TestTeamMembersTool(t *testing.T) {
	b := new(MockBackend)

	b.On("ListTeamMembers", uint(1)).Return([]team.MemberResponse{
		{ID: 10, Email: "alice@example.com", Username: "alice", Role: "engineer"},
		{ID: 11, Email: "bob@example.com", Username: "bob", Role: "support"},
	}, nil)

	out, summary, err := teamMembers(b, teamMembersInput{TeamID: 1})
	require.NoError(t, err)
	assert.Equal(t, 2, out.Total)
	require.Len(t, out.Members, 2)
	assert.Equal(t, "alice@example.com", out.Members[0].Email)
	assert.Contains(t, summary, "#1")
	b.AssertExpectations(t)
}

func TestTeamAddAndRemoveMember(t *testing.T) {
	b := new(MockBackend)

	b.On("AddTeamMember", uint(1), uint(10)).Return(nil)
	aout, summary, err := teamAddMember(b, teamMemberInput{TeamID: 1, UserID: 10})
	require.NoError(t, err)
	assert.Equal(t, uint(1), aout.ID)
	assert.Equal(t, "member_added", aout.Status)
	assert.Contains(t, summary, "#10")

	b.On("RemoveTeamMember", uint(1), uint(10)).Return(nil)
	rout, _, err := teamRemoveMember(b, teamMemberInput{TeamID: 1, UserID: 10})
	require.NoError(t, err)
	assert.Equal(t, "member_removed", rout.Status)
	b.AssertExpectations(t)
}

func TestTeamPermissionDenied(t *testing.T) {
	// Session with no team:write cannot invoke team_create.
	ctx := ctxWithSession(newTestSession("team:read")) // has read but not write
	err := RequirePermission(ctx, "team:write")
	var permErr *PermissionError
	require.ErrorAs(t, err, &permErr)
	assert.Equal(t, "team:write", permErr.Code)
}

func TestTeamUpdateTool(t *testing.T) {
	b := new(MockBackend)
	name := "Tier-1 Support"

	b.On("UpdateTeam", uint(2), &team.UpdateRequest{Name: &name}).
		Return(&team.TeamResponse{ID: 2, Name: "Tier-1 Support"}, nil)

	out, _, err := teamUpdate(b, teamUpdateInput{ID: 2, Name: &name})
	require.NoError(t, err)
	assert.Equal(t, "Tier-1 Support", out.Name)
	b.AssertExpectations(t)
}
