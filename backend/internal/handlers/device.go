package handlers

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	deviceTokenRepo repo.DeviceTokenRepository
}

func NewDeviceHandler(deviceTokenRepo repo.DeviceTokenRepository) *DeviceHandler {
	return &DeviceHandler{deviceTokenRepo: deviceTokenRepo}
}

func (h *DeviceHandler) RegisterDevice(c *gin.Context) {
	var req entity.RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	userID := c.GetUint("user_id")

	// Удаляем старый токен, если такой уже есть (переустановка приложения и т.п.)
	_ = h.deviceTokenRepo.DeleteByToken(req.Token)

	token := &entity.DeviceToken{
		UserID:   userID,
		Token:    req.Token,
		Platform: req.Platform,
	}

	if err := h.deviceTokenRepo.Create(token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения токена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Устройство зарегистрировано"})
}

func (h *DeviceHandler) UnregisterDevice(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	if err := h.deviceTokenRepo.DeleteByToken(req.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления токена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Устройство отрегистрировано"})
}
