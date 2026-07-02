package handler

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

var mentionRe = regexp.MustCompile(`@(\w+)`)

type CommentHandler struct {
	service    *service.CommentService
	notifSvc   *service.NotificationService
	watcherSvc *service.WatcherService
	docSvc     *service.DocumentService
}

func NewCommentHandler(
	svc *service.CommentService,
	notifSvc *service.NotificationService,
	watcherSvc *service.WatcherService,
	docSvc *service.DocumentService,
) *CommentHandler {
	return &CommentHandler{service: svc, notifSvc: notifSvc, watcherSvc: watcherSvc, docSvc: docSvc}
}

func (h *CommentHandler) GetAll(c *gin.Context) {
	docID := c.Param("id")
	comments, err := h.service.GetAll(c.Request.Context(), docID)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch comments")
		return
	}
	if comments == nil {
		comments = []*models.Comment{}
	}
	utils.Success(c, gin.H{"comments": comments})
}

func (h *CommentHandler) Create(c *gin.Context) {
	docID := c.Param("id")
	userID, _ := c.Get("userId")
	commenterName := c.GetString("username")

	var body struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Content) == "" {
		utils.Error(c, http.StatusBadRequest, "content is required")
		return
	}
	content := strings.TrimSpace(body.Content)

	comment, err := h.service.Create(c.Request.Context(), docID, userID.(string), content)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to create comment")
		return
	}

	go h.fireCommentNotifications(context.Background(), docID, userID.(string), commenterName, content)

	utils.Created(c, gin.H{"comment": comment})
}

func (h *CommentHandler) fireCommentNotifications(ctx context.Context, docID, commenterID, commenterName, content string) {
	doc, err := h.docSvc.GetByID(ctx, docID, commenterID)
	if err != nil || doc == nil {
		return
	}

	// Notify document owner.
	if doc.OwnerID != commenterID {
		h.notifSvc.NotifyCommentAdded(ctx, doc.OwnerID, commenterName, doc.Title, docID)
	}

	// Notify watchers.
	watcherIDs, _ := h.watcherSvc.WatcherIDs(ctx, docID)
	h.notifSvc.NotifyWatchers(ctx, watcherIDs, commenterID, commenterName, doc.Title, docID, "comment")

	// Parse @mentions and notify each mentioned user.
	for _, m := range mentionRe.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		mentionedUser, err := h.service.GetUserByName(ctx, m[1])
		if err != nil || mentionedUser == nil || mentionedUser.ID == commenterID {
			continue
		}
		h.notifSvc.NotifyMentioned(ctx, mentionedUser.ID, commenterName, doc.Title, docID)
	}
}

func (h *CommentHandler) Delete(c *gin.Context) {
	commentID := c.Param("commentId")
	userID, _ := c.Get("userId")
	role, _ := c.Get("role")

	roleStr, _ := role.(string)
	if err := h.service.Delete(c.Request.Context(), commentID, userID.(string), roleStr); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "authorised") {
			utils.Error(c, http.StatusForbidden, "comment not found or you do not have permission")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to delete comment")
		return
	}
	utils.Success(c, gin.H{"deleted": true})
}
