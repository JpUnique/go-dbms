package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterStatsRoutes(router *gin.RouterGroup, handler *handler.StatsHandler) {

	stats := router.Group("/stats")
	stats.Use(middleware.AuthMiddleware())

	stats.GET("/dashboard", handler.Dashboard)
	stats.GET("/activity", handler.Activity)
}
