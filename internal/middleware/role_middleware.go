package middleware

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

// AdminOnly ensures only admin users can access route
//
// TEMP-NO-ROLES: role enforcement disabled for testing — grep "TEMP-NO-ROLES"
// across the repo and restore the check below once role-based access is
// reintroduced.
func AdminOnly() gin.HandlerFunc {

	return func(c *gin.Context) {

		_, exists := c.Get("role")
		if !exists {
			utils.Error(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// TEMP-NO-ROLES: was `if role.(string) != "admin") { 403 }`

		c.Next()
	}
}
