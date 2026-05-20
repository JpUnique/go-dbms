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

		//  Ensure role is admin
		if role.(string) != "admin" {
			utils.Error(c, http.StatusForbidden, "forbidden: admin only")
			c.Abort()
			return
		}

		c.Next()
	}
}
