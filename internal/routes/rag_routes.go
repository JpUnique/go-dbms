package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRAGRoutes(router *gin.RouterGroup, h *handler.RAGHandler) {
	chat := router.Group("/chat")
	chat.Use(middleware.AuthMiddleware())

	chat.POST("/ask", h.Ask)
	chat.POST("/sessions", h.CreateSession)
	chat.GET("/sessions", h.ListSessions)
	chat.GET("/sessions/:id", h.GetSession)
	chat.DELETE("/sessions/:id", h.DeleteSession)
}
