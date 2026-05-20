package handler

import (
	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

type TrashHandler struct {
	service *service.TrashService
}

func NewTrashHandler(service *service.TrashService) *TrashHandler {
	return &TrashHandler{service: service}
}

func (h *TrashHandler) GetAll(c *gin.Context) {

	userID, _ := c.Get("userId")

	docs, err := h.service.GetAll(c.Request.Context(), userID.(string))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch trash")
		return
	}

	utils.Success(c, gin.H{"documents": docs})
}

func (h *TrashHandler) Restore(c *gin.Context) {

	id := c.Param("id")

	doc, err := h.service.Restore(c.Request.Context(), id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "document not found")
		return
	}

	utils.Success(c, gin.H{"document": doc})
}

func (h *TrashHandler) Delete(c *gin.Context) {

	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		utils.Error(c, http.StatusNotFound, "document not found")
		return
	}

	utils.Success(c, gin.H{"message": "deleted permanently"})
}

func (h *TrashHandler) Empty(c *gin.Context) {

	count, err := h.service.Empty(c.Request.Context())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to empty trash")
		return
	}

	utils.Success(c, gin.H{"purged": count})
}
