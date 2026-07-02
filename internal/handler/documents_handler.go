package handler

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/storage"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentHandler struct {
	service  *service.DocumentService
	db       *pgxpool.Pool
	notifSvc *service.NotificationService
	ragSvc   *service.RAGService
}

func NewDocumentHandler(service *service.DocumentService, db *pgxpool.Pool, notifSvc *service.NotificationService, ragSvc *service.RAGService) *DocumentHandler {
	return &DocumentHandler{service: service, db: db, notifSvc: notifSvc, ragSvc: ragSvc}
}

func (h *DocumentHandler) logAudit(c *gin.Context, action, resourceType string, resourceID *string, details map[string]interface{}) {
	uid, _ := c.Get("userId")
	id := uid.(string)
	ip, ua := c.ClientIP(), c.GetHeader("User-Agent")
	go utils.LogAudit(context.Background(), h.db, utils.AuditEntry{
		UserID:       &id,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    &ip,
		UserAgent:    &ua,
	})
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

	// ── File-type validation (extension + magic bytes) ────────────────────
	if err := utils.ValidateFileType(file.Filename, buf); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// ── ClamAV antivirus scan ─────────────────────────────────────────────
	// Enabled when CLAMAV_URL is set (e.g. "clamd:3310").
	// If CLAMAV_REQUIRED=true, a connection failure also rejects the upload.
	if clamdAddr := os.Getenv("CLAMAV_URL"); clamdAddr != "" {
		if scanErr := utils.ScanClamAV(clamdAddr, buf); scanErr != nil {
			required := os.Getenv("CLAMAV_REQUIRED") == "true"
			// If clamd is unreachable and not required, allow upload with a warning log
			isThreat := strings.Contains(scanErr.Error(), "threat detected")
			if required || isThreat {
				utils.Error(c, http.StatusUnprocessableEntity, "file rejected: "+scanErr.Error())
				return
			}
		}
	}

	doc, err := h.service.Upload(
		c.Request.Context(),
		buf,
		file.Filename,
		file.Header.Get("Content-Type"),
		userID.(string),
		service.UploadMeta{
			Title:       c.PostForm("title"),
			Description: c.PostForm("description"),
			FolderID:    c.PostForm("folder_id"),
			Department:  c.PostForm("department"),
			Status:      c.PostForm("status"),
		},
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "upload failed")
		return
	}

	h.logAudit(c, "upload", "document", &doc.ID, map[string]interface{}{
		"title": doc.Title, "file_name": doc.FileName, "size": doc.FileSize,
	})
	go h.notifSvc.NotifyDocumentUploaded(context.Background(), userID.(string), doc.Title, doc.ID)
	fileExt := ""
	if i := strings.LastIndex(doc.FileName, "."); i != -1 {
		fileExt = doc.FileName[i+1:]
	}
	go h.ragSvc.IndexDocument(context.Background(), doc.ID, fileExt, buf)
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

	h.logAudit(c, "view", "document", &docID, map[string]interface{}{"title": doc.Title})
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

	h.logAudit(c, "download", "document", &docID, map[string]interface{}{"file_name": fileName})
	utils.Success(c, gin.H{
		"url":       url,
		"file_name": fileName,
	})
}

// ==============================
// STREAM DOCUMENT (inline preview)
// Auth via ?token= query param so iframes can load it without headers.
// ==============================
func (h *DocumentHandler) Stream(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.Status(http.StatusUnauthorized)
		return
	}

	claims, err := utils.VerifyAccessToken(tokenStr)
	if err != nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	docID := c.Param("id")
	doc, err := h.service.GetByID(c.Request.Context(), docID, claims.UserID)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// The file lives in the OWNER's bucket regardless of who's viewing it
	// (a shared, non-owner user has their own separate bucket).
	data, contentType, err := storage.GetFileBytes(doc.OwnerID, doc.FileKey)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if contentType == "" || contentType == "application/octet-stream" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "private, max-age=300")
	c.Data(http.StatusOK, contentType, data)
}

// ==============================
// UPDATE DOCUMENT
// ==============================
func (h *DocumentHandler) Update(c *gin.Context) {

	userID, _ := c.Get("userId")
	docID := c.Param("id")

	// Use a raw map so we can distinguish "field absent" from "field = null"
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	var title *string
	if v, ok := raw["title"].(string); ok {
		title = &v
	}
	var status *string
	if v, ok := raw["status"].(string); ok {
		status = &v
	}
	var isStarred *bool
	if v, ok := raw["is_starred"].(bool); ok {
		isStarred = &v
	}

	// folder_id: key absent → nil (no change), key present+null → &nil (root), key+string → &ptr
	var folderID **string
	if _, exists := raw["folder_id"]; exists {
		if raw["folder_id"] == nil {
			var nilPtr *string
			folderID = &nilPtr
		} else if v, ok := raw["folder_id"].(string); ok {
			vCopy := v
			vPtr := &vCopy
			folderID = &vPtr
		}
	}

	doc, err := h.service.Update(
		c.Request.Context(),
		docID,
		userID.(string),
		title,
		status,
		isStarred,
		folderID,
	)

	if err != nil {

		if err == utils.ErrNotFound {
			utils.Error(c, http.StatusNotFound, "document not found")
			return
		}

		utils.Error(c, http.StatusInternalServerError, "failed to update document")
		return
	}

	h.logAudit(c, "update", "document", &docID, map[string]interface{}{"title": doc.Title})
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

	h.logAudit(c, "delete", "document", &docID, nil)
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

	action := "star"
	if !starred {
		action = "unstar"
	}
	h.logAudit(c, action, "document", &docID, nil)
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
	if query.Limit <= 0 {
		query.Limit = 20
	} else if query.Limit > 200 {
		query.Limit = 200
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
