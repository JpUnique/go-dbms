package service

import (
	"context"
	"fmt"
	"sync"

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

// ======================================
// GET ALL TRASH
// ======================================
func (s *TrashService) GetAll(
	ctx context.Context,
	userID string,
) ([]models.Document, error) {

	return s.repo.GetAll(ctx, userID)
}

// ======================================
// RESTORE DOCUMENT ✅ FIXED
// ======================================
func (s *TrashService) Restore(
	ctx context.Context,
	id string,
	userID string,
) (*models.Document, error) {

	// ✅ FIX: pass userID correctly
	doc, err := s.repo.Restore(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("trash restore: %w", err)
	}

	if doc == nil {
		return nil, fmt.Errorf("not found")
	}

	return doc, nil
}

// ======================================
// DELETE SINGLE DOCUMENT ✅ FIXED
// ======================================
func (s *TrashService) Delete(
	ctx context.Context,
	id string,
	userID string,
	username string,
) error {

	doc, err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("trash delete: %w", err)
	}

	if doc == nil {
		return fmt.Errorf("not found")
	}

	// ✅ delete file from MinIO (correct user bucket)
	if err := storage.Delete(username, doc.FileKey); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

// ======================================
// EMPTY TRASH (SCALABLE) ✅ READY
// ======================================
func (s *TrashService) Empty(
	ctx context.Context,
	userID string,
	username string,
) (int, error) {

	docs, err := s.repo.Empty(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("empty trash: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(docs))

	// ✅ concurrent deletion for performance
	for _, d := range docs {
		wg.Add(1)

		go func(fileKey string) {
			defer wg.Done()

			if err := storage.Delete(username, fileKey); err != nil {
				errChan <- err
			}
		}(d.FileKey)
	}

	wg.Wait()
	close(errChan)

	// ✅ detect errors (fail fast)
	for err := range errChan {
		if err != nil {
			return len(docs), fmt.Errorf("partial delete failed: %w", err)
		}
	}

	return len(docs), nil
}
