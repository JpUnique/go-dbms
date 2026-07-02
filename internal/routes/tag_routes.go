package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterTagRoutes(router *gin.RouterGroup, handler *handler.TagHandler) {

	tags := router.Group("/tags")
	tags.Use(middleware.AuthMiddleware())

	tags.GET("", handler.GetAll)
	tags.POST("", handler.Create)
	tags.PATCH("/:id", handler.Update)
	tags.DELETE("/:id", handler.Delete)
	tags.GET("/by-tag/:id", handler.GetDocuments)
	tags.POST("/documents/:docId/:tagId", handler.Attach)
	tags.DELETE("/documents/:docId/:tagId", handler.Detach)
	tags.GET("/documents/:docId", handler.GetDocumentTags)
}
