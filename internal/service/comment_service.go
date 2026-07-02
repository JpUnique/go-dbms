package service

import (
	"context"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
)

type CommentService struct {
	repo     *repository.CommentRepository
	userRepo *repository.UserRepository
}

func NewCommentService(repo *repository.CommentRepository, userRepo *repository.UserRepository) *CommentService {
	return &CommentService{repo: repo, userRepo: userRepo}
}

func (s *CommentService) Create(ctx context.Context, documentID, userID, content string) (*models.Comment, error) {
	return s.repo.Create(ctx, documentID, userID, content)
}

func (s *CommentService) GetAll(ctx context.Context, documentID string) ([]*models.Comment, error) {
	return s.repo.GetAll(ctx, documentID)
}

func (s *CommentService) Delete(ctx context.Context, commentID, userID, role string) error {
	return s.repo.Delete(ctx, commentID, userID, role)
}

// GetUserByName looks up a user by display name — used for @mention resolution.
func (s *CommentService) GetUserByName(ctx context.Context, name string) (*models.User, error) {
	return s.userRepo.GetByUsername(ctx, name)
}
