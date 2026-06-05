package realtime_test

import (
	"testing"
	"time"

	"github.com/company/smartticket/internal/realtime"
	"github.com/stretchr/testify/assert"
)

func TestHubBroadcastReachesRoomMembers(t *testing.T) {
	h := realtime.NewHub()
	go h.Run()
	a := h.Subscribe("ticket:1")
	b := h.Subscribe("ticket:1")
	other := h.Subscribe("ticket:2")
	h.Broadcast("ticket:1", []byte(`{"type":"message"}`))
	assert.Equal(t, []byte(`{"type":"message"}`), <-a)
	assert.Equal(t, []byte(`{"type":"message"}`), <-b)
	select {
	case <-other:
		t.Fatal("ticket:2 must not receive ticket:1 broadcast")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHubPresence(t *testing.T) {
	h := realtime.NewHub()
	go h.Run()

	assert.Equal(t, 0, h.Presence("ticket:42"))
	ch1 := h.Subscribe("ticket:42")
	ch2 := h.Subscribe("ticket:42")
	// Give the hub goroutine time to process the registers.
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 2, h.Presence("ticket:42"))

	h.Unsubscribe("ticket:42", ch1)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, h.Presence("ticket:42"))

	h.Unsubscribe("ticket:42", ch2)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, h.Presence("ticket:42"))
}
