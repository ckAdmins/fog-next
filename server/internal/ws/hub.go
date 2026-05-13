// Package ws implements a broadcast WebSocket hub for live server-side events.
// Clients connect to /api/v1/ws and receive JSON event messages whenever
// tasks, hosts, or other resources change state.
package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// EventType identifies the kind of event being broadcast.
type EventType string

const (
	EventTaskProgress EventType = "task.progress"
	EventTaskCreated  EventType = "task.created"
	EventTaskComplete EventType = "task.complete"
	EventTaskCanceled EventType = "task.canceled"
	EventHostOnline   EventType = "host.online"
	EventHostOffline  EventType = "host.offline"
	EventAgentLog     EventType = "agent.log"
)

// Event is the JSON payload sent to connected clients.
type Event struct {
	Type    EventType `json:"type"`
	Payload any       `json:"payload"`
	At      time.Time `json:"at"`
}

// client wraps a single WS connection.
type client struct {
	conn   *websocket.Conn
	send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc
}

// Hub manages all connected WebSocket clients and routes broadcasts.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

// New creates an idle Hub. Call Run to start the background cleanup loop.
func New() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

// Register adds a connection to the hub and starts its write pump.
// The caller should call Register inside the HTTP handler after upgrading.
// The returned context is cancelled when the client disconnects.
func (h *Hub) Register(conn *websocket.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &client{conn: conn, send: make(chan []byte, 64), ctx: ctx, cancel: cancel}

	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	go c.writePump()
	go func() {
		// Read loop: keeps the connection alive and detects close/error.
		for {
			_, _, err := conn.Read(ctx)
			if err != nil {
				break
			}
		}
		h.unregister(c)
	}()
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		c.cancel()
		close(c.send)
	}
	h.mu.Unlock()
}

// Broadcast sends an event to all connected clients.
// Slow clients whose buffers are full are dropped.
func (h *Hub) Broadcast(evt Event) {
	evt.At = time.Now()
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Error("ws: marshal event", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			// Buffer full — drop this client.
			go h.unregister(c)
		}
	}
}

// ClientCount returns the number of currently connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (c *client) writePump() {
	defer func() {
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
			err := c.conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
