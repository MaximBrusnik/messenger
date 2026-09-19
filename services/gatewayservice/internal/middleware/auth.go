package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"messengermax/pkg/jwt"
)

// ActiveChecker проверяет, что аккаунт не заблокирован и существует.
type ActiveChecker interface {
	IsUserActive(ctx context.Context, userID uint) (active bool, found bool)
}

type Auth struct {
	jwt    *jwt.Manager
	active ActiveChecker
}

func NewAuth(m *jwt.Manager, active ActiveChecker) *Auth {
	return &Auth{jwt: m, active: active}
}

func (a *Auth) Required(c *gin.Context) {
	token := ""
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		token = strings.TrimPrefix(h, "Bearer ")
	} else if cookie, err := c.Cookie("access_token"); err == nil {
		token = cookie
	}
	if token == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Токен не предоставлен"})
		return
	}
	userID, jti, err := a.jwt.ParseToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен"})
		return
	}
	if a.active != nil {
		active, found := a.active.IsUserActive(c.Request.Context(), userID)
		if !found {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Сессия завершена"})
			return
		}
		if !active {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Аккаунт заблокирован"})
			return
		}
	}
	c.Set("user_id", userID)
	c.Set("token_id", jti)
	c.Set("token", token)
	c.Next()
}
