package handlers

import (
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
			return true // В продакшене нужно ограничить
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
	// Получаем токен из query параметра
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Токен не предоставлен"})
		return
	}

	// Валидируем токен
	userID, err := h.jwtUtils.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен"})
		return
	}

	// Обновляем до WebSocket соединения
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Регистрируем клиента
	registerClient(userID, conn)
	defer unregisterClient(userID)

	// Отправляем приветственное сообщение
	conn.WriteJSON(map[string]interface{}{
		"type": "CONNECTED",
		"payload": map[string]interface{}{
			"message": "WebSocket подключен",
			"user_id": userID,
		},
	})

	// Обрабатываем сообщения
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		if messageType == websocket.TextMessage {
			log.Printf("Received: %s", p)
			// Можно обрабатывать входящие сообщения от клиента
		}
	}
}

// Регистрация клиента
func registerClient(userID uint, conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	// Закрываем старое соединение если есть
	if oldConn, exists := clients[userID]; exists {
		oldConn.Close()
	}

	clients[userID] = conn
	log.Printf("Client registered: user_id=%d, total_clients=%d", userID, len(clients))
}

// Удаление клиента
func unregisterClient(userID uint) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	delete(clients, userID)
	log.Printf("Client unregistered: user_id=%d, total_clients=%d", userID, len(clients))
}

// Отправка сообщения пользователю
func SendToUser(userID uint, message interface{}) error {
	clientsMu.RLock()
	conn, exists := clients[userID]
	clientsMu.RUnlock()

	if !exists {
		return nil // Пользователь не подключен
	}

	return conn.WriteJSON(message)
}

// Отправка сообщения нескольким пользователям
func SendToUsers(userIDs []uint, message interface{}) {
	for _, userID := range userIDs {
		go SendToUser(userID, message)
	}
}

// Broadcast сообщения всем подключенным пользователям
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
