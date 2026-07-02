package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterUserShareRoutes registers direct per-user document sharing routes
// — distinct from RegisterShareRoutes, which handles token-based public links.
func RegisterUserShareRoutes(router *gin.RouterGroup, h *handler.UserShareHandler) {

	docs := router.Group("/documents/:id/user-shares")
	docs.Use(middleware.AuthMiddleware())
	docs.POST("", h.Grant)
	docs.GET("", h.List)
	docs.DELETE("/:userId", h.Revoke)

	sharedWithMe := router.Group("/shared-with-me")
	sharedWithMe.Use(middleware.AuthMiddleware())
	sharedWithMe.GET("", h.SharedWithMe)
}
