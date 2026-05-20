package handler

import (
	"net/http"
	"strconv"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	service *service.AuditService
}

func NewAuditHandler(service *service.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

func (h *AuditHandler) GetAll(c *gin.Context) {

	userID, _ := c.Get("userId")
	role, _ := c.Get("role")

	// ✅ query params
	userFilter := c.Query("user_id")
	resourceType := c.Query("resource_type")
	resourceID := c.Query("resource_id")
	action := c.Query("action")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := h.service.GetAll(
		c.Request.Context(),
		userID.(string),
		role.(string),
		userFilter,
		resourceType,
		resourceID,
		action,
		limit,
		offset,
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch audit logs")
		return
	}

	utils.Success(c, gin.H{
		"logs": logs,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (h *AuditHandler) Delete(c *gin.Context) {

	role, _ := c.Get("role")

	if role.(string) != "admin" {
		utils.Error(c, http.StatusForbidden, "admin only")
		return
	}

	before := c.Query("before")

	err := h.service.Delete(c.Request.Context(), before)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to delete logs")
		return
	}

	utils.Success(c, gin.H{"success": true})
}
