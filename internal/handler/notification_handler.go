package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) GetAll(c *gin.Context) {
	userID, _ := c.Get("userId")
	list, err := h.svc.GetForUser(c.Request.Context(), userID.(string))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch notifications")
		return
	}
	count, _ := h.svc.UnreadCount(c.Request.Context(), userID.(string))
	utils.Success(c, gin.H{"notifications": list, "unread_count": count})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, _ := c.Get("userId")
	id := c.Param("id")
	if err := h.svc.MarkRead(c.Request.Context(), id, userID.(string)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to mark notification as read")
		return
	}
	utils.Success(c, gin.H{"updated": true})
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get("userId")
	if err := h.svc.MarkAllRead(c.Request.Context(), userID.(string)); err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to mark all as read")
		return
	}
	utils.Success(c, gin.H{"updated": true})
}
