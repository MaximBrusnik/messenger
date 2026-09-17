package ws

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	userID uint
}

func (c *Client) writeJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.conn.WriteJSON(v)
}

type Hub struct {
	clients   map[uint]*Client
	clientsMu sync.RWMutex
}

func New() *Hub {
	return &Hub{clients: make(map[uint]*Client)}
}

func (h *Hub) Register(userID uint, conn *websocket.Conn) {
	h.clientsMu.Lock()
	if prev, ok := h.clients[userID]; ok {
		_ = prev.conn.Close()
	}
	h.clients[userID] = &Client{conn: conn, userID: userID}
	h.clientsMu.Unlock()
}

func (h *Hub) Unregister(userID uint, conn *websocket.Conn) {
	h.clientsMu.Lock()
	if c, ok := h.clients[userID]; ok && c.conn == conn {
		delete(h.clients, userID)
	}
	h.clientsMu.Unlock()
}

func (h *Hub) IsOnline(userID uint) bool {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

func (h *Hub) SendToUser(userID uint, msg WSMessage) {
	h.clientsMu.RLock()
	c, ok := h.clients[userID]
	h.clientsMu.RUnlock()
	if ok {
		if err := c.writeJSON(msg); err != nil {
			log.Printf("calls hub: write to user %d failed: %v", userID, err)
		}
	}
}

func (h *Hub) SendToUsers(userIDs []uint, msg WSMessage) {
	for _, id := range userIDs {
		id := id
		go h.SendToUser(id, msg)
	}
}
