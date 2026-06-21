package handlers

import (
	entity2 "MessangerMax/internal/entity"
	"MessangerMax/internal/logic"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ReactionHandler struct {
	reactionService logic.ReactionService
}

func NewReactionHandler(reactionService logic.ReactionService) *ReactionHandler {
	return &ReactionHandler{reactionService: reactionService}
}

func (h *ReactionHandler) AddReaction(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("msgId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID сообщения"})
		return
	}

	var req entity2.AddReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	userID := c.GetUint("user_id")
	reaction, err := h.reactionService.AddReaction(userID, uint(messageID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Реакция добавлена",
		"data":    reaction,
	})
}

func (h *ReactionHandler) RemoveReaction(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("msgId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID сообщения"})
		return
	}

	reaction := c.Query("reaction")
	if reaction == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не указана реакция"})
		return
	}

	userID := c.GetUint("user_id")
	if err := h.reactionService.RemoveReaction(userID, uint(messageID), reaction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Реакция удалена",
	})
}

func (h *ReactionHandler) GetReactions(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("msgId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID сообщения"})
		return
	}

	reactions, err := h.reactionService.GetMessageReactions(uint(messageID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения реакций"})
		return
	}

	if reactions == nil {
		reactions = []entity2.ReactionResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": reactions,
	})
}
