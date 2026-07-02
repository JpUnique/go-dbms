package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterFolderRoutes(
	router *gin.RouterGroup,
	handler *handler.FolderHandler,
) {

	folders := router.Group("/folders")
	folders.Use(middleware.AuthMiddleware())

	folders.GET("", handler.GetAll)
	folders.GET("/:id", handler.GetOne)
	folders.POST("", handler.Create)
	folders.PATCH("/:id", handler.Update)
	folders.DELETE("/:id", handler.Delete)
}
