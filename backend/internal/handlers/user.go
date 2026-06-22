package handlers

import (
	"MessangerMax/internal/entity"
	logic2 "MessangerMax/internal/logic"
	"github.com/gin-gonic/gin"
	"net/http"
)

type UserHandler struct {
	userService logic2.UserService
	authService logic2.AuthService
}

func NewUserHandler(userService logic2.UserService, authService logic2.AuthService) *UserHandler {
	return &UserHandler{
		userService: userService,
		authService: authService,
	}
}

func (h *UserHandler) GetAllUsers(c *gin.Context) {
	userID := c.GetUint("user_id")
	users, err := h.userService.GetAllUsers(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения пользователей"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": users,
	})
}

func (h *UserHandler) SearchUsers(c *gin.Context) {
	var req entity.SearchUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные параметры поиска"})
		return
	}

	userID := c.GetUint("user_id")
	users, err := h.userService.SearchUsers(req.Query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка поиска"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": users,
	})
}

func (h *UserHandler) GetContacts(c *gin.Context) {
	userID := c.GetUint("user_id")
	contacts, err := h.userService.GetContacts(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения контактов"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": contacts,
	})
}

func (h *UserHandler) AddContact(c *gin.Context) {
	var req entity.AddContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	userID := c.GetUint("user_id")
	if err := h.userService.AddContact(userID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Контакт добавлен",
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req entity.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	userID := c.GetUint("user_id")
	user, err := h.userService.UpdateProfile(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Профиль обновлен",
		"data":    user,
	})
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req entity.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	userID := c.GetUint("user_id")
	if err := h.userService.ChangePassword(userID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Пароль успешно изменен",
	})
}

func (h *UserHandler) GetUser(c *gin.Context) {
	requesterID := c.GetUint("user_id")

	var uri struct {
		ID uint `uri:"id"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	profile, err := h.userService.GetUserProfile(uri.ID, requesterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": profile,
	})
}

func (h *UserHandler) GetSettings(c *gin.Context) {
	userID := c.GetUint("user_id")
	settings, err := h.userService.GetSettings(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": settings,
	})
}

func (h *UserHandler) UpdateSettings(c *gin.Context) {
	var req entity.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	userID := c.GetUint("user_id")
	if err := h.userService.UpdateSettings(userID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Настройки обновлены",
	})
}
