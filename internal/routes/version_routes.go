package routes

import (
	"github.com/JpUnique/go-dbms/internal/handler"
	"github.com/JpUnique/go-dbms/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterDocumentVersionRoutes(
	router *gin.RouterGroup,
	handler *handler.DocumentVersionHandler,
) {

	group := router.Group("/documents/:id/versions")
	group.Use(middleware.AuthMiddleware())

	group.GET("/get-versions", handler.GetVersions)
	group.POST("/upload-version", handler.UploadVersion)
	group.GET("/:versionId/download", handler.DownloadVersion)
}
