package automation

import "strings"

// TicketView is a normalized, model-free view of a ticket used by the condition
// matcher. The server adapter converts *models.Ticket → TicketView so that
// internal/automation never needs to import internal/models.
type TicketView struct {
	Status        string
	Priority      string
	Severity      string
	Channel       string
	CustomerEmail string
	Tags          []string
}

// Condition represents one predicate in an automation rule.
type Condition struct {
	Field string `json:"field"` // status|priority|severity|channel|customer_email|tags
	Op    string `json:"op"`    // eq|neq|contains|in|gt|lt
	Value any    `json:"value"` // string, []any, or comparable
}

// priorityRank maps priority strings to a comparable integer. Higher = more urgent.
var priorityRank = map[string]int{
	"low":      1,
	"medium":   2,
	"high":     3,
	"critical": 4,
}

// severityRank maps severity strings to a comparable integer. Higher = more severe.
var severityRank = map[string]int{
	"trivial":  1,
	"minor":    2,
	"major":    3,
	"critical": 4,
}

// Match evaluates conditions against tv using matchMode ("all" = AND, "any" = OR).
// An empty conditions slice always returns true (unconditional rule).
func Match(matchMode string, conds []Condition, tv TicketView) bool {
	if len(conds) == 0 {
		return true
	}
	for _, c := range conds {
		result := evalCond(c, tv)
		if matchMode == "any" && result {
			return true
		}
		if matchMode != "any" && !result {
			return false
		}
	}
	return matchMode != "any"
}

// evalCond evaluates a single condition against tv.
func evalCond(c Condition, tv TicketView) bool {
	switch c.Field {
	case "tags":
		return evalTagsCond(c, tv.Tags)
	case "status":
		return evalStringCond(c, tv.Status)
	case "priority":
		return evalRankedCond(c, tv.Priority, priorityRank)
	case "severity":
		return evalRankedCond(c, tv.Severity, severityRank)
	case "channel":
		return evalStringCond(c, tv.Channel)
	case "customer_email":
		return evalStringCond(c, tv.CustomerEmail)
	default:
		return false
	}
}

// evalStringCond handles eq/neq/contains/in for plain string fields.
func evalStringCond(c Condition, fieldVal string) bool {
	switch c.Op {
	case "eq":
		s, _ := c.Value.(string)
		return fieldVal == s
	case "neq":
		s, _ := c.Value.(string)
		return fieldVal != s
	case "contains":
		s, _ := c.Value.(string)
		return strings.Contains(fieldVal, s)
	case "in":
		items, _ := c.Value.([]any)
		for _, item := range items {
			if s, ok := item.(string); ok && fieldVal == s {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// evalRankedCond handles eq/neq/gt/lt/contains/in for ranked enum fields
// (priority, severity).
func evalRankedCond(c Condition, fieldVal string, rank map[string]int) bool {
	switch c.Op {
	case "eq":
		s, _ := c.Value.(string)
		return fieldVal == s
	case "neq":
		s, _ := c.Value.(string)
		return fieldVal != s
	case "gt":
		s, _ := c.Value.(string)
		lv, lok := rank[fieldVal]
		rv, rok := rank[s]
		if !lok || !rok {
			// Unknown rank value — don't fire on garbage input.
			return false
		}
		return lv > rv
	case "lt":
		s, _ := c.Value.(string)
		lv, lok := rank[fieldVal]
		rv, rok := rank[s]
		if !lok || !rok {
			return false
		}
		return lv < rv
	case "contains":
		s, _ := c.Value.(string)
		return strings.Contains(fieldVal, s)
	case "in":
		items, _ := c.Value.([]any)
		for _, item := range items {
			if s, ok := item.(string); ok && fieldVal == s {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// evalTagsCond handles contains/in for the tags field.
// contains: ticket has a tag equal to value.
// in: ticket has at least one tag that appears in the value list.
func evalTagsCond(c Condition, tags []string) bool {
	tagSet := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		tagSet[t] = struct{}{}
	}
	switch c.Op {
	case "contains":
		s, _ := c.Value.(string)
		_, ok := tagSet[s]
		return ok
	case "in":
		items, _ := c.Value.([]any)
		for _, item := range items {
			if s, ok := item.(string); ok {
				if _, has := tagSet[s]; has {
					return true
				}
			}
		}
		return false
	case "eq":
		// eq on tags: ticket has this tag
		s, _ := c.Value.(string)
		_, ok := tagSet[s]
		return ok
	default:
		return false
	}
}
