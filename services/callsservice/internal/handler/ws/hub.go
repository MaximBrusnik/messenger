package ws

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage is the JSON envelope exchanged through the call signaling
// WebSocket. Type carries the semantic (CALL_INVITE, CALL_ACCEPT ...),
// payload is the message body routed to one or both peers.
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

// Hub tracks connected clients by user id. One connection per user
// (last-wins). Messages are relayed to a target user (and optionally a
// call room shared by two users).
type Hub struct {
	clients   map[uint]*Client
	clientsMu sync.RWMutex
}

func New() *Hub {
	return &Hub{clients: make(map[uint]*Client)}
}

// Register adds a connection, closing any previous one for the user.
func (h *Hub) Register(userID uint, conn *websocket.Conn) {
	h.clientsMu.Lock()
	if prev, ok := h.clients[userID]; ok {
		_ = prev.conn.Close()
	}
	h.clients[userID] = &Client{conn: conn, userID: userID}
	h.clientsMu.Unlock()
}

// Unregister removes the connection if it is still current.
func (h *Hub) Unregister(userID uint, conn *websocket.Conn) {
	h.clientsMu.Lock()
	if c, ok := h.clients[userID]; ok && c.conn == conn {
		delete(h.clients, userID)
	}
	h.clientsMu.Unlock()
}

// IsOnline reports whether the user has a live signaling connection.
func (h *Hub) IsOnline(userID uint) bool {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// SendToUser delivers a message to a single connected user.
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

// SendToUsers delivers a message to a set of users concurrently.
func (h *Hub) SendToUsers(userIDs []uint, msg WSMessage) {
	for _, id := range userIDs {
		id := id
		go h.SendToUser(id, msg)
	}
}
