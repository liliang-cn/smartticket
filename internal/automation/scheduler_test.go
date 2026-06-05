package automation_test

import (
	"testing"
	"time"

	"github.com/company/smartticket/internal/automation"
	"github.com/stretchr/testify/assert"
)

// fakeTicketRepo implements automation.TicketRepo for scheduler tests.
type fakeTicketRepo struct {
	overdueIDs        []uint
	silentCustomerIDs []uint
}

func (r *fakeTicketRepo) OverdueTicketIDs(now time.Time) ([]uint, error) {
	return r.overdueIDs, nil
}
func (r *fakeTicketRepo) SilentCustomerTicketIDs(now time.Time, windowSeconds int64) ([]uint, error) {
	return r.silentCustomerIDs, nil
}

// fakeSettings implements automation.AutoResolveSettingsReader for scheduler tests.
type fakeSettings struct{ enabled bool }

func (f *fakeSettings) AutoResolveEnabled() bool { return f.enabled }

// newTestEngine builds a minimal Engine using the provided rule store.
func newTestEngine(store automation.RuleStore, eff *fakeEffector) *automation.Engine {
	exec := automation.NewExecutor(eff)
	loadView := func(uint) (automation.TicketView, error) {
		return automation.TicketView{Status: "open"}, nil
	}
	return automation.NewEngine(store, exec, loadView)
}

func TestScheduler_Tick_EmitsSLAWarning(t *testing.T) {
	bus := automation.NewBus()

	var gotEvents []automation.Event
	bus.Subscribe(automation.EventSLAWarning, func(e automation.Event) {
		gotEvents = append(gotEvents, e)
	})

	repo := &fakeTicketRepo{overdueIDs: []uint{11, 22}}
	engine := newTestEngine(&fakeRuleStore{rules: nil}, &fakeEffector{})

	sched := automation.NewScheduler(bus, repo, engine, automation.SchedulerConfig{
		SilentWindowSeconds: 3600,
	})

	sched.Tick(time.Now())

	// Both overdue tickets should generate an SLA warning event.
	assert.Len(t, gotEvents, 2)
	ids := []uint{gotEvents[0].TicketID, gotEvents[1].TicketID}
	assert.Contains(t, ids, uint(11))
	assert.Contains(t, ids, uint(22))
	// Source must be "" so the engine's recursion guard does NOT block it and
	// admin rules with event="ticket.sla_warning" can fire.
	for _, ev := range gotEvents {
		assert.Equal(t, "", ev.Source, "sla_warning event must have empty Source to pass engine recursion guard")
	}
}

// fakeSLAWarnRuleStore returns rules only for the ticket.sla_warning event,
// simulating an admin rule that fires specifically on that trigger.
type fakeSLAWarnRuleStore struct {
	rules []automation.Rule
}

func (s *fakeSLAWarnRuleStore) RulesForEvent(event string) ([]automation.Rule, error) {
	if event == string(automation.EventSLAWarning) {
		return s.rules, nil
	}
	return nil, nil // no rules for other events (e.g. internal "schedule")
}

// TestScheduler_Tick_SLAWarning_RuleFires proves that an enabled ticket.sla_warning
// rule's action executes when the scheduler tick finds an overdue ticket.
func TestScheduler_Tick_SLAWarning_RuleFires(t *testing.T) {
	bus := automation.NewBus()

	eff := &fakeEffector{}
	store := &fakeSLAWarnRuleStore{
		rules: []automation.Rule{
			{
				ID:    1,
				Match: "all",
				// No conditions → fires unconditionally on the event.
				Actions: []automation.Action{{Type: "add_tag", Params: map[string]any{"tag": "sla-breached"}}},
			},
		},
	}
	exec := automation.NewExecutor(eff)
	loadView := func(uint) (automation.TicketView, error) {
		return automation.TicketView{Status: "open"}, nil
	}
	engine := automation.NewEngine(store, exec, loadView)
	engine.Subscribe(bus) // must be subscribed to EventSLAWarning via bus

	repo := &fakeTicketRepo{overdueIDs: []uint{55}}

	sched := automation.NewScheduler(bus, repo, engine, automation.SchedulerConfig{
		SilentWindowSeconds: 3600,
	})

	sched.Tick(time.Now())

	// The sla_warning rule must have run its add_tag action exactly once.
	assert.Len(t, eff.tags, 1, "ticket.sla_warning rule action must fire for overdue ticket")
	assert.Equal(t, uint(55), eff.tags[0].ticketID)
	assert.Equal(t, "sla-breached", eff.tags[0].tag)
}

func TestScheduler_Tick_AutoResolve_ClosesTickets(t *testing.T) {
	bus := automation.NewBus()

	repo := &fakeTicketRepo{silentCustomerIDs: []uint{33}}
	eff := &fakeEffector{}
	engine := newTestEngine(&fakeRuleStore{rules: nil}, eff)

	sched := automation.NewScheduler(bus, repo, engine, automation.SchedulerConfig{
		SilentWindowSeconds: 3600,
		Settings:            &fakeSettings{enabled: true},
	})

	sched.Tick(time.Now())

	assert.Equal(t, []uint{33}, eff.closes, "auto-resolve should close silent-customer tickets")
}

func TestScheduler_Tick_AutoResolveDisabled_DoesNotClose(t *testing.T) {
	bus := automation.NewBus()
	repo := &fakeTicketRepo{silentCustomerIDs: []uint{33}}
	eff := &fakeEffector{}
	engine := newTestEngine(&fakeRuleStore{rules: nil}, eff)

	sched := automation.NewScheduler(bus, repo, engine, automation.SchedulerConfig{
		SilentWindowSeconds: 3600,
		Settings:            &fakeSettings{enabled: false},
	})
	sched.Tick(time.Now())
	assert.Empty(t, eff.closes)
}

func TestScheduler_Tick_AutoResolve_NilSettings_DoesNotClose(t *testing.T) {
	// nil Settings reader → auto-resolve must be treated as disabled.
	bus := automation.NewBus()
	repo := &fakeTicketRepo{silentCustomerIDs: []uint{33}}
	eff := &fakeEffector{}
	engine := newTestEngine(&fakeRuleStore{rules: nil}, eff)

	sched := automation.NewScheduler(bus, repo, engine, automation.SchedulerConfig{
		SilentWindowSeconds: 3600,
		Settings:            nil,
	})
	sched.Tick(time.Now())
	assert.Empty(t, eff.closes)
}
