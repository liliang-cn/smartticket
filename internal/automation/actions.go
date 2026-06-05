package automation

import (
	"go.uber.org/zap"

	"github.com/company/smartticket/internal/logger"
)

// Action is one step to execute when a rule fires.
type Action struct {
	Type   string         `json:"type"`   // see Executor.Run for valid types
	Params map[string]any `json:"params"` // type-specific parameters
}

// Effector is the side-effect interface the Executor delegates to.
// A concrete adapter in internal/server wires in the ticket/notification/email/AI
// services without creating an import cycle.
//
// All implementations MUST emit any resulting ticket domain events with
// Source:"automation" so the Engine's recursion guard (ev.Source != "") fires.
type Effector interface {
	// Assign sets ticket assignment. Nil userID/teamID leaves that field unchanged.
	Assign(ticketID uint, userID, teamID *uint) error
	// AddTag appends tag to the ticket's tag list.
	AddTag(ticketID uint, tag string) error
	// SetField updates a single enumerated field: priority|status|severity.
	SetField(ticketID uint, field, value string) error
	// Notify creates an in-app notification for the ticket's assigned agent.
	Notify(ticketID uint, message string) error
	// SendEmail sends an email to the ticket's requester.
	SendEmail(ticketID uint, subject, body string) error
	// Escalate bumps the ticket's priority one level.
	Escalate(ticketID uint) error
	// AISuggest triggers an on-demand AI reply suggestion for the assigned agent.
	AISuggest(ticketID uint) error
	// AIAutoReply posts an AI-generated reply directly to the ticket.
	AIAutoReply(ticketID uint) error
	// Close sets the ticket status to "closed".
	Close(ticketID uint) error
}

// Executor dispatches a list of Actions against a single ticket.
type Executor struct {
	eff Effector
}

// NewExecutor creates an Executor backed by eff.
func NewExecutor(eff Effector) *Executor {
	return &Executor{eff: eff}
}

// Run executes each action in order. An unknown action type is logged and
// skipped — it does not stop execution of subsequent actions or return an error.
func (x *Executor) Run(ticketID uint, actions []Action) error {
	for _, a := range actions {
		if err := x.dispatch(ticketID, a); err != nil {
			// Log but continue — a single action failure must not abort the run.
			logger.Warn("automation: action failed",
				zap.Uint("ticket_id", ticketID),
				zap.String("action_type", a.Type),
				zap.Error(err),
			)
		}
	}
	return nil
}

// dispatch routes a single action to the appropriate Effector method.
func (x *Executor) dispatch(ticketID uint, a Action) error {
	switch a.Type {
	case "assign":
		var userID, teamID *uint
		if v, ok := a.Params["user_id"]; ok {
			if f, ok2 := v.(float64); ok2 {
				u := uint(f)
				userID = &u
			}
		}
		if v, ok := a.Params["team_id"]; ok {
			if f, ok2 := v.(float64); ok2 {
				t := uint(f)
				teamID = &t
			}
		}
		return x.eff.Assign(ticketID, userID, teamID)

	case "add_tag":
		tag, _ := a.Params["tag"].(string)
		return x.eff.AddTag(ticketID, tag)

	case "set_priority":
		val, _ := a.Params["value"].(string)
		return x.eff.SetField(ticketID, "priority", val)

	case "set_status":
		val, _ := a.Params["value"].(string)
		return x.eff.SetField(ticketID, "status", val)

	case "set_severity":
		val, _ := a.Params["value"].(string)
		return x.eff.SetField(ticketID, "severity", val)

	case "notify":
		msg, _ := a.Params["message"].(string)
		return x.eff.Notify(ticketID, msg)

	case "send_email":
		subject, _ := a.Params["subject"].(string)
		body, _ := a.Params["body"].(string)
		return x.eff.SendEmail(ticketID, subject, body)

	case "escalate":
		return x.eff.Escalate(ticketID)

	case "ai_suggest":
		return x.eff.AISuggest(ticketID)

	case "ai_auto_reply":
		return x.eff.AIAutoReply(ticketID)

	case "close":
		return x.eff.Close(ticketID)

	default:
		logger.Warn("automation: unknown action type; skipping",
			zap.Uint("ticket_id", ticketID),
			zap.String("action_type", a.Type),
		)
		return nil
	}
}
