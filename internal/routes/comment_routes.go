package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterCommentRoutes(router *gin.RouterGroup, h *handler.CommentHandler) {
	group := router.Group("/documents/:id/comments")
	group.Use(middleware.AuthMiddleware())

	group.GET("", h.GetAll)
	group.POST("", h.Create)
	group.DELETE("/:commentId", h.Delete)
}
