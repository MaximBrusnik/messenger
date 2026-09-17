package ws

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"messengermax/pkg/redis"
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
	redis     *redis.Client
}

func New(redis *redis.Client) *Hub {
	return &Hub{clients: make(map[uint]*Client), redis: redis}
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

func (h *Hub) Disconnect(userID uint) *websocket.Conn {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	if c, ok := h.clients[userID]; ok {
		delete(h.clients, userID)
		return c.conn
	}
	return nil
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
			log.Printf("hub: write to user %d failed: %v", userID, err)
		}
	}
}

func (h *Hub) SendToUsers(userIDs []uint, msg WSMessage) {
	for _, id := range userIDs {
		id := id
		go h.SendToUser(id, msg)
	}
}

func (h *Hub) Broadcast(msg WSMessage) {
	h.clientsMu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		clients = append(clients, c)
	}
	h.clientsMu.RUnlock()
	for _, c := range clients {
		c := c
		go func() {
			if err := c.writeJSON(msg); err != nil {
				log.Printf("hub: broadcast write failed: %v", err)
			}
		}()
	}
}

func (h *Hub) Count() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}
