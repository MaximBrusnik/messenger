package handlers

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/logic"
	"github.com/gin-gonic/gin"
	"net/http"
)

type UserHandler struct {
	userService logic.UserService
	authService logic.AuthService
}

func NewUserHandler(userService logic.UserService, authService logic.AuthService) *UserHandler {
	return &UserHandler{
		userService: userService,
		authService: authService,
	}
}

// @Summary Получить всех пользователей
// @Description Получение списка всех пользователей (кроме себя)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/users [get]
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

// @Summary Поиск пользователей
// @Description Поиск пользователей по имени или email
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param q query string true "Поисковый запрос"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/users/search [get]
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

// @Summary Получить контакты
// @Description Получение списка контактов пользователя
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/contacts [get]
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

// @Summary Добавить контакт
// @Description Добавление пользователя в контакты
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.AddContactRequest true "ID пользователя"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/contacts [post]
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

// @Summary Обновить профиль
// @Description Обновление данных профиля пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateProfileRequest true "Данные профиля"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/auth/profile [put]
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

// @Summary Сменить пароль
// @Description Изменение пароля пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChangePasswordRequest true "Пароли"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/auth/change-password [post]
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
