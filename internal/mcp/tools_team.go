package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/team"
)

// team.TeamResponse and team.MemberResponse are flat scalar structs with no
// slice/map fields, so they can be reused directly as MCP Output values.
// List outputs wrap a slice with omitempty to prevent JSON null on empty pages.

// ----------------------------------------------------------------------------
// Output types
// ----------------------------------------------------------------------------

type teamListOutput struct {
	Teams []team.TeamResponse `json:"teams,omitempty" jsonschema:"the teams"`
	Total int                 `json:"total" jsonschema:"total number of teams"`
}

type teamMembersOutput struct {
	Members []team.MemberResponse `json:"members,omitempty" jsonschema:"the team members"`
	Total   int                   `json:"total" jsonschema:"total number of members"`
}

type teamActionOutput struct {
	ID      uint   `json:"id" jsonschema:"the affected team ID"`
	Status  string `json:"status" jsonschema:"the result status"`
	Message string `json:"message" jsonschema:"a human-readable summary"`
}

// ----------------------------------------------------------------------------
// Input types
// ----------------------------------------------------------------------------

type teamListInput struct{}

type teamGetInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the team to retrieve"`
}

type teamCreateInput struct {
	Name        string `json:"name" jsonschema:"the team name (required, unique, max 120 chars)"`
	Description string `json:"description,omitempty" jsonschema:"optional team description"`
}

type teamUpdateInput struct {
	ID          uint    `json:"id" jsonschema:"the numeric ID of the team to update"`
	Name        *string `json:"name,omitempty" jsonschema:"new team name"`
	Description *string `json:"description,omitempty" jsonschema:"new description"`
}

type teamDeleteInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the team to delete"`
}

type teamMemberInput struct {
	TeamID uint `json:"team_id" jsonschema:"the numeric ID of the team"`
	UserID uint `json:"user_id" jsonschema:"the numeric ID of the user to add or remove"`
}

type teamMembersInput struct {
	TeamID uint `json:"team_id" jsonschema:"the numeric ID of the team whose members to list"`
}

// ----------------------------------------------------------------------------
// Registration
// ----------------------------------------------------------------------------

func registerTeamTools(s *mcp.Server, b Backend) {
	registerTool(s, "team_list",
		"List all teams ordered by name.",
		"team:read",
		func(_ context.Context, _ teamListInput) (teamListOutput, string, error) {
			return teamList(b)
		})

	registerTool(s, "team_get",
		"Retrieve a single team by numeric ID.",
		"team:read",
		func(_ context.Context, in teamGetInput) (team.TeamResponse, string, error) {
			return teamGet(b, in)
		})

	registerTool(s, "team_create",
		"Create a new team.",
		"team:write",
		func(_ context.Context, in teamCreateInput) (team.TeamResponse, string, error) {
			return teamCreate(b, in)
		})

	registerTool(s, "team_update",
		"Update the name and/or description of an existing team.",
		"team:write",
		func(_ context.Context, in teamUpdateInput) (team.TeamResponse, string, error) {
			return teamUpdate(b, in)
		})

	registerTool(s, "team_delete",
		"Delete a team and remove all its memberships.",
		"team:write",
		func(_ context.Context, in teamDeleteInput) (teamActionOutput, string, error) {
			return teamDelete(b, in)
		})

	registerTool(s, "team_add_member",
		"Add a user to a team (idempotent — succeeds even if the user is already a member).",
		"team:write",
		func(_ context.Context, in teamMemberInput) (teamActionOutput, string, error) {
			return teamAddMember(b, in)
		})

	registerTool(s, "team_remove_member",
		"Remove a user from a team (no-op if the user is not a member).",
		"team:write",
		func(_ context.Context, in teamMemberInput) (teamActionOutput, string, error) {
			return teamRemoveMember(b, in)
		})

	registerTool(s, "team_members",
		"List the users belonging to a team.",
		"team:read",
		func(_ context.Context, in teamMembersInput) (teamMembersOutput, string, error) {
			return teamMembers(b, in)
		})
}

// ----------------------------------------------------------------------------
// Handlers
// ----------------------------------------------------------------------------

func teamList(b Backend) (teamListOutput, string, error) {
	teams, err := b.ListTeams()
	if err != nil {
		return teamListOutput{}, "", err
	}
	out := teamListOutput{Teams: teams, Total: len(teams)}
	return out, fmt.Sprintf("Listed %d team(s).", len(teams)), nil
}

func teamGet(b Backend, in teamGetInput) (team.TeamResponse, string, error) {
	t, err := b.GetTeam(in.ID)
	if err != nil {
		return team.TeamResponse{}, "", err
	}
	return *t, fmt.Sprintf("Retrieved team #%d (%s).", t.ID, t.Name), nil
}

func teamCreate(b Backend, in teamCreateInput) (team.TeamResponse, string, error) {
	t, err := b.CreateTeam(&team.CreateRequest{Name: in.Name, Description: in.Description})
	if err != nil {
		return team.TeamResponse{}, "", err
	}
	return *t, fmt.Sprintf("Created team #%d (%s).", t.ID, t.Name), nil
}

func teamUpdate(b Backend, in teamUpdateInput) (team.TeamResponse, string, error) {
	t, err := b.UpdateTeam(in.ID, &team.UpdateRequest{Name: in.Name, Description: in.Description})
	if err != nil {
		return team.TeamResponse{}, "", err
	}
	return *t, fmt.Sprintf("Updated team #%d (%s).", t.ID, t.Name), nil
}

func teamDelete(b Backend, in teamDeleteInput) (teamActionOutput, string, error) {
	if err := b.DeleteTeam(in.ID); err != nil {
		return teamActionOutput{}, "", err
	}
	msg := fmt.Sprintf("Deleted team #%d.", in.ID)
	return teamActionOutput{ID: in.ID, Status: "deleted", Message: msg}, msg, nil
}

func teamAddMember(b Backend, in teamMemberInput) (teamActionOutput, string, error) {
	if err := b.AddTeamMember(in.TeamID, in.UserID); err != nil {
		return teamActionOutput{}, "", err
	}
	msg := fmt.Sprintf("Added user #%d to team #%d.", in.UserID, in.TeamID)
	return teamActionOutput{ID: in.TeamID, Status: "member_added", Message: msg}, msg, nil
}

func teamRemoveMember(b Backend, in teamMemberInput) (teamActionOutput, string, error) {
	if err := b.RemoveTeamMember(in.TeamID, in.UserID); err != nil {
		return teamActionOutput{}, "", err
	}
	msg := fmt.Sprintf("Removed user #%d from team #%d.", in.UserID, in.TeamID)
	return teamActionOutput{ID: in.TeamID, Status: "member_removed", Message: msg}, msg, nil
}

func teamMembers(b Backend, in teamMembersInput) (teamMembersOutput, string, error) {
	members, err := b.ListTeamMembers(in.TeamID)
	if err != nil {
		return teamMembersOutput{}, "", err
	}
	out := teamMembersOutput{Members: members, Total: len(members)}
	return out, fmt.Sprintf("Listed %d member(s) in team #%d.", len(members), in.TeamID), nil
}
