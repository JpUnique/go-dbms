package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterTagRoutes(router *gin.RouterGroup, handler *handler.TagHandler) {

	tags := router.Group("/tags")

	protected := tags.Group("/")
	protected.Use(middleware.AuthMiddleware())

	protected.GET("/", handler.GetAll)
	protected.POST("/", middleware.AdminOnly(), handler.Create)
	protected.PATCH("/:id", middleware.AdminOnly(), handler.Update)
	protected.DELETE("/:id", middleware.AdminOnly(), handler.Delete)

	protected.POST("/documents/:docId/:tagId", handler.Attach)
	protected.DELETE("/documents/:docId/:tagId", handler.Detach)
	protected.GET("/documents/:docId", handler.GetDocumentTags)
}
