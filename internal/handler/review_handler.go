package handler

import (
	"net/http"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/service"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	svc     *service.ReviewService
	notifSvc *service.NotificationService
	userSvc  *service.AuthService
}

func NewReviewHandler(svc *service.ReviewService, notifSvc *service.NotificationService, userSvc *service.AuthService) *ReviewHandler {
	return &ReviewHandler{svc: svc, notifSvc: notifSvc, userSvc: userSvc}
}

// Submit sets the document status to pending_review.
func (h *ReviewHandler) Submit(c *gin.Context) {
	docID := c.Param("id")
	userID, _ := c.Get("userId")
	userName := c.GetString("username")

	rev, err := h.svc.Submit(c.Request.Context(), docID, userID.(string))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Notify all admins (best-effort)
	go func() {
		admins, _ := h.userSvc.GetAdmins(c.Request.Context())
		for _, a := range admins {
			if a.ID != userID.(string) {
				h.notifSvc.NotifyReviewSubmitted(c.Request.Context(), a.ID, userName, rev.DocumentTitle, docID)
			}
		}
	}()

	utils.Created(c, gin.H{"review": rev})
}

// Approve approves the pending review and publishes the document.
func (h *ReviewHandler) Approve(c *gin.Context) {
	h.decide(c, "approved")
}

// Reject rejects the pending review.
func (h *ReviewHandler) Reject(c *gin.Context) {
	h.decide(c, "rejected")
}

func (h *ReviewHandler) decide(c *gin.Context, decision string) {
	docID := c.Param("id")
	reviewerID, _ := c.Get("userId")
	reviewerName := c.GetString("username")

	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)
	var note *string
	if body.Note != "" {
		note = &body.Note
	}

	var (
		rev *models.DocumentReview
		err error
	)
	if decision == "approved" {
		rev, err = h.svc.Approve(c.Request.Context(), docID, reviewerID.(string), note)
	} else {
		rev, err = h.svc.Reject(c.Request.Context(), docID, reviewerID.(string), note)
	}
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Notify submitter (best-effort)
	noteStr := ""
	if note != nil {
		noteStr = *note
	}
	go h.notifSvc.NotifyReviewDecision(c.Request.Context(), rev.SubmitterID, reviewerName,
		rev.DocumentTitle, docID, decision, noteStr)

	utils.Success(c, gin.H{"review": rev})
}

// GetByDocument returns review history for a document.
func (h *ReviewHandler) GetByDocument(c *gin.Context) {
	docID := c.Param("id")
	reviews, err := h.svc.GetByDocument(c.Request.Context(), docID)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch reviews")
		return
	}
	if reviews == nil {
		reviews = []*models.DocumentReview{}
	}
	utils.Success(c, gin.H{"reviews": reviews})
}

// PendingQueue returns all documents awaiting review (admin only).
func (h *ReviewHandler) PendingQueue(c *gin.Context) {
	queue, err := h.svc.PendingQueue(c.Request.Context())
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "failed to fetch review queue")
		return
	}
	if queue == nil {
		queue = []*models.DocumentReview{}
	}
	utils.Success(c, gin.H{"queue": queue})
}
