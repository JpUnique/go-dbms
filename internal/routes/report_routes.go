package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterReportRoutes registers reporting routes. Every user can generate
// their own report (scope=own, the default); scope=all is admin-only,
// enforced inside ReportHandler.Get itself.
func RegisterReportRoutes(
	router *gin.RouterGroup,
	reportHandler *handler.ReportHandler,
) {
	reports := router.Group("/reports")
	reports.Use(middleware.AuthMiddleware())

	reports.GET("", reportHandler.Get)
}
