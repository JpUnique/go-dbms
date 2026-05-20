package middleware

import (
	"net/http"
	"strings"

	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware verifies JWT access token
func AuthMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			utils.Error(c, http.StatusUnauthorized, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")

		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Error(c, http.StatusUnauthorized, "invalid token format")
			c.Abort()
			return
		}

		token := parts[1]

		claims, err := utils.VerifyAccessToken(token)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		// attach user info to context
		c.Set("userId", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}
