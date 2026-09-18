package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"messengermax/pkg/jwt"
)

type Auth struct {
	jwt *jwt.Manager
}

func NewAuth(m *jwt.Manager) *Auth {
	return &Auth{jwt: m}
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
	c.Set("user_id", userID)
	c.Set("token_id", jti)
	c.Set("token", token)
	c.Next()
}
