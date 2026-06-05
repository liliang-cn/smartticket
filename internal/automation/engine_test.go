package automation_test

import (
	"testing"

	"github.com/company/smartticket/internal/automation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRuleStore returns a fixed list of rules.
type fakeRuleStore struct {
	rules []automation.Rule
	err   error
}

func (s *fakeRuleStore) RulesForEvent(event string) ([]automation.Rule, error) {
	return s.rules, s.err
}

func TestEngine_MatchingRulesRun(t *testing.T) {
	eff := &fakeEffector{}
	exec := automation.NewExecutor(eff)

	// Two enabled rules, both match — both should execute in order.
	rules := []automation.Rule{
		{
			ID:         1,
			Match:      "all",
			Conditions: nil, // no conditions → always matches
			Actions:    []automation.Action{{Type: "add_tag", Params: map[string]any{"tag": "rule1"}}},
		},
		{
			ID:         2,
			Match:      "all",
			Conditions: nil,
			Actions:    []automation.Action{{Type: "add_tag", Params: map[string]any{"tag": "rule2"}}},
		},
	}

	store := &fakeRuleStore{rules: rules}
	loadView := func(uint) (automation.TicketView, error) {
		return automation.TicketView{Status: "open", Priority: "high"}, nil
	}

	engine := automation.NewEngine(store, exec, loadView)
	engine.Handle(automation.Event{
		Type:     automation.EventTicketCreated,
		TicketID: 10,
		Source:   "", // human-initiated
	})

	require.Len(t, eff.tags, 2)
	assert.Equal(t, "rule1", eff.tags[0].tag)
	assert.Equal(t, "rule2", eff.tags[1].tag)
}

func TestEngine_NonMatchingRuleSkipped(t *testing.T) {
	eff := &fakeEffector{}
	exec := automation.NewExecutor(eff)

	rules := []automation.Rule{
		{
			ID:    1,
			Match: "all",
			Conditions: []automation.Condition{
				{Field: "status", Op: "eq", Value: "closed"}, // won't match "open"
			},
			Actions: []automation.Action{{Type: "close"}},
		},
	}

	store := &fakeRuleStore{rules: rules}
	loadView := func(uint) (automation.TicketView, error) {
		return automation.TicketView{Status: "open"}, nil
	}

	engine := automation.NewEngine(store, exec, loadView)
	engine.Handle(automation.Event{Type: automation.EventTicketCreated, TicketID: 1})

	assert.Empty(t, eff.closes, "non-matching rule must not execute")
}

func TestEngine_AutomationSourceGuard(t *testing.T) {
	eff := &fakeEffector{}
	exec := automation.NewExecutor(eff)

	rules := []automation.Rule{
		{ID: 1, Match: "all", Actions: []automation.Action{{Type: "close"}}},
	}

	store := &fakeRuleStore{rules: rules}
	loadView := func(uint) (automation.TicketView, error) {
		return automation.TicketView{Status: "open"}, nil
	}

	engine := automation.NewEngine(store, exec, loadView)
	// Source:"automation" — must be ignored entirely.
	engine.Handle(automation.Event{
		Type:     automation.EventTicketUpdated,
		TicketID: 5,
		Source:   "automation",
	})

	assert.Empty(t, eff.closes, "automation-sourced event must be ignored (recursion guard)")
}

func TestEngine_AISourceGuard(t *testing.T) {
	eff := &fakeEffector{}
	exec := automation.NewExecutor(eff)
	store := &fakeRuleStore{rules: []automation.Rule{
		{ID: 1, Match: "all", Actions: []automation.Action{{Type: "close"}}},
	}}
	loadView := func(uint) (automation.TicketView, error) {
		return automation.TicketView{}, nil
	}
	engine := automation.NewEngine(store, exec, loadView)
	engine.Handle(automation.Event{
		Type:     automation.EventMessageCreated,
		TicketID: 5,
		Source:   "ai",
	})
	assert.Empty(t, eff.closes)
}

func TestEngine_Subscribe_WiresHandlers(t *testing.T) {
	bus := automation.NewBus()
	eff := &fakeEffector{}
	exec := automation.NewExecutor(eff)
	store := &fakeRuleStore{rules: []automation.Rule{
		{ID: 1, Match: "all", Actions: []automation.Action{{Type: "close"}}},
	}}
	loadView := func(uint) (automation.TicketView, error) {
		return automation.TicketView{}, nil
	}
	engine := automation.NewEngine(store, exec, loadView)
	engine.Subscribe(bus)

	// Publishing a human-sourced ticket.created event should trigger the rule.
	bus.Publish(automation.Event{
		Type:     automation.EventTicketCreated,
		TicketID: 99,
		Source:   "",
	})

	assert.Equal(t, []uint{99}, eff.closes)
}
