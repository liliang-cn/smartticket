package realtime

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow if Origin host matches request host, or if Origin is empty
	// (e.g. native WebSocket clients that omit the header).
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Parse origin to extract host.
		// We accept if the origin's host equals the request's host.
		originURL, err := parseOriginHost(origin)
		if err != nil {
			return false
		}
		return originURL == r.Host
	},
}

// publicUpgrader accepts WebSocket connections from any origin. Used by the
// customer-facing widget endpoint where cross-origin connections are expected
// and safe (all sensitive operations are scoped by the conversation token).
var publicUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

// parseOriginHost extracts the host (host:port) from an Origin header value.
func parseOriginHost(origin string) (string, error) {
	// Origin format: scheme://host[:port]
	// Strip scheme.
	for i := 0; i < len(origin)-2; i++ {
		if origin[i] == ':' && origin[i+1] == '/' && origin[i+2] == '/' {
			return origin[i+3:], nil
		}
	}
	return origin, nil
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// ServeWS upgrades the HTTP connection to a WebSocket, subscribes the connection
// to room on hub, pumps hub→socket, and re-broadcasts any inbound frames from
// the client to the same room (used for typing indicators / presence events).
//
// This function blocks until the connection is closed; it should be called from
// a Gin handler (which runs in its own goroutine per request).
func ServeWS(hub *Hub, room string, w http.ResponseWriter, r *http.Request) {
	serveWS(upgrader, hub, room, w, r)
}

// ServeWSPublic is identical to ServeWS but uses a cross-origin-permissive
// upgrader. Use this for public widget WebSocket endpoints where the client
// originates from a different domain than the server.
func ServeWSPublic(hub *Hub, room string, w http.ResponseWriter, r *http.Request) {
	serveWS(publicUpgrader, hub, room, w, r)
}

func serveWS(up websocket.Upgrader, hub *Hub, room string, w http.ResponseWriter, r *http.Request) {
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade writes the error response itself; nothing more to do.
		return
	}
	defer conn.Close()

	ch := hub.Subscribe(room)
	defer hub.Unsubscribe(room, ch)

	// Drain ch on exit so the hub never blocks on a closed client.
	defer func() {
		for {
			select {
			case <-ch:
			default:
				return
			}
		}
	}()

	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// outbound: hub → socket
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case payload, ok := <-ch:
				if !ok {
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// inbound: socket → hub (typing / presence frames are re-broadcast to the same room)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		hub.Broadcast(room, msg)
	}
}
