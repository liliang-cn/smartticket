package automation_test

import (
	"testing"

	"github.com/company/smartticket/internal/automation"
	"github.com/stretchr/testify/assert"
)

func TestBusDispatchCallsSubscribers(t *testing.T) {
	bus := automation.NewBus()
	got := 0
	bus.Subscribe(automation.EventTicketCreated, func(e automation.Event) { got++ })
	bus.Publish(automation.Event{Type: automation.EventTicketCreated, TicketID: 7})
	assert.Equal(t, 1, got)
}

func TestBusPanicRecovery(t *testing.T) {
	bus := automation.NewBus()
	var second int
	bus.Subscribe(automation.EventTicketUpdated, func(e automation.Event) {
		panic("intentional panic")
	})
	bus.Subscribe(automation.EventTicketUpdated, func(e automation.Event) {
		second++
	})
	// Should not panic; second handler must still run.
	assert.NotPanics(t, func() {
		bus.Publish(automation.Event{Type: automation.EventTicketUpdated, TicketID: 1})
	})
	assert.Equal(t, 1, second)
}

func TestBusMultipleEventTypes(t *testing.T) {
	bus := automation.NewBus()
	var created, resolved int
	bus.Subscribe(automation.EventTicketCreated, func(e automation.Event) { created++ })
	bus.Subscribe(automation.EventTicketResolved, func(e automation.Event) { resolved++ })

	bus.Publish(automation.Event{Type: automation.EventTicketCreated, TicketID: 1})
	bus.Publish(automation.Event{Type: automation.EventTicketCreated, TicketID: 2})
	bus.Publish(automation.Event{Type: automation.EventTicketResolved, TicketID: 1})

	assert.Equal(t, 2, created)
	assert.Equal(t, 1, resolved)
}
