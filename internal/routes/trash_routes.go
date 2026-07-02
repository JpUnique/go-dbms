package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterTrashRoutes(router *gin.RouterGroup, handler *handler.TrashHandler) {

	trash := router.Group("/trash")
	trash.Use(middleware.AuthMiddleware())

	trash.GET("", handler.GetAll)
	trash.POST("/:id/restore", handler.Restore)
	trash.DELETE("/:id", handler.Delete)
	trash.DELETE("", handler.Empty)
}
