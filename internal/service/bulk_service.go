package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/repository"
)

type BulkService struct {
	repo       *repository.BulkRepository
	folderRepo *repository.FolderRepository
}

func NewBulkService(repo *repository.BulkRepository, folderRepo *repository.FolderRepository) *BulkService {
	return &BulkService{
		repo:       repo,
		folderRepo: folderRepo,
	}
}

func (s *BulkService) Delete(ctx context.Context, userID string, ids []string) (int, error) {
	return s.repo.Delete(ctx, userID, ids)
}

func (s *BulkService) Archive(ctx context.Context, userID string, ids []string) (int, error) {
	return s.repo.Archive(ctx, userID, ids)
}

func (s *BulkService) Move(ctx context.Context, userID string, ids []string, folderID *string) (int, error) {

	if folderID != nil {
		folder, err := s.folderRepo.GetByID(ctx, *folderID, userID)
		if err != nil || folder == nil {
			return 0, fmt.Errorf("invalid folder")
		}
	}

	return s.repo.Move(ctx, userID, ids, folderID)
}

func (s *BulkService) Update(
	ctx context.Context,
	userID string,
	ids []string,
	status *string,
	department *string,
) (int, error) {

	return s.repo.Update(ctx, userID, ids, status, department)
}
