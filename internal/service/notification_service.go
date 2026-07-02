package service

import (
	"context"
	"fmt"
	"log"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
	"github.com/JpUnique/go-dbms/internal/utils"
)

type NotificationService struct {
	repo *repository.NotificationRepository
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// Notify creates an in-app notification and optionally sends an email.
// It respects the user's email_notifications preference.
func (s *NotificationService) Notify(ctx context.Context, n *models.Notification, emailSubject, emailBody string) {
	if _, err := s.repo.Create(ctx, n); err != nil {
		log.Printf("notification create: %v", err)
	}
	if emailBody != "" {
		email, wantsEmail, err := s.repo.GetUserEmailPref(ctx, n.UserID)
		if err != nil {
			log.Printf("notification email pref lookup: %v", err)
			return
		}
		if wantsEmail {
			if err := utils.SendEmail(email, emailSubject, emailBody); err != nil {
				log.Printf("notification send email: %v", err)
			}
		}
	}
}

// NotifyCommentAdded notifies the document owner when someone comments on their document.
func (s *NotificationService) NotifyCommentAdded(ctx context.Context, ownerID, commenterName, docTitle, docID string) {
	if ownerID == "" {
		return
	}
	resourceID := docID
	s.Notify(ctx, &models.Notification{
		UserID:       ownerID,
		Type:         "comment_added",
		Title:        fmt.Sprintf("New comment on \"%s\"", docTitle),
		Body:         fmt.Sprintf("%s commented on your document.", commenterName),
		ResourceType: "document",
		ResourceID:   &resourceID,
	},
		fmt.Sprintf("[PETRODATA] New comment on \"%s\"", docTitle),
		fmt.Sprintf("Hi,\n\n%s left a comment on your document \"%s\".\n\nLog in to view it.\n\nPETRODATA Team", commenterName, docTitle),
	)
}

// NotifyMentioned notifies a user they were @mentioned in a comment.
func (s *NotificationService) NotifyMentioned(ctx context.Context, mentionedUserID, mentionerName, docTitle, docID string) {
	resourceID := docID
	s.Notify(ctx, &models.Notification{
		UserID:       mentionedUserID,
		Type:         "mentioned",
		Title:        fmt.Sprintf("You were mentioned in \"%s\"", docTitle),
		Body:         fmt.Sprintf("%s mentioned you in a comment.", mentionerName),
		ResourceType: "document",
		ResourceID:   &resourceID,
	},
		fmt.Sprintf("[PETRODATA] %s mentioned you in \"%s\"", mentionerName, docTitle),
		fmt.Sprintf("Hi,\n\n%s mentioned you in a comment on \"%s\".\n\nLog in to view it.\n\nPETRODATA Team", mentionerName, docTitle),
	)
}

// NotifyDocumentShared notifies the recipient of a share link.
func (s *NotificationService) NotifyDocumentShared(ctx context.Context, recipientID, sharerName, docTitle, docID string) {
	resourceID := docID
	s.Notify(ctx, &models.Notification{
		UserID:       recipientID,
		Type:         "doc_shared",
		Title:        fmt.Sprintf("\"%s\" was shared with you", docTitle),
		Body:         fmt.Sprintf("%s shared a document with you.", sharerName),
		ResourceType: "document",
		ResourceID:   &resourceID,
	},
		fmt.Sprintf("[PETRODATA] \"%s\" was shared with you", docTitle),
		fmt.Sprintf("Hi,\n\n%s shared the document \"%s\" with you on PETRODATA.\n\nLog in to view it.\n\nPETRODATA Team", sharerName, docTitle),
	)
}

// NotifyReviewSubmitted notifies admins that a document needs review.
func (s *NotificationService) NotifyReviewSubmitted(ctx context.Context, adminID, submitterName, docTitle, docID string) {
	resourceID := docID
	s.Notify(ctx, &models.Notification{
		UserID:       adminID,
		Type:         "review_submitted",
		Title:        fmt.Sprintf("\"%s\" submitted for review", docTitle),
		Body:         fmt.Sprintf("%s submitted a document for your review.", submitterName),
		ResourceType: "document",
		ResourceID:   &resourceID,
	},
		fmt.Sprintf("[PETRODATA] Document awaiting review: \"%s\"", docTitle),
		fmt.Sprintf("Hi,\n\n%s submitted \"%s\" for your review.\n\nLog in to approve or reject it.\n\nPETRODATA Team", submitterName, docTitle),
	)
}

// NotifyReviewDecision notifies the submitter of an approve/reject outcome.
func (s *NotificationService) NotifyReviewDecision(ctx context.Context, submitterID, reviewerName, docTitle, docID, decision, note string) {
	resourceID := docID
	approved := decision == "approved"
	notifType := "review_rejected"
	title := fmt.Sprintf("Your document \"%s\" was rejected", docTitle)
	body := fmt.Sprintf("%s rejected your document.", reviewerName)
	if note != "" {
		body += " Reason: " + note
	}
	emailSubject := fmt.Sprintf("[PETRODATA] Document rejected: \"%s\"", docTitle)
	emailBody := fmt.Sprintf("Hi,\n\n%s rejected your document \"%s\".\nReason: %s\n\nPETRODATA Team", reviewerName, docTitle, note)

	if approved {
		notifType = "review_approved"
		title = fmt.Sprintf("Your document \"%s\" was approved", docTitle)
		body = fmt.Sprintf("%s approved your document — it is now published.", reviewerName)
		emailSubject = fmt.Sprintf("[PETRODATA] Document approved: \"%s\"", docTitle)
		emailBody = fmt.Sprintf("Hi,\n\nGreat news! %s approved your document \"%s\" and it is now published.\n\nPETRODATA Team", reviewerName, docTitle)
	}

	s.Notify(ctx, &models.Notification{
		UserID:       submitterID,
		Type:         notifType,
		Title:        title,
		Body:         body,
		ResourceType: "document",
		ResourceID:   &resourceID,
	}, emailSubject, emailBody)
}

// NotifyDocumentUploaded notifies the uploader that their document was saved successfully.
func (s *NotificationService) NotifyDocumentUploaded(ctx context.Context, ownerID, docTitle, docID string) {
	resourceID := docID
	s.Notify(ctx, &models.Notification{
		UserID:       ownerID,
		Type:         "doc_uploaded",
		Title:        "Document uploaded",
		Body:         fmt.Sprintf("\"%s\" has been uploaded and is ready to view.", docTitle),
		ResourceType: "document",
		ResourceID:   &resourceID,
	}, "", "")
}

// NotifyShareAccessed notifies the document owner the first time a share link is used.
func (s *NotificationService) NotifyShareAccessed(ctx context.Context, ownerID, docTitle, docID string) {
	resourceID := docID
	s.Notify(ctx, &models.Notification{
		UserID:       ownerID,
		Type:         "share_accessed",
		Title:        "Your shared document was viewed",
		Body:         fmt.Sprintf("Someone accessed your shared link for \"%s\".", docTitle),
		ResourceType: "document",
		ResourceID:   &resourceID,
	}, "", "")
}

// NotifyWatchers notifies all watchers of a document about an update.
func (s *NotificationService) NotifyWatchers(ctx context.Context, watcherIDs []string, excludeUserID, actorName, docTitle, docID, updateType string) {
	resourceID := docID
	title := fmt.Sprintf("Update on \"%s\"", docTitle)
	body := fmt.Sprintf("%s made a change to a document you're watching.", actorName)
	emailSubject := fmt.Sprintf("[PETRODATA] Update on \"%s\"", docTitle)
	emailBody := fmt.Sprintf("Hi,\n\n%s made an update to \"%s\" which you are watching.\n\nLog in to see the changes.\n\nPETRODATA Team", actorName, docTitle)

	_ = updateType // can extend with specific messages later

	for _, uid := range watcherIDs {
		if uid == excludeUserID {
			continue // don't notify the person who made the change
		}
		s.Notify(ctx, &models.Notification{
			UserID:       uid,
			Type:         "doc_updated",
			Title:        title,
			Body:         body,
			ResourceType: "document",
			ResourceID:   &resourceID,
		}, emailSubject, emailBody)
	}
}

func (s *NotificationService) GetForUser(ctx context.Context, userID string) ([]*models.Notification, error) {
	return s.repo.GetForUser(ctx, userID, 30)
}

func (s *NotificationService) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, userID string) error {
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
