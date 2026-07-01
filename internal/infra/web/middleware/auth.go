package middleware

import (
	"strings"

	"github.com/gabrielgcosta/ticketblast-core/internal/infra/auth"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/apierror"
	"github.com/gin-gonic/gin"
)

func Auth(tokenEngine *auth.TokenEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apierror.Write(c, apierror.Unauthorized("Missing authorization header", nil))
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			apierror.Write(c, apierror.Unauthorized("Invalid authorization header format", nil))
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := tokenEngine.ValidateToken(tokenStr)
		if err != nil {
			apierror.Write(c, apierror.Unauthorized("Invalid or expired token", err))
			c.Abort()
			return
		}

		// Inject user info into the Gin context for handlers and logging
		c.Set("usuario_id", claims.UserID)
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}
