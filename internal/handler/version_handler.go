package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type DocumentVersionHandler struct {
	service *service.DocumentVersionService
}

func NewDocumentVersionHandler(service *service.DocumentVersionService) *DocumentVersionHandler {
	return &DocumentVersionHandler{service: service}
}
func (h *DocumentVersionHandler) GetVersions(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	docID := c.Param("id")

	versions, err := h.service.GetVersions(
		c.Request.Context(),
		docID,
		userID.(string),
	)

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch versions")
		return
	}

	utils.Success(c, gin.H{"versions": versions})
}

func (h *DocumentVersionHandler) UploadVersion(c *gin.Context) {

	userID, _ := c.Get("userId")
	docID := c.Param("id")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "file required")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to open file")
		return
	}
	defer file.Close()

	buffer := make([]byte, fileHeader.Size)
	if _, err := file.Read(buffer); err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to read file")
		return
	}

	changeNote := c.PostForm("change_note")

	version, err := h.service.UploadVersion(
		c.Request.Context(),
		docID,
		userID.(string),
		buffer,
		fileHeader.Filename,
		fileHeader.Header.Get("Content-Type"),
		changeNote,
	)

	if err != nil {
		if err == utils.ErrNotFound {
			utils.Error(c, http.StatusNotFound, "document not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to upload version")
		return
	}

	utils.Created(c, gin.H{"version": version})
}

func (h *DocumentVersionHandler) DownloadVersion(c *gin.Context) {

	userID, _ := c.Get("userId")
	docID := c.Param("id")
	versionID := c.Param("versionId")

	url, fileName, err := h.service.DownloadVersion(
		c.Request.Context(),
		docID,
		versionID,
		userID.(string),
	)

	if err != nil {
		utils.Error(c, http.StatusNotFound, "version not found")
		return
	}

	utils.Success(c, gin.H{
		"url":       url,
		"file_name": fileName,
	})
}
