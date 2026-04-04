package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub рассылает JSON всем подключённым клиентам дашборда.
type Hub struct {
	log *slog.Logger

	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func NewHub(log *slog.Logger) *Hub {
	if log == nil {
		log = slog.Default()
	}
	return &Hub{log: log, clients: make(map[*websocket.Conn]struct{})}
}

func (h *Hub) Register(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	n := len(h.clients)
	h.mu.Unlock()
	h.log.Info("ws client connected", "clients", n)
}

func (h *Hub) Unregister(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	n := len(h.clients)
	h.mu.Unlock()
	_ = c.Close()
	h.log.Info("ws client disconnected", "clients", n)
}

// Broadcast отправляет одно и то же сообщение всем (копия на клиента).
func (h *Hub) Broadcast(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		h.log.Error("ws marshal", "err", err)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
			h.log.Warn("ws write", "err", err)
			_ = c.Close()
			delete(h.clients, c)
		}
	}
}
