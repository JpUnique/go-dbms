package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	service *service.DocumentService
}

func NewDocumentHandler(service *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{service: service}
}

// ==============================
// UPLOAD DOCUMENT
// ==============================
func (h *DocumentHandler) Upload(c *gin.Context) {

	userID, _ := c.Get("userId")

	file, err := c.FormFile("file")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "file is required")
		return
	}

	f, err := file.Open()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to open file")
		return
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	if _, err := f.Read(buf); err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to read file")
		return
	}

	doc, err := h.service.Upload(
		c.Request.Context(),
		buf,
		file.Filename,
		file.Header.Get("Content-Type"),
		userID.(string),
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "upload failed")
		return
	}

	utils.Created(c, gin.H{"document": doc})
}

// ==============================
// GET ALL DOCUMENTS
// ==============================
func (h *DocumentHandler) GetAll(c *gin.Context) {

	userID, _ := c.Get("userId")

	docs, err := h.service.GetAll(c.Request.Context(), userID.(string))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch documents")
		return
	}

	utils.Success(c, gin.H{"documents": docs})
}

// ==============================
// GET SINGLE DOCUMENT
// ==============================
func (h *DocumentHandler) GetOne(c *gin.Context) {

	userID, _ := c.Get("userId")
	docID := c.Param("id")

	doc, err := h.service.GetByID(c.Request.Context(), docID, userID.(string))
	if err != nil {

		if err == utils.ErrNotFound {
			utils.Error(c, http.StatusNotFound, "document not found")
			return
		}

		utils.Error(c, http.StatusInternalServerError, "failed to fetch document")
		return
	}

	utils.Success(c, gin.H{"document": doc})
}

// ==============================
// DOWNLOAD DOCUMENT
// ==============================
func (h *DocumentHandler) Download(c *gin.Context) {

	userID, _ := c.Get("userId")
	docID := c.Param("id")

	url, fileName, err :=
		h.service.GetDownloadURL(c.Request.Context(), docID, userID.(string))

	if err != nil {
		utils.Error(c, http.StatusNotFound, "document not found")
		return
	}

	utils.Success(c, gin.H{
		"url":       url,
		"file_name": fileName,
	})
}

// ==============================
// UPDATE DOCUMENT
// ==============================
func (h *DocumentHandler) Update(c *gin.Context) {

	userID, _ := c.Get("userId")
	docID := c.Param("id")

	var req struct {
		Title     *string `json:"title"`
		Status    *string `json:"status"`
		IsStarred *bool   `json:"is_starred"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	doc, err := h.service.Update(
		c.Request.Context(),
		docID,
		userID.(string),
		req.Title,
		req.Status,
		req.IsStarred,
	)

	if err != nil {

		if err == utils.ErrNotFound {
			utils.Error(c, http.StatusNotFound, "document not found")
			return
		}

		utils.Error(c, http.StatusInternalServerError, "failed to update document")
		return
	}

	utils.Success(c, gin.H{"document": doc})
}

// ==============================
// DELETE DOCUMENT
// ==============================
func (h *DocumentHandler) Delete(c *gin.Context) {

	userID, _ := c.Get("userId")
	docID := c.Param("id")

	err := h.service.Delete(
		c.Request.Context(),
		docID,
		userID.(string),
	)

	if err != nil {

		if err == utils.ErrNotFound {
			utils.Error(c, http.StatusNotFound, "document not found")
			return
		}

		utils.Error(c, http.StatusInternalServerError, "failed to delete document")
		return
	}

	utils.Success(c, gin.H{"message": "document deleted"})
}

// ==============================
// STAR DOCUMENT
// ==============================
func (h *DocumentHandler) ToggleStar(c *gin.Context) {

	userID, _ := c.Get("userId")
	docID := c.Param("id")

	starred, err := h.service.ToggleStar(
		c.Request.Context(),
		docID,
		userID.(string),
	)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "document not found")
		return
	}

	utils.Success(c, gin.H{"is_starred": starred})
}

// ==============================
// GET ALL DOCUMENTS BY FILTER
// ==============================
func (h *DocumentHandler) GetAllByFilter(c *gin.Context) {

	userID, exists := c.Get("userId")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var query models.DocumentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	// ✅ defaults
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 50 {
		query.Limit = 10
	}

	docs, total, err := h.service.GetAllByFilter(
		c.Request.Context(),
		userID.(string),
		query,
	)

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch documents")
		return
	}

	utils.Success(c, gin.H{
		"documents": docs,
		"meta": gin.H{
			"page":  query.Page,
			"limit": query.Limit,
			"total": total,
		},
	})
}
