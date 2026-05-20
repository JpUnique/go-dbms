package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterAuditRoutes(router *gin.RouterGroup, handler *handler.AuditHandler) {

	group := router.Group("/audit")
	group.Use(middleware.AuthMiddleware())

	group.GET("/", handler.GetAll)
	group.DELETE("/", handler.Delete)
}
