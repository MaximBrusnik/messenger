package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"MessangerMax/internal/entity"
	"MessangerMax/internal/logic"
)

type AuthHandler struct {
	authService logic.AuthService
}

func NewAuthHandler(authService logic.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req entity.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Неверные данные",
			"details": err.Error(),
		})
		return
	}

	user, err := h.authService.Register(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Пользователь успешно создан",
		"data":    user,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req entity.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверные данные",
		})
		return
	}

	token, user, err := h.authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Установка токена в куки (опционально)
	c.SetCookie("access_token", token, 3600*24, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Успешный вход",
		"token":   token,
		"data":    user,
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не авторизован",
		})
		return
	}

	user, err := h.authService.GetUserProfile(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Очистка куки
	c.SetCookie("access_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Успешный выход",
	})
}
