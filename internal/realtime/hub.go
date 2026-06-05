// Package realtime provides an in-process WebSocket pub/sub hub.
// Clients subscribe to named rooms (e.g. "ticket:42") and receive
// broadcast payloads over buffered channels.
package realtime

import "sync"

const chBufSize = 16

type cmdKind int

const (
	cmdRegister   cmdKind = iota
	cmdUnregister cmdKind = iota
	cmdBroadcast  cmdKind = iota
)

type hubCmd struct {
	kind    cmdKind
	room    string
	ch      chan []byte // register / unregister
	payload []byte     // broadcast
}

// Hub is a goroutine-safe, in-process pub/sub hub keyed by room name.
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[chan []byte]struct{}
	cmds  chan hubCmd
}

// NewHub allocates a Hub ready to be started with Run.
func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[chan []byte]struct{}),
		// Single, ordered command channel guarantees that registers sent
		// before a broadcast are always processed before the broadcast.
		cmds: make(chan hubCmd, 512),
	}
}

// Run processes hub commands until the channel is closed. Call this in a
// dedicated goroutine: go hub.Run().
func (h *Hub) Run() {
	for cmd := range h.cmds {
		switch cmd.kind {
		case cmdRegister:
			h.mu.Lock()
			if h.rooms[cmd.room] == nil {
				h.rooms[cmd.room] = make(map[chan []byte]struct{})
			}
			h.rooms[cmd.room][cmd.ch] = struct{}{}
			h.mu.Unlock()

		case cmdUnregister:
			h.mu.Lock()
			if subs, exists := h.rooms[cmd.room]; exists {
				delete(subs, cmd.ch)
				if len(subs) == 0 {
					delete(h.rooms, cmd.room)
				}
			}
			h.mu.Unlock()

		case cmdBroadcast:
			h.mu.RLock()
			subs := h.rooms[cmd.room]
			// Snapshot subscriber set before releasing the lock.
			targets := make([]chan []byte, 0, len(subs))
			for ch := range subs {
				targets = append(targets, ch)
			}
			h.mu.RUnlock()

			for _, ch := range targets {
				// Non-blocking: drop the message if the subscriber is slow.
				select {
				case ch <- cmd.payload:
				default:
				}
			}
		}
	}
}

// Subscribe registers the caller to receive broadcasts for room and returns a
// buffered channel (size 16) on which payloads will arrive.
func (h *Hub) Subscribe(room string) chan []byte {
	ch := make(chan []byte, chBufSize)
	h.cmds <- hubCmd{kind: cmdRegister, room: room, ch: ch}
	return ch
}

// Unsubscribe removes ch from room. The caller should drain or discard ch afterwards.
func (h *Hub) Unsubscribe(room string, ch chan []byte) {
	h.cmds <- hubCmd{kind: cmdUnregister, room: room, ch: ch}
}

// Broadcast sends payload to every subscriber in room. The send is non-blocking;
// subscribers whose buffer is full silently miss the message.
func (h *Hub) Broadcast(room string, payload []byte) {
	h.cmds <- hubCmd{kind: cmdBroadcast, room: room, payload: payload}
}

// Presence returns the current subscriber count for room.
func (h *Hub) Presence(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room])
}
