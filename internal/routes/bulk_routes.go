package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterBulkRoutes(router *gin.RouterGroup, handler *handler.BulkHandler) {

	bulk := router.Group("/bulk")
	bulk.Use(middleware.AuthMiddleware())

	bulk.POST("/documents/delete", handler.Delete)
	bulk.POST("/documents/archive", handler.Archive)
	bulk.POST("/documents/move", handler.Move)
	bulk.POST("/documents/update", handler.Update)
}
