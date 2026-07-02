package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type TrashHandler struct {
	service *service.TrashService
}

func NewTrashHandler(service *service.TrashService) *TrashHandler {
	return &TrashHandler{service: service}
}

// ======================================
// HELPER: GET USER CONTEXT SAFELY ✅
// ======================================
func getUserContext(c *gin.Context) (string, string, bool) {
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
// GET ALL TRASH
// ======================================
func (h *TrashHandler) GetAll(c *gin.Context) {

	userIDVal, ok := c.Get("userId")
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID := userIDVal.(string)

	docs, err := h.service.GetAll(c.Request.Context(), userID)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch trash")
		return
	}

	utils.Success(c, gin.H{"documents": docs})
}

// ======================================
// RESTORE DOCUMENT
// ======================================
func (h *TrashHandler) Restore(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		utils.Error(c, http.StatusBadRequest, "invalid document id")
		return
	}

	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	doc, err := h.service.Restore(
		c.Request.Context(),
		id,
		userID,
	)

	if err != nil {
		utils.Error(c, http.StatusNotFound, "document not found")
		return
	}

	utils.Success(c, gin.H{"document": doc})
}

// ======================================
// DELETE DOCUMENT
// ======================================
func (h *TrashHandler) Delete(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		utils.Error(c, http.StatusBadRequest, "invalid document id")
		return
	}

	userID, username, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := h.service.Delete(
		c.Request.Context(),
		id,
		userID,
		username,
	)

	if err != nil {
		utils.Error(c, http.StatusNotFound, "document not found")
		return
	}

	utils.Success(c, gin.H{"message": "deleted permanently"})
}

// ======================================
// EMPTY TRASH
// ======================================
func (h *TrashHandler) Empty(c *gin.Context) {

	userID, username, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	count, err := h.service.Empty(
		c.Request.Context(),
		userID,
		username,
	)

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to empty trash")
		return
	}

	utils.Success(c, gin.H{"purged": count})
}
