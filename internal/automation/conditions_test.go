package automation_test

import (
	"testing"

	"github.com/company/smartticket/internal/automation"
	"github.com/stretchr/testify/assert"
)

func tv() automation.TicketView {
	return automation.TicketView{
		Status:        "open",
		Priority:      "high",
		Severity:      "major",
		Channel:       "email",
		CustomerEmail: "alice@example.com",
		Tags:          []string{"billing", "urgent"},
	}
}

// --- eq / neq ---

func TestMatch_Eq(t *testing.T) {
	conds := []automation.Condition{{Field: "status", Op: "eq", Value: "open"}}
	assert.True(t, automation.Match("all", conds, tv()))
}

func TestMatch_EqFail(t *testing.T) {
	conds := []automation.Condition{{Field: "status", Op: "eq", Value: "closed"}}
	assert.False(t, automation.Match("all", conds, tv()))
}

func TestMatch_Neq(t *testing.T) {
	conds := []automation.Condition{{Field: "status", Op: "neq", Value: "closed"}}
	assert.True(t, automation.Match("all", conds, tv()))
}

// --- contains ---

func TestMatch_Contains_String(t *testing.T) {
	conds := []automation.Condition{{Field: "customer_email", Op: "contains", Value: "example"}}
	assert.True(t, automation.Match("all", conds, tv()))
}

func TestMatch_Contains_Tags(t *testing.T) {
	conds := []automation.Condition{{Field: "tags", Op: "contains", Value: "billing"}}
	assert.True(t, automation.Match("all", conds, tv()))
}

func TestMatch_Contains_TagsMiss(t *testing.T) {
	conds := []automation.Condition{{Field: "tags", Op: "contains", Value: "payments"}}
	assert.False(t, automation.Match("all", conds, tv()))
}

// --- in ---

func TestMatch_In_String(t *testing.T) {
	conds := []automation.Condition{{Field: "status", Op: "in", Value: []any{"open", "in_progress"}}}
	assert.True(t, automation.Match("all", conds, tv()))
}

func TestMatch_In_Tags(t *testing.T) {
	// tags "in" checks overlap: ticket tags include any of the listed values
	conds := []automation.Condition{{Field: "tags", Op: "in", Value: []any{"billing", "payments"}}}
	assert.True(t, automation.Match("all", conds, tv()))
}

func TestMatch_In_Miss(t *testing.T) {
	conds := []automation.Condition{{Field: "status", Op: "in", Value: []any{"closed", "resolved"}}}
	assert.False(t, automation.Match("all", conds, tv()))
}

// --- gt / lt (priority rank) ---

func TestMatch_Gt_Priority(t *testing.T) {
	// high > medium
	conds := []automation.Condition{{Field: "priority", Op: "gt", Value: "medium"}}
	assert.True(t, automation.Match("all", conds, tv()))
}

func TestMatch_Lt_Priority(t *testing.T) {
	// high < critical
	conds := []automation.Condition{{Field: "priority", Op: "lt", Value: "critical"}}
	assert.True(t, automation.Match("all", conds, tv()))
}

func TestMatch_Gt_Severity(t *testing.T) {
	// major > minor
	conds := []automation.Condition{{Field: "severity", Op: "gt", Value: "minor"}}
	assert.True(t, automation.Match("all", conds, tv()))
}

// --- all / any ---

func TestMatch_AllMode_AllPass(t *testing.T) {
	conds := []automation.Condition{
		{Field: "status", Op: "eq", Value: "open"},
		{Field: "priority", Op: "eq", Value: "high"},
	}
	assert.True(t, automation.Match("all", conds, tv()))
}

func TestMatch_AllMode_OneFails(t *testing.T) {
	conds := []automation.Condition{
		{Field: "status", Op: "eq", Value: "open"},
		{Field: "priority", Op: "eq", Value: "critical"}, // fails
	}
	assert.False(t, automation.Match("all", conds, tv()))
}

func TestMatch_AnyMode_OnePasses(t *testing.T) {
	conds := []automation.Condition{
		{Field: "status", Op: "eq", Value: "closed"},  // fails
		{Field: "priority", Op: "eq", Value: "high"},  // passes
	}
	assert.True(t, automation.Match("any", conds, tv()))
}

func TestMatch_AnyMode_AllFail(t *testing.T) {
	conds := []automation.Condition{
		{Field: "status", Op: "eq", Value: "closed"},
		{Field: "priority", Op: "eq", Value: "low"},
	}
	assert.False(t, automation.Match("any", conds, tv()))
}

func TestMatch_EmptyConds_All(t *testing.T) {
	// No conditions → always matches (unconditional rule)
	assert.True(t, automation.Match("all", nil, tv()))
}

func TestMatch_UnknownField(t *testing.T) {
	// Unknown field → condition evaluates to false (don't panic)
	conds := []automation.Condition{{Field: "does_not_exist", Op: "eq", Value: "foo"}}
	assert.False(t, automation.Match("all", conds, tv()))
}
