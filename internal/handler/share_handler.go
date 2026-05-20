package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type ShareHandler struct {
	service *service.ShareService
}

func NewShareHandler(service *service.ShareService) *ShareHandler {
	return &ShareHandler{service: service}
}

func (h *ShareHandler) Create(c *gin.Context) {

	userID, _ := c.Get("userId")

	var req struct {
		DocumentID string  `json:"document_id" binding:"required"`
		Permission string  `json:"permission"`
		Password   *string `json:"password"`
		ExpiresAt  *string `json:"expires_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	share, err := h.service.Create(
		c.Request.Context(),
		userID.(string),
		req.DocumentID,
		req.Permission,
		req.Password,
		req.ExpiresAt,
	)

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to create share")
		return
	}

	utils.Created(c, gin.H{"share": share})
}

func (h *ShareHandler) GetAll(c *gin.Context) {

	userID, _ := c.Get("userId")

	shares, err := h.service.GetAll(c.Request.Context(), userID.(string))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch shares")
		return
	}

	utils.Success(c, gin.H{"shares": shares})
}

func (h *ShareHandler) Delete(c *gin.Context) {

	userID, _ := c.Get("userId")
	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id, userID.(string))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "share not found")
		return
	}

	utils.Success(c, gin.H{"message": "share revoked"})
}

func (h *ShareHandler) PublicAccess(c *gin.Context) {

	token := c.Param("token")

	result, err := h.service.PublicAccess(c.Request.Context(), token)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "share not available")
		return
	}

	utils.Success(c, result)
}

func (h *ShareHandler) Download(c *gin.Context) {

	token := c.Param("token")

	var body struct {
		Password string `json:"password"`
	}

	c.ShouldBindJSON(&body)

	url, fileName, err := h.service.Download(
		c.Request.Context(),
		token,
		body.Password,
	)

	if err != nil {
		utils.Error(c, http.StatusForbidden, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"url":       url,
		"file_name": fileName,
	})
}
