package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterWatcherRoutes(router *gin.RouterGroup, h *handler.WatcherHandler) {
	g := router.Group("/documents/:id/watch")
	g.Use(middleware.AuthMiddleware())

	g.GET("", h.Status)
	g.POST("", h.Toggle)
}
