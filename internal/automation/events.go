// Package automation provides the synchronous domain event bus used to decouple
// side-effects (notifications, automation rules, AI triggers) from the core
// ticket service without introducing import cycles.
package automation

import "sync"

// EventType identifies the kind of domain event.
type EventType string

const (
	EventTicketCreated  EventType = "ticket.created"
	EventTicketUpdated  EventType = "ticket.updated"
	EventMessageCreated EventType = "message.created"
	EventSLAWarning     EventType = "ticket.sla_warning"
	EventTicketResolved EventType = "ticket.resolved"
)

// Event carries all contextual data for a domain event.
type Event struct {
	// Type is the event kind.
	Type EventType
	// TicketID is the primary ticket this event concerns.
	TicketID uint
	// ActorID is the user that triggered the event (0 = system).
	ActorID uint
	// Source distinguishes human-initiated events from automated ones.
	// Use "" for human, "automation" or "ai" for system-generated events to
	// prevent automation loops.
	Source string
	// Payload carries arbitrary structured data specific to the event type.
	Payload map[string]any
}

// Handler is a function that processes a domain event.
type Handler func(Event)

// Bus is a synchronous, goroutine-safe publish/subscribe event bus.
type Bus struct {
	mu   sync.RWMutex
	subs map[EventType][]Handler
}

// NewBus allocates an empty Bus.
func NewBus() *Bus {
	return &Bus{
		subs: make(map[EventType][]Handler),
	}
}

// Subscribe registers h to be called whenever an event of type t is published.
// Multiple handlers for the same event type are called in registration order.
func (b *Bus) Subscribe(t EventType, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[t] = append(b.subs[t], h)
}

// Publish dispatches e to every handler registered for e.Type.
// Execution is synchronous (in the caller's goroutine).
// A panic inside a handler is recovered so that one bad handler cannot prevent
// subsequent handlers from running.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.subs[e.Type]))
	copy(handlers, b.subs[e.Type])
	b.mu.RUnlock()

	for _, h := range handlers {
		safeCall(h, e)
	}
}

// safeCall invokes h(e) and recovers from any panic so subsequent handlers
// in the same Publish call can still run.
func safeCall(h Handler, e Event) {
	defer func() { recover() }() //nolint:errcheck
	h(e)
}
