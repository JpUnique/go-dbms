package middleware

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

// AdminOnly ensures only admin users can access route
func AdminOnly() gin.HandlerFunc {

	return func(c *gin.Context) {

		role, exists := c.Get("role")
		if !exists {
			utils.Error(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		if role.(string) != "admin" {
			utils.Error(c, http.StatusForbidden, "admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
