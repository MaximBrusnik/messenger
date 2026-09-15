package hub

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"messengermax/pkg/redis"
)

// WSMessage mirrors the JSON envelope the frontend expects.
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Client is a single WebSocket connection.
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

// Hub tracks all connected clients in memory. Online status is also
// persisted to Redis with a TTL so other services can query it.
type Hub struct {
	clients   map[uint]*Client
	clientsMu sync.RWMutex
	redis     *redis.Client
}

func New(redis *redis.Client) *Hub {
	return &Hub{clients: make(map[uint]*Client), redis: redis}
}

// Register adds a connection, closing any existing connection for the user.
func (h *Hub) Register(userID uint, conn *websocket.Conn) {
	h.clientsMu.Lock()
	// close the previous connection if present
	if prev, ok := h.clients[userID]; ok {
		_ = prev.conn.Close()
	}
	h.clients[userID] = &Client{conn: conn, userID: userID}
	h.clientsMu.Unlock()
}

// Unregister removes the connection if it is still the current one.
func (h *Hub) Unregister(userID uint, conn *websocket.Conn) {
	h.clientsMu.Lock()
	if c, ok := h.clients[userID]; ok && c.conn == conn {
		delete(h.clients, userID)
	}
	h.clientsMu.Unlock()
}

// Disconnect removes and returns the current connection for a user without
// requiring the caller to hold it, enabling programmatic teardown.
func (h *Hub) Disconnect(userID uint) *websocket.Conn {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	if c, ok := h.clients[userID]; ok {
		delete(h.clients, userID)
		return c.conn
	}
	return nil
}

// IsOnline reports whether the user has a live connection.
func (h *Hub) IsOnline(userID uint) bool {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// SendToUser sends a message to a single connected user.
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

// SendToUsers sends a message to many users concurrently.
func (h *Hub) SendToUsers(userIDs []uint, msg WSMessage) {
	for _, id := range userIDs {
		id := id
		go h.SendToUser(id, msg)
	}
}

// Broadcast sends a message to every connected user.
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

// Count returns the number of live connections.
func (h *Hub) Count() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}
