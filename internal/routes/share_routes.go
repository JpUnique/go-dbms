package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterShareRoutes(router *gin.RouterGroup, handler *handler.ShareHandler) {

	// public (no auth)
	pub := router.Group("/shares")
	pub.GET("/public/:token", handler.PublicAccess)
	pub.POST("/public/:token/download", handler.Download)

	// protected
	shares := router.Group("/shares")
	shares.Use(middleware.AuthMiddleware())
	shares.POST("", handler.Create)
	shares.GET("", handler.GetAll)
	shares.DELETE("/:id", handler.Delete)
}
