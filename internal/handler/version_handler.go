package handler

import (
	"io"
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

// ======================================
// HELPER: GET USER CONTEXT ✅
// ======================================
func getUserCont(c *gin.Context) (string, string, bool) {
	userIDVal, ok1 := c.Get("userId")
	usernameVal, ok2 := c.Get("username")

	if !ok1 || !ok2 {
		return "", "", false
	}

	userID, ok1 := userIDVal.(string)
	username, ok2 := usernameVal.(string)

	if !ok1 || !ok2 {
		return "", "", false
	}

	return userID, username, true
}

// ======================================
// GET VERSIONS
// ======================================
func (h *DocumentVersionHandler) GetVersions(c *gin.Context) {

	docID := c.Param("id")
	if docID == "" {
		utils.Error(c, http.StatusBadRequest, "invalid document id")
		return
	}

	userID, _, ok := getUserCont(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	role, _ := c.Get("role")
	versions, err := h.service.GetVersions(
		c.Request.Context(),
		docID,
		userID,
		role.(string),
	)

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch versions")
		return
	}

	utils.Success(c, gin.H{"versions": versions})
}

// ======================================
// UPLOAD VERSION ✅ FIXED
// ======================================
func (h *DocumentVersionHandler) UploadVersion(c *gin.Context) {

	docID := c.Param("id")
	if docID == "" {
		utils.Error(c, http.StatusBadRequest, "invalid document id")
		return
	}

	userID, username, ok := getUserCont(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

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

	// ✅ SAFER FILE READ
	buffer, err := io.ReadAll(file)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to read file")
		return
	}

	changeNote := c.PostForm("change_note")

	role, _ := c.Get("role")
	version, err := h.service.UploadVersion(
		c.Request.Context(),
		docID,
		userID,
		role.(string),
		username,
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

// ======================================
// DOWNLOAD VERSION ✅ FIXED
// ======================================
func (h *DocumentVersionHandler) DownloadVersion(c *gin.Context) {

	docID := c.Param("id")
	versionID := c.Param("versionId")

	if docID == "" || versionID == "" {
		utils.Error(c, http.StatusBadRequest, "invalid parameters")
		return
	}

	userID, _, ok := getUserCont(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	role, _ := c.Get("role")
	url, fileName, err := h.service.DownloadVersion(
		c.Request.Context(),
		docID,
		versionID,
		userID,
		role.(string),
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
