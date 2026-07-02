package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
)

type ReviewService struct {
	repo     *repository.ReviewRepository
	docRepo  *repository.DocumentRepository
	userRepo *repository.UserRepository
}

func NewReviewService(
	repo *repository.ReviewRepository,
	docRepo *repository.DocumentRepository,
	userRepo *repository.UserRepository,
) *ReviewService {
	return &ReviewService{repo: repo, docRepo: docRepo, userRepo: userRepo}
}

// Submit sets the document to pending_review and creates a review record.
// Returns the review and the document owner's ID (for notifying admins).
func (s *ReviewService) Submit(ctx context.Context, documentID, submitterID string) (*models.DocumentReview, error) {
	// Update document status
	if err := s.docRepo.UpdateStatus(ctx, documentID, submitterID, "pending_review"); err != nil {
		return nil, fmt.Errorf("review service submit status: %w", err)
	}
	return s.repo.Submit(ctx, documentID, submitterID)
}

// Approve approves a document review and publishes the document.
func (s *ReviewService) Approve(ctx context.Context, documentID, reviewerID string, note *string) (*models.DocumentReview, error) {
	reviewID, err := s.repo.GetPendingReviewID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("no pending review found for document")
	}
	if err := s.repo.Decide(ctx, reviewID, reviewerID, "approved", note); err != nil {
		return nil, err
	}
	// Publish the document
	if err := s.docRepo.UpdateStatus(ctx, documentID, reviewerID, "published"); err != nil {
		return nil, fmt.Errorf("review service approve: %w", err)
	}
	reviews, err := s.repo.GetByDocument(ctx, documentID)
	if err != nil || len(reviews) == 0 {
		return nil, err
	}
	return reviews[0], nil
}

// Reject rejects a document review and moves it back to draft.
func (s *ReviewService) Reject(ctx context.Context, documentID, reviewerID string, note *string) (*models.DocumentReview, error) {
	reviewID, err := s.repo.GetPendingReviewID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("no pending review found for document")
	}
	if err := s.repo.Decide(ctx, reviewID, reviewerID, "rejected", note); err != nil {
		return nil, err
	}
	if err := s.docRepo.UpdateStatus(ctx, documentID, reviewerID, "draft"); err != nil {
		return nil, fmt.Errorf("review service reject: %w", err)
	}
	reviews, err := s.repo.GetByDocument(ctx, documentID)
	if err != nil || len(reviews) == 0 {
		return nil, err
	}
	return reviews[0], nil
}

func (s *ReviewService) GetByDocument(ctx context.Context, documentID string) ([]*models.DocumentReview, error) {
	return s.repo.GetByDocument(ctx, documentID)
}

func (s *ReviewService) PendingQueue(ctx context.Context) ([]*models.DocumentReview, error) {
	return s.repo.PendingQueue(ctx)
}
