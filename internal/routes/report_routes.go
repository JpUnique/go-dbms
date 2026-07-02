package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterReportRoutes registers admin-only reporting routes.
func RegisterReportRoutes(
	router *gin.RouterGroup,
	reportHandler *handler.ReportHandler,
) {
	reports := router.Group("/reports")
	reports.Use(middleware.AuthMiddleware())

	reports.GET("", reportHandler.Get)
}
