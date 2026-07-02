package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterNotificationRoutes(router *gin.RouterGroup, h *handler.NotificationHandler) {
	g := router.Group("/notifications")
	g.Use(middleware.AuthMiddleware())

	g.GET("", h.GetAll)
	g.PATCH("/:id/read", h.MarkRead)
	g.PATCH("/read-all", h.MarkAllRead)
}
