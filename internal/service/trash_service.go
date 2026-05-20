package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
	"github.com/JpUnique/go-dbms/internal/storage"
)

type TrashService struct {
	repo *repository.TrashRepository
}

func NewTrashService(repo *repository.TrashRepository) *TrashService {
	return &TrashService{repo: repo}
}

func (s *TrashService) GetAll(ctx context.Context, userID string) ([]models.Document, error) {

	return s.repo.GetAll(ctx, userID)
}

func (s *TrashService) Restore(ctx context.Context, id string) (*models.Document, error) {

	doc, err := s.repo.Restore(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("trash restore: %w", err)
	}

	if doc == nil {
		return nil, fmt.Errorf("not found")
	}

	return doc, nil
}

func (s *TrashService) Delete(ctx context.Context, id string) error {

	doc, err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("trash delete: %w", err)
	}

	if doc == nil {
		return fmt.Errorf("not found")
	}

	// ✅ delete file from MinIO
	if err := storage.Delete(doc.FileKey); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

func (s *TrashService) Empty(ctx context.Context) (int, error) {

	docs, err := s.repo.Empty(ctx)
	if err != nil {
		return 0, err
	}

	// ✅ delete files
	for _, d := range docs {
		storage.Delete(d.FileKey)
	}

	return len(docs), nil
}
