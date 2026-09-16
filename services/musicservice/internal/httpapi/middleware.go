package httpapi

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GatewayIdentity injects the identity forwarded by the api-gateway. It must
// only be exposed behind the gateway, never to the public internet.
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
