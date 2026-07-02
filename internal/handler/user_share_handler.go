package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type UserShareHandler struct {
	service *service.UserShareService
}

func NewUserShareHandler(service *service.UserShareService) *UserShareHandler {
	return &UserShareHandler{service: service}
}

// Grant — POST /documents/:id/user-shares
func (h *UserShareHandler) Grant(c *gin.Context) {

	ownerID, _ := c.Get("userId")
	sharerName, _ := c.Get("username")
	docID := c.Param("id")

	var req struct {
		UserID     string `json:"user_id" binding:"required"`
		Permission string `json:"permission" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Permission != "view" && req.Permission != "download" {
		utils.Error(c, http.StatusBadRequest, "permission must be view or download")
		return
	}

	err := h.service.Grant(c.Request.Context(), docID, ownerID.(string), sharerName.(string), req.UserID, req.Permission)
	if err != nil {
		switch err {
		case utils.ErrUnauthorized:
			utils.Error(c, http.StatusForbidden, "only the document owner can share it")
			return
		default:
			utils.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	utils.Success(c, gin.H{"message": "document shared"})
}

// List — GET /documents/:id/user-shares
func (h *UserShareHandler) List(c *gin.Context) {

	ownerID, _ := c.Get("userId")
	docID := c.Param("id")

	recipients, err := h.service.ListRecipients(c.Request.Context(), docID, ownerID.(string))
	if err != nil {
		if err == utils.ErrUnauthorized {
			utils.Error(c, http.StatusForbidden, "only the document owner can view this")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to fetch recipients")
		return
	}

	utils.Success(c, gin.H{"recipients": recipients})
}

// Revoke — DELETE /documents/:id/user-shares/:userId
func (h *UserShareHandler) Revoke(c *gin.Context) {

	ownerID, _ := c.Get("userId")
	docID := c.Param("id")
	recipientID := c.Param("userId")

	err := h.service.Revoke(c.Request.Context(), docID, ownerID.(string), recipientID)
	if err != nil {
		if err == utils.ErrUnauthorized {
			utils.Error(c, http.StatusForbidden, "only the document owner can revoke access")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to revoke access")
		return
	}

	utils.Success(c, gin.H{"message": "access revoked"})
}

// SharedWithMe — GET /shared-with-me
func (h *UserShareHandler) SharedWithMe(c *gin.Context) {

	userID, _ := c.Get("userId")

	docs, err := h.service.SharedWithMe(c.Request.Context(), userID.(string))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch shared documents")
		return
	}

	utils.Success(c, gin.H{"documents": docs})
}
