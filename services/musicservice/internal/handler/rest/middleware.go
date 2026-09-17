package rest

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func GatewayIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v := c.GetHeader("X-User-Id"); v != "" {
			if id, err := strconv.ParseUint(v, 10, 64); err == nil {
				c.Set("user_id", uint(id))
			}
		}
		c.Set("is_admin", c.GetHeader("X-Is-Admin") == "true")
		c.Next()
	}
}
