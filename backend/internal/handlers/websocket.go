package handlers

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"MessangerMax/utils"
	"log"
	"net/http"
	"sync"

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

	clients   = make(map[uint]*websocket.Conn)
	clientsMu sync.RWMutex
)

type WSHandler struct {
	jwtUtils utils.JWTUtils
}

func NewWSHandler(jwtUtils utils.JWTUtils) *WSHandler {
	return &WSHandler{jwtUtils: jwtUtils}
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
	defer unregisterClient(userID)

	conn.WriteJSON(map[string]interface{}{
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

	if oldConn, exists := clients[userID]; exists {
		oldConn.Close()
	}

	clients[userID] = conn
	log.Printf("Client registered: user_id=%d, total_clients=%d", userID, len(clients))
}

func unregisterClient(userID uint) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	delete(clients, userID)
	log.Printf("Client unregistered: user_id=%d, total_clients=%d", userID, len(clients))
}

func SendToUser(userID uint, message interface{}) error {
	clientsMu.RLock()
	conn, exists := clients[userID]
	clientsMu.RUnlock()

	if !exists {
		return nil
	}

	return conn.WriteJSON(message)
}

func SendToUsers(userIDs []uint, message interface{}) {
	for _, userID := range userIDs {
		go SendToUser(userID, message)
	}
}

func Broadcast(message interface{}) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	for userID, conn := range clients {
		go func(uid uint, c *websocket.Conn) {
			if err := c.WriteJSON(message); err != nil {
				log.Printf("Broadcast error to user %d: %v", uid, err)
			}
		}(userID, conn)
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
