package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/company/smartticket/internal/automation"
)

// automationRuleResponse is the MCP-local view of an automation.RuleResponse.
// Conditions and Actions are raw JSON strings (the service layer stores them
// as JSON text); they are passed through as strings in the MCP schema.
type automationRuleResponse struct {
	ID          uint      `json:"id" jsonschema:"the rule's numeric ID"`
	Name        string    `json:"name" jsonschema:"the rule name"`
	Description string    `json:"description,omitempty" jsonschema:"optional description"`
	Enabled     bool      `json:"enabled" jsonschema:"whether the rule is active"`
	Event       string    `json:"event" jsonschema:"the trigger event type (ticket.created, ticket.updated, message.created, ticket.sla_warning, schedule)"`
	Match       string    `json:"match" jsonschema:"condition matching mode: all or any"`
	Conditions  string    `json:"conditions,omitempty" jsonschema:"JSON array of condition objects [{field,op,value}]"`
	Actions     string    `json:"actions,omitempty" jsonschema:"JSON array of action objects [{type,params}]"`
	Position    int       `json:"position" jsonschema:"evaluation order (ascending)"`
	CreatedAt   time.Time `json:"created_at" jsonschema:"when the rule was created"`
	UpdatedAt   time.Time `json:"updated_at" jsonschema:"when the rule was last updated"`
}

func automationRuleResponseFrom(r *automation.RuleResponse) automationRuleResponse {
	if r == nil {
		return automationRuleResponse{}
	}
	return automationRuleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Enabled:     r.Enabled,
		Event:       r.Event,
		Match:       r.Match,
		Conditions:  r.Conditions,
		Actions:     r.Actions,
		Position:    r.Position,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func automationRuleResponsesFrom(rs []automation.RuleResponse) []automationRuleResponse {
	if len(rs) == 0 {
		return nil
	}
	out := make([]automationRuleResponse, len(rs))
	for i := range rs {
		out[i] = automationRuleResponseFrom(&rs[i])
	}
	return out
}

// automationListOutput is the structured output for automation_list.
type automationListOutput struct {
	Rules []automationRuleResponse `json:"rules,omitempty" jsonschema:"the automation rules ordered by position"`
	Total int                      `json:"total" jsonschema:"total number of rules"`
}

// ----------------------------------------------------------------------------
// Input types
// ----------------------------------------------------------------------------

type automationListInput struct{}

type automationGetInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the automation rule to retrieve"`
}

type automationCreateInput struct {
	Name        string `json:"name" jsonschema:"the rule name (required)"`
	Description string `json:"description,omitempty" jsonschema:"optional description"`
	Enabled     bool   `json:"enabled,omitempty" jsonschema:"whether to enable the rule immediately (default false)"`
	Event       string `json:"event" jsonschema:"trigger event type: ticket.created | ticket.updated | message.created | ticket.sla_warning | schedule (required)"`
	Match       string `json:"match,omitempty" jsonschema:"condition matching mode: all or any (defaults all)"`
	Conditions  string `json:"conditions,omitempty" jsonschema:"JSON array of condition objects e.g. [{\"field\":\"priority\",\"op\":\"eq\",\"value\":\"high\"}]"`
	Actions     string `json:"actions,omitempty" jsonschema:"JSON array of action objects e.g. [{\"type\":\"assign_team\",\"params\":{\"team_id\":\"1\"}}]"`
	Position    int    `json:"position,omitempty" jsonschema:"evaluation order position (default 0)"`
}

type automationUpdateInput struct {
	ID          uint    `json:"id" jsonschema:"the numeric ID of the rule to update"`
	Name        *string `json:"name,omitempty" jsonschema:"new rule name"`
	Description *string `json:"description,omitempty" jsonschema:"new description"`
	Enabled     *bool   `json:"enabled,omitempty" jsonschema:"new enabled state"`
	Event       *string `json:"event,omitempty" jsonschema:"new trigger event type"`
	Match       *string `json:"match,omitempty" jsonschema:"new condition matching mode: all or any"`
	Conditions  *string `json:"conditions,omitempty" jsonschema:"new JSON conditions array"`
	Actions     *string `json:"actions,omitempty" jsonschema:"new JSON actions array"`
	Position    *int    `json:"position,omitempty" jsonschema:"new position"`
}

type automationDeleteInput struct {
	ID uint `json:"id" jsonschema:"the numeric ID of the automation rule to delete"`
}

// ----------------------------------------------------------------------------
// Registration
// ----------------------------------------------------------------------------

func registerAutomationTools(s *mcp.Server, b Backend) {
	registerTool(s, "automation_list",
		"List all automation rules ordered by their evaluation position.",
		"automation:read",
		func(_ context.Context, _ automationListInput) (automationListOutput, string, error) {
			return automationList(b)
		})

	registerTool(s, "automation_get",
		"Retrieve a single automation rule by numeric ID.",
		"automation:read",
		func(_ context.Context, in automationGetInput) (automationRuleResponse, string, error) {
			return automationGet(b, in)
		})

	registerTool(s, "automation_create",
		"Create a new automation rule with trigger event, conditions, and actions.",
		"automation:write",
		func(_ context.Context, in automationCreateInput) (automationRuleResponse, string, error) {
			return automationCreate(b, in)
		})

	registerTool(s, "automation_update",
		"Update fields of an existing automation rule. Only provided fields are changed.",
		"automation:write",
		func(_ context.Context, in automationUpdateInput) (automationRuleResponse, string, error) {
			return automationUpdate(b, in)
		})

	registerTool(s, "automation_delete",
		"Permanently delete an automation rule by numeric ID.",
		"automation:write",
		func(_ context.Context, in automationDeleteInput) (deleteOutput, string, error) {
			return automationDelete(b, in)
		})
}

// ----------------------------------------------------------------------------
// Handlers
// ----------------------------------------------------------------------------

func automationList(b Backend) (automationListOutput, string, error) {
	rules, err := b.ListRules()
	if err != nil {
		return automationListOutput{}, "", err
	}
	out := automationListOutput{Rules: automationRuleResponsesFrom(rules), Total: len(rules)}
	return out, fmt.Sprintf("Listed %d automation rule(s).", len(rules)), nil
}

func automationGet(b Backend, in automationGetInput) (automationRuleResponse, string, error) {
	r, err := b.GetRule(in.ID)
	if err != nil {
		return automationRuleResponse{}, "", err
	}
	return automationRuleResponseFrom(r), fmt.Sprintf("Retrieved automation rule #%d (%s).", r.ID, r.Name), nil
}

func automationCreate(b Backend, in automationCreateInput) (automationRuleResponse, string, error) {
	req := &automation.CreateRuleRequest{
		Name:        in.Name,
		Description: in.Description,
		Enabled:     in.Enabled,
		Event:       in.Event,
		Match:       in.Match,
		Conditions:  in.Conditions,
		Actions:     in.Actions,
		Position:    in.Position,
	}
	r, err := b.CreateRule(req)
	if err != nil {
		return automationRuleResponse{}, "", err
	}
	return automationRuleResponseFrom(r), fmt.Sprintf("Created automation rule #%d (%s).", r.ID, r.Name), nil
}

func automationUpdate(b Backend, in automationUpdateInput) (automationRuleResponse, string, error) {
	req := &automation.UpdateRuleRequest{
		Name:        in.Name,
		Description: in.Description,
		Enabled:     in.Enabled,
		Event:       in.Event,
		Match:       in.Match,
		Conditions:  in.Conditions,
		Actions:     in.Actions,
		Position:    in.Position,
	}
	r, err := b.UpdateRule(in.ID, req)
	if err != nil {
		return automationRuleResponse{}, "", err
	}
	return automationRuleResponseFrom(r), fmt.Sprintf("Updated automation rule #%d (%s).", r.ID, r.Name), nil
}

func automationDelete(b Backend, in automationDeleteInput) (deleteOutput, string, error) {
	if err := b.DeleteRule(in.ID); err != nil {
		return deleteOutput{}, "", err
	}
	return deleteOutput{ID: in.ID, Deleted: true}, fmt.Sprintf("Deleted automation rule #%d.", in.ID), nil
}
