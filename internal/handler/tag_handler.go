package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type TagHandler struct {
	service *service.TagService
}

func NewTagHandler(service *service.TagService) *TagHandler {
	return &TagHandler{service: service}
}

func (h *TagHandler) GetAll(c *gin.Context) {

	tags, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch tags")
		return
	}

	utils.Success(c, gin.H{"tags": tags})
}

// Admin Only
func (h *TagHandler) Create(c *gin.Context) {

	var req struct {
		Name  string `json:"name" binding:"required"`
		Color string `json:"color"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	tag, err := h.service.Create(c.Request.Context(), req.Name, req.Color)
	if err != nil {
		utils.Error(c, http.StatusConflict, err.Error())
		return
	}

	utils.Created(c, gin.H{"tag": tag})
}

func (h *TagHandler) Update(c *gin.Context) {

	id := c.Param("id")

	var req struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid body")
		return
	}

	tag, err := h.service.Update(
		c.Request.Context(),
		id,
		req.Name,
		req.Color,
	)

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to update tag")
		return
	}

	utils.Success(c, gin.H{"tag": tag})
}

func (h *TagHandler) Delete(c *gin.Context) {

	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "tag not found")
		return
	}

	utils.Success(c, gin.H{"message": "tag deleted"})
}

func (h *TagHandler) GetDocuments(c *gin.Context) {
	tagID := c.Param("id")
	docs, err := h.service.GetDocumentsByTag(c.Request.Context(), tagID)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch documents")
		return
	}
	if docs == nil {
		docs = []models.DocumentWithOwner{}
	}
	utils.Success(c, gin.H{"documents": docs})
}

func (h *TagHandler) Attach(c *gin.Context) {

	userID, _ := c.Get("userId")

	docID := c.Param("docId")
	tagID := c.Param("tagId")

	err := h.service.Attach(c.Request.Context(), docID, tagID, userID.(string))
	if err != nil {
		utils.Error(c, http.StatusForbidden, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "tag attached"})
}

func (h *TagHandler) Detach(c *gin.Context) {

	userID, _ := c.Get("userId")

	docID := c.Param("docId")
	tagID := c.Param("tagId")

	err := h.service.Detach(c.Request.Context(), docID, tagID, userID.(string))
	if err != nil {
		utils.Error(c, http.StatusForbidden, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "tag removed"})
}

func (h *TagHandler) GetDocumentTags(c *gin.Context) {

	userID, _ := c.Get("userId")
	docID := c.Param("docId")

	tags, err := h.service.GetByDocument(
		c.Request.Context(),
		docID,
		userID.(string),
	)
	if err != nil {
		utils.Error(c, http.StatusForbidden, "not allowed")
		return
	}

	utils.Success(c, gin.H{"tags": tags})
}
