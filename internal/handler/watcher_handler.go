package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type WatcherHandler struct {
	svc *service.WatcherService
}

func NewWatcherHandler(svc *service.WatcherService) *WatcherHandler {
	return &WatcherHandler{svc: svc}
}

// Status returns whether the current user is watching + total watcher count.
func (h *WatcherHandler) Status(c *gin.Context) {
	docID := c.Param("id")
	userID, _ := c.Get("userId")

	watching, count, err := h.svc.Status(c.Request.Context(), docID, userID.(string))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to get watcher status")
		return
	}
	utils.Success(c, gin.H{"watching": watching, "watcher_count": count})
}

// Toggle adds or removes the current user as a watcher.
func (h *WatcherHandler) Toggle(c *gin.Context) {
	docID := c.Param("id")
	userID, _ := c.Get("userId")

	watching, err := h.svc.Toggle(c.Request.Context(), docID, userID.(string))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to toggle watch")
		return
	}
	utils.Success(c, gin.H{"watching": watching})
}
