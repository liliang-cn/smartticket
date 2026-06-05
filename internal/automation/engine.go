package automation

import (
	"go.uber.org/zap"

	"github.com/company/smartticket/internal/logger"
)

// RuleStore retrieves automation rules from persistent storage.
type RuleStore interface {
	// RulesForEvent returns all enabled rules for the given event type, ordered
	// by Position ascending.
	RulesForEvent(event string) ([]Rule, error)
}

// Rule is the in-memory representation of a stored AutomationRule, with
// Conditions and Actions already parsed into typed structs.
type Rule struct {
	ID         uint
	Match      string      // "all" | "any"
	Conditions []Condition // parsed from AutomationRule.Conditions JSON
	Actions    []Action    // parsed from AutomationRule.Actions JSON
}

// Engine evaluates rules against domain events and runs matching actions.
type Engine struct {
	rules    RuleStore
	exec     *Executor
	loadView func(ticketID uint) (TicketView, error)
}

// NewEngine constructs an Engine. All parameters are required.
func NewEngine(rules RuleStore, exec *Executor, loadView func(uint) (TicketView, error)) *Engine {
	return &Engine{rules: rules, exec: exec, loadView: loadView}
}

// Subscribe wires Handle onto the event bus for the four ticket event types.
func (e *Engine) Subscribe(bus *Bus) {
	bus.Subscribe(EventTicketCreated, e.Handle)
	bus.Subscribe(EventTicketUpdated, e.Handle)
	bus.Subscribe(EventMessageCreated, e.Handle)
	bus.Subscribe(EventSLAWarning, e.Handle)
}

// Handle processes one domain event. It returns immediately when the event was
// produced by the automation engine itself (Source != "") to prevent infinite
// action→event→action loops.
func (e *Engine) Handle(ev Event) {
	// Recursion guard: skip any event that was emitted by automation or AI.
	if ev.Source != "" {
		return
	}

	rules, err := e.rules.RulesForEvent(string(ev.Type))
	if err != nil {
		logger.Warn("automation: failed to load rules",
			zap.String("event", string(ev.Type)),
			zap.Error(err),
		)
		return
	}
	if len(rules) == 0 {
		return
	}

	view, err := e.loadView(ev.TicketID)
	if err != nil {
		logger.Warn("automation: failed to load ticket view",
			zap.Uint("ticket_id", ev.TicketID),
			zap.Error(err),
		)
		return
	}

	for _, r := range rules {
		if !Match(r.Match, r.Conditions, view) {
			continue
		}
		if err := e.exec.Run(ev.TicketID, r.Actions); err != nil {
			// Run already logs individual action errors; this guards against
			// an unexpected error return (shouldn't happen with current impl).
			logger.Warn("automation: rule execution error",
				zap.Uint("rule_id", r.ID),
				zap.Error(err),
			)
		}
	}
}

// CloseTicket exposes the Effector.Close method for the Scheduler's auto-resolve,
// avoiding the need to make exec or eff public.
func (e *Engine) CloseTicket(ticketID uint) error {
	return e.exec.eff.Close(ticketID)
}
