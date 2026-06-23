package middleware

import (
	"MessangerMax/internal/repo"
	"MessangerMax/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func AuthMiddleware(jwtUtils utils.JWTUtils) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получение токена из заголовка
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Попытка получить из куки
			if token, err := c.Cookie("access_token"); err == nil {
				tokenString = token
			}
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Требуется авторизация",
			})
			c.Abort()
			return
		}

		// Валидация токена
		userID, err := jwtUtils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Неверный токен",
			})
			c.Abort()
			return
		}

		// Сохранение userID в контексте
		c.Set("user_id", userID)
		c.Next()
	}
}

// AdminMiddleware проверяет, что пользователь является администратором
func AdminMiddleware(userRepo repo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
			c.Abort()
			return
		}

		user, err := userRepo.FindByID(userID.(uint))
		if err != nil || !user.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Доступ запрещён"})
			c.Abort()
			return
		}

		c.Set("is_admin", true)
		c.Next()
	}
}

// Middleware для CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
