package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type BulkHandler struct {
	service *service.BulkService
}

func NewBulkHandler(service *service.BulkService) *BulkHandler {
	return &BulkHandler{service: service}
}

func (h *BulkHandler) Delete(c *gin.Context) {

	userID, _ := c.Get("userId")

	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		utils.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	count, err := h.service.Delete(
		c.Request.Context(),
		userID.(string),
		req.IDs,
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "bulk delete failed")
		return
	}

	utils.Success(c, gin.H{
		"deleted": count,
	})
}

func (h *BulkHandler) Archive(c *gin.Context) {

	userID, _ := c.Get("userId")

	var req struct {
		IDs []string `json:"ids"`
	}

	c.ShouldBindJSON(&req)

	count, err := h.service.Archive(
		c.Request.Context(),
		userID.(string),
		req.IDs,
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "bulk archive failed")
		return
	}

	utils.Success(c, gin.H{"archived": count})
}

func (h *BulkHandler) Move(c *gin.Context) {

	userID, _ := c.Get("userId")

	var req struct {
		IDs      []string `json:"ids"`
		FolderID *string  `json:"folder_id"`
	}

	c.ShouldBindJSON(&req)

	count, err := h.service.Move(
		c.Request.Context(),
		userID.(string),
		req.IDs,
		req.FolderID,
	)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, gin.H{"moved": count})
}

func (h *BulkHandler) Update(c *gin.Context) {

	userID, _ := c.Get("userId")

	var req struct {
		IDs        []string `json:"ids"`
		Status     *string  `json:"status"`
		Department *string  `json:"department"`
	}

	c.ShouldBindJSON(&req)

	count, err := h.service.Update(
		c.Request.Context(),
		userID.(string),
		req.IDs,
		req.Status,
		req.Department,
	)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "bulk update failed")
		return
	}

	utils.Success(c, gin.H{"updated": count})
}
