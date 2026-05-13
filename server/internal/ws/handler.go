package ws

import (
	"net/http"

	"github.com/coder/websocket"
)

// Handler upgrades HTTP connections to WebSocket and registers them with the Hub.
type Handler struct {
	hub *Hub
}

// NewHandler creates an HTTP handler backed by the given Hub.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Allow any origin in dev; restrict via OriginPatterns in production.
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	h.hub.Register(conn)
}
