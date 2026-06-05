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

func TestScheduler_Tick_EmitsSLAWarning(t *testing.T) {
	bus := automation.NewBus()

	var gotEvents []automation.Event
	bus.Subscribe(automation.EventSLAWarning, func(e automation.Event) {
		gotEvents = append(gotEvents, e)
	})

	repo := &fakeTicketRepo{overdueIDs: []uint{11, 22}}

	eff := &fakeEffector{}
	exec := automation.NewExecutor(eff)
	store := &fakeRuleStore{rules: nil} // no schedule rules
	loadView := func(uint) (automation.TicketView, error) {
		return automation.TicketView{}, nil
	}
	engine := automation.NewEngine(store, exec, loadView)

	sched := automation.NewScheduler(bus, repo, engine, automation.SchedulerConfig{
		AutoResolveEnabled:  false,
		SilentWindowSeconds: 3600,
	})

	sched.Tick(time.Now())

	// Both overdue tickets should generate an SLA warning event.
	assert.Len(t, gotEvents, 2)
	ids := []uint{gotEvents[0].TicketID, gotEvents[1].TicketID}
	assert.Contains(t, ids, uint(11))
	assert.Contains(t, ids, uint(22))
	// Source must be "automation" so the engine's recursion guard fires.
	for _, ev := range gotEvents {
		assert.Equal(t, "automation", ev.Source)
	}
}

func TestScheduler_Tick_AutoResolve_ClosesTickets(t *testing.T) {
	bus := automation.NewBus()

	repo := &fakeTicketRepo{silentCustomerIDs: []uint{33}}
	eff := &fakeEffector{}
	exec := automation.NewExecutor(eff)
	store := &fakeRuleStore{rules: nil}
	loadView := func(uint) (automation.TicketView, error) {
		return automation.TicketView{}, nil
	}
	engine := automation.NewEngine(store, exec, loadView)

	sched := automation.NewScheduler(bus, repo, engine, automation.SchedulerConfig{
		AutoResolveEnabled:  true,
		SilentWindowSeconds: 3600,
	})

	sched.Tick(time.Now())

	assert.Equal(t, []uint{33}, eff.closes, "auto-resolve should close silent-customer tickets")
}

func TestScheduler_Tick_AutoResolveDisabled_DoesNotClose(t *testing.T) {
	bus := automation.NewBus()
	repo := &fakeTicketRepo{silentCustomerIDs: []uint{33}}
	eff := &fakeEffector{}
	exec := automation.NewExecutor(eff)
	store := &fakeRuleStore{rules: nil}
	loadView := func(uint) (automation.TicketView, error) { return automation.TicketView{}, nil }
	engine := automation.NewEngine(store, exec, loadView)

	sched := automation.NewScheduler(bus, repo, engine, automation.SchedulerConfig{
		AutoResolveEnabled:  false,
		SilentWindowSeconds: 3600,
	})
	sched.Tick(time.Now())
	assert.Empty(t, eff.closes)
}
