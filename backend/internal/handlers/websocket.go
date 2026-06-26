package handlers

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"MessangerMax/utils"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
	writeWait  = 10 * time.Second
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	clients   = make(map[uint]*Client)
	clientsMu sync.RWMutex
)

type Client struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	active atomic.Bool
}

func (c *Client) writeJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

type WSHandler struct {
	jwtUtils utils.JWTUtils
}

func NewWSHandler(jwtUtils utils.JWTUtils) *WSHandler {
	return &WSHandler{jwtUtils: jwtUtils}
}

func IsUserOnline(userID uint) bool {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	_, ok := clients[userID]
	return ok
}

func DisconnectUser(userID uint) {
	clientsMu.Lock()
	client, exists := clients[userID]
	if exists {
		delete(clients, userID)
	}
	clientsMu.Unlock()

	if exists && client.active.Load() {
		client.active.Store(false)
		client.mu.Lock()
		client.conn.Close()
		client.mu.Unlock()

		Broadcast(entity.WSMessage{
			Type: "USER_STATUS",
			Payload: map[string]interface{}{
				"user_id": userID,
				"status":  "offline",
			},
		})
	}
}

func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Токен не предоставлен"})
		return
	}

	userID, err := h.jwtUtils.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := registerClient(userID, conn)

	Broadcast(entity.WSMessage{
		Type: "USER_STATUS",
		Payload: map[string]interface{}{
			"user_id": userID,
			"status":  "online",
		},
	})

	SendToUser(userID, map[string]interface{}{
		"type": "CONNECTED",
		"payload": map[string]interface{}{
			"message": "WebSocket подключен",
			"user_id": userID,
		},
	})

	// Cleanup on disconnect: unregister, broadcast offline, close conn
	defer func() {
		unregisterClient(userID, client)
		if client.active.Load() {
			Broadcast(entity.WSMessage{
				Type: "USER_STATUS",
				Payload: map[string]interface{}{
					"user_id": userID,
					"status":  "offline",
				},
			})
		}
		conn.Close()
	}()

	// Ping/pong heartbeat
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}()

	// Read loop
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		log.Printf("Received: %s", p)
	}
}

func registerClient(userID uint, conn *websocket.Conn) *Client {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	client := &Client{conn: conn}
	client.active.Store(true)

	if old, exists := clients[userID]; exists {
		old.active.Store(false)
		old.mu.Lock()
		old.conn.Close()
		old.mu.Unlock()
	}

	clients[userID] = client
	log.Printf("Client registered: user_id=%d, total_clients=%d", userID, len(clients))
	return client
}

func unregisterClient(userID uint, client *Client) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if current, exists := clients[userID]; exists && current == client {
		delete(clients, userID)
		log.Printf("Client unregistered: user_id=%d, total_clients=%d", userID, len(clients))
	}
}

func SendToUser(userID uint, message interface{}) error {
	clientsMu.RLock()
	client, exists := clients[userID]
	clientsMu.RUnlock()

	if !exists {
		return nil
	}

	return client.writeJSON(message)
}

func SendToUsers(userIDs []uint, message interface{}) {
	for _, userID := range userIDs {
		go SendToUser(userID, message)
	}
}

func Broadcast(message interface{}) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	for userID, client := range clients {
		go func(uid uint, c *Client) {
			if err := c.writeJSON(message); err != nil {
				log.Printf("Broadcast error to user %d: %v", uid, err)
			}
		}(userID, client)
	}
}

// WSNotifier — реализация logic.Notifier для рассылки через WebSocket
type WSNotifier struct {
	chatRepo repo.ChatRepository
}

func NewWSNotifier(chatRepo repo.ChatRepository) *WSNotifier {
	return &WSNotifier{chatRepo: chatRepo}
}

func (n *WSNotifier) SendNewMessage(chatID uint, message *entity.MessageResponse) {
	participants, err := n.chatRepo.GetParticipantIDs(chatID)
	if err != nil {
		log.Printf("WSNotifier: failed to get participants for chat %d: %v", chatID, err)
		return
	}

	payload := map[string]interface{}{
		"type": "NEW_MESSAGE",
		"payload": map[string]interface{}{
			"message": message,
		},
	}

	SendToUsers(participants, payload)
}

func (n *WSNotifier) SendMessageEdited(chatID uint, message *entity.MessageResponse) {
	participants, err := n.chatRepo.GetParticipantIDs(chatID)
	if err != nil {
		log.Printf("WSNotifier: failed to get participants for chat %d: %v", chatID, err)
		return
	}

	payload := map[string]interface{}{
		"type": "MESSAGE_EDITED",
		"payload": map[string]interface{}{
			"message": message,
		},
	}

	SendToUsers(participants, payload)
}

func (n *WSNotifier) SendReactionAdded(chatID uint, messageID uint, reaction *entity.ReactionResponse) {
	participants, err := n.chatRepo.GetParticipantIDs(chatID)
	if err != nil {
		log.Printf("WSNotifier: failed to get participants for chat %d: %v", chatID, err)
		return
	}

	payload := map[string]interface{}{
		"type": "REACTION_ADDED",
		"payload": map[string]interface{}{
			"reaction": reaction,
		},
	}

	SendToUsers(participants, payload)
}

func (n *WSNotifier) SendReactionRemoved(chatID uint, messageID uint, userID uint, reaction string) {
	participants, err := n.chatRepo.GetParticipantIDs(chatID)
	if err != nil {
		log.Printf("WSNotifier: failed to get participants for chat %d: %v", chatID, err)
		return
	}

	payload := map[string]interface{}{
		"type": "REACTION_REMOVED",
		"payload": map[string]interface{}{
			"message_id": messageID,
			"user_id":    userID,
			"reaction":   reaction,
		},
	}

	SendToUsers(participants, payload)
}

func (n *WSNotifier) SendMessageDeleted(chatID uint, messageID uint) {
	participants, err := n.chatRepo.GetParticipantIDs(chatID)
	if err != nil {
		log.Printf("WSNotifier: failed to get participants for chat %d: %v", chatID, err)
		return
	}

	payload := map[string]interface{}{
		"type": "MESSAGE_DELETED",
		"payload": map[string]interface{}{
			"message_id": messageID,
			"chat_id":    chatID,
		},
	}

	SendToUsers(participants, payload)
}

func (n *WSNotifier) SendChatDeleted(chatID uint) {
	Broadcast(map[string]interface{}{
		"type": "CHAT_DELETED",
		"payload": map[string]interface{}{
			"chat_id": chatID,
		},
	})
}

func (n *WSNotifier) SendMessagePinned(chatID uint, message *entity.MessageResponse) {
	participants, err := n.chatRepo.GetParticipantIDs(chatID)
	if err != nil {
		log.Printf("WSNotifier: failed to get participants for chat %d: %v", chatID, err)
		return
	}

	payload := map[string]interface{}{
		"type": "MESSAGE_PINNED",
		"payload": map[string]interface{}{
			"chat_id": chatID,
			"message": message,
		},
	}

	SendToUsers(participants, payload)
}

func (n *WSNotifier) SendMessageUnpinned(chatID uint) {
	participants, err := n.chatRepo.GetParticipantIDs(chatID)
	if err != nil {
		log.Printf("WSNotifier: failed to get participants for chat %d: %v", chatID, err)
		return
	}

	payload := map[string]interface{}{
		"type": "MESSAGE_UNPINNED",
		"payload": map[string]interface{}{
			"chat_id": chatID,
		},
	}

	SendToUsers(participants, payload)
}

func (n *WSNotifier) SendMessagesRead(chatID uint, messageIDs []uint, readByUserID uint) {
	participants, err := n.chatRepo.GetParticipantIDs(chatID)
	if err != nil {
		log.Printf("WSNotifier: failed to get participants for chat %d: %v", chatID, err)
		return
	}

	payload := map[string]interface{}{
		"type": "MESSAGES_READ",
		"payload": map[string]interface{}{
			"chat_id":     chatID,
			"message_ids": messageIDs,
			"read_by":     readByUserID,
		},
	}

	SendToUsers(participants, payload)
}
