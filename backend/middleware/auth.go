package middleware

import (
	"strings"

	"livepoll/config"
	"livepoll/utils"

	"github.com/gin-gonic/gin"
)

func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			utils.Error(c, 401, "Authentication required", "missing token")
			c.Abort()
			return
		}

		parts := strings.Fields(authorization)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.Error(c, 401, "Authentication required", "invalid authorization format")
			c.Abort()
			return
		}

		claims, err := utils.ValidateJWT(cfg.JWTSecret, parts[1])
		if err != nil {
			utils.Error(c, 401, "Authentication required", "invalid token")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Next()
	}
}
