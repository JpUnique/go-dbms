package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterReviewRoutes(router *gin.RouterGroup, h *handler.ReviewHandler) {
	// Per-document review actions
	doc := router.Group("/documents/:id")
	doc.Use(middleware.AuthMiddleware())
	doc.POST("/submit-review", h.Submit)
	doc.POST("/approve", h.Approve)
	doc.POST("/reject", h.Reject)
	doc.GET("/reviews", h.GetByDocument)

	// Admin review queue
	queue := router.Group("/review-queue")
	queue.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	queue.GET("", h.PendingQueue)
}
