package handlers

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"MessangerMax/utils"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *Client) writeJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

type WSHandler struct {
	jwtUtils utils.JWTUtils
	userRepo repo.UserRepository
}

func NewWSHandler(jwtUtils utils.JWTUtils, userRepo repo.UserRepository) *WSHandler {
	return &WSHandler{jwtUtils: jwtUtils, userRepo: userRepo}
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
	defer conn.Close()

	registerClient(userID, conn)
	h.setUserOnline(userID)
	defer unregisterClient(userID)
	defer h.setUserOffline(userID)

	clients[userID].writeJSON(map[string]interface{}{
		"type": "CONNECTED",
		"payload": map[string]interface{}{
			"message": "WebSocket подключен",
			"user_id": userID,
		},
	})

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		if messageType == websocket.TextMessage {
			log.Printf("Received: %s", p)
		}
	}
}

func registerClient(userID uint, conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if old, exists := clients[userID]; exists {
		old.mu.Lock()
		old.conn.Close()
		old.mu.Unlock()
	}

	clients[userID] = &Client{conn: conn}
	log.Printf("Client registered: user_id=%d, total_clients=%d", userID, len(clients))
}

func unregisterClient(userID uint) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	delete(clients, userID)
	log.Printf("Client unregistered: user_id=%d, total_clients=%d", userID, len(clients))
}

func (h *WSHandler) setUserOnline(userID uint) {
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		return
	}
	user.Status = "online"
	h.userRepo.Update(user)

	Broadcast(entity.WSMessage{
		Type: "USER_STATUS",
		Payload: map[string]interface{}{
			"user_id": userID,
			"status":  "online",
		},
	})
}

func (h *WSHandler) setUserOffline(userID uint) {
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		return
	}
	user.Status = "offline"
	user.LastLogin = time.Now()
	h.userRepo.Update(user)

	Broadcast(entity.WSMessage{
		Type: "USER_STATUS",
		Payload: map[string]interface{}{
			"user_id":    userID,
			"status":     "offline",
			"last_login": user.LastLogin,
		},
	})
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
