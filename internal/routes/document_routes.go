package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterDocumentRoutes registers all document endpoints
func RegisterDocumentRoutes(
	router *gin.RouterGroup,
	documentHandler *handler.DocumentHandler,
) {

	// Token-authenticated streaming (no middleware — handler validates ?token= itself)
	router.GET("/documents/:id/stream", documentHandler.Stream)

	docs := router.Group("/documents")
	docs.Use(middleware.AuthMiddleware())

	// =========================
	// DOCUMENT ENDPOINTS
	// =========================

	docs.POST("", documentHandler.Upload)
	docs.GET("", documentHandler.GetAllByFilter)

	// Admin-only cross-department browsing — registered before "/:id" so the
	// literal "by-department"/"department-counts" segments match first.
	docs.GET("/by-department", middleware.AdminOnly(), documentHandler.GetByDepartment)
	docs.GET("/department-counts", middleware.AdminOnly(), documentHandler.CountByDepartment)

	docs.GET("/:id", documentHandler.GetOne)
	docs.GET("/:id/download", documentHandler.Download)
	docs.PATCH("/:id", documentHandler.Update)
	docs.DELETE("/:id", documentHandler.Delete)
	docs.POST("/:id/star", documentHandler.ToggleStar)
}
