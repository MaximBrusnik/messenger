package handlers

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/logic"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type ChatHandler struct {
	chatService logic.ChatService
}

func NewChatHandler(chatService logic.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// @Summary Создать чат
// @Description Создание нового чата с пользователем
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateChatRequest true "Данные для создания чата"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/chats [post]
func (h *ChatHandler) CreateChat(c *gin.Context) {
	var req entity.CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	userID := c.GetUint("user_id")
	chat, err := h.chatService.CreateChat(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Чат создан",
		"data":    chat,
	})
}

// @Summary Получить список чатов
// @Description Получение всех чатов пользователя
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/chats [get]
func (h *ChatHandler) GetChats(c *gin.Context) {
	userID := c.GetUint("user_id")
	chats, err := h.chatService.GetUserChats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения чатов"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": chats,
	})
}

// @Summary Получить информацию о чате
// @Description Получение информации о конкретном чате
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID чата"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/chats/{id} [get]
func (h *ChatHandler) GetChat(c *gin.Context) {
	chatID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID чата"})
		return
	}

	userID := c.GetUint("user_id")
	chat, err := h.chatService.GetChatByID(uint(chatID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": chat,
	})
}

// @Summary Отправить сообщение
// @Description Отправка сообщения в чат
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID чата"
// @Param request body models.SendMessageRequest true "Текст сообщения"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/chats/{id}/messages [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	chatID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID чата"})
		return
	}

	var req entity.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	userID := c.GetUint("user_id")
	message, err := h.chatService.SendMessage(userID, uint(chatID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Сообщение отправлено",
		"data":    message,
	})
}

// @Summary Получить сообщения чата
// @Description Получение сообщений из чата
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID чата"
// @Param limit query int false "Лимит сообщений"
// @Param offset query int false "Смещение"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/chats/{id}/messages [get]
func (h *ChatHandler) GetMessages(c *gin.Context) {
	chatID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID чата"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	userID := c.GetUint("user_id")
	messages, err := h.chatService.GetChatMessages(uint(chatID), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": messages,
	})
}

// @Summary Отметить как прочитанное
// @Description Отметить все сообщения в чате как прочитанные
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID чата"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/chats/{id}/read [post]
func (h *ChatHandler) MarkAsRead(c *gin.Context) {
	chatID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID чата"})
		return
	}

	userID := c.GetUint("user_id")
	if err := h.chatService.MarkAsRead(uint(chatID), userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Сообщения отмечены как прочитанные",
	})
}
