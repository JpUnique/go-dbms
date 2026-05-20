package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterShareRoutes(router *gin.RouterGroup, handler *handler.ShareHandler) {

	shares := router.Group("/shares")

	// protected
	protected := shares.Group("/")
	protected.Use(middleware.AuthMiddleware())

	protected.POST("/", handler.Create)
	protected.GET("/", handler.GetAll)
	protected.DELETE("/:id", handler.Delete)

	// public
	shares.GET("/public/:token", handler.PublicAccess)
	shares.POST("/public/:token/download", handler.Download)
}
