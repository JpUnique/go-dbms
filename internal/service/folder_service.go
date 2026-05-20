package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
	"github.com/JpUnique/go-dbms/internal/utils"
)

type FolderService struct {
	repo *repository.FolderRepository
}

// constructor
func NewFolderService(repo *repository.FolderRepository) *FolderService {
	return &FolderService{repo: repo}
}
func (s *FolderService) GetAllFolders(
	ctx context.Context,
	userID string,
	parentID string,
	limit int,
	offset int,
) ([]models.Folder, error) {

	folders, err := s.repo.GetAllFolders(ctx, userID, parentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("folder service get all: %w", err)
	}

	return folders, nil
}
func (s *FolderService) GetByID(
	ctx context.Context,
	folderID string,
	userID string,
) (*models.Folder, error) {

	folder, err := s.repo.GetByID(ctx, folderID, userID)
	if err != nil {
		return nil, fmt.Errorf("folder service get by id: %w", err)
	}

	if folder == nil {
		return nil, utils.ErrNotFound
	}

	return folder, nil
}
func (s *FolderService) CreateFolder(
	ctx context.Context,
	userID string,
	name string,
	parentID *string,
	department *string,
) (*models.Folder, error) {

	// validate parent folder ownership
	if parentID != nil && *parentID != "" {

		parent, err := s.repo.GetByID(ctx, *parentID, userID)
		if err != nil {
			return nil, fmt.Errorf("folder service validate parent: %w", err)
		}

		if parent == nil {
			return nil, fmt.Errorf("invalid parent folder")
		}
	}

	folder := &models.Folder{
		Name:       name,
		OwnerID:    userID,
		ParentID:   parentID,
		Department: department,
	}

	err := s.repo.CreateFolder(ctx, folder)
	if err != nil {
		return nil, fmt.Errorf("folder service create: %w", err)
	}

	return folder, nil
}

func (s *FolderService) UpdateFolder(
	ctx context.Context,
	folderID string,
	userID string,
	name *string,
	parentID *string,
	department *string,
) (*models.Folder, error) {

	// ✅ prevent self-parenting
	if parentID != nil && *parentID == folderID {
		return nil, utils.ErrInvalidInput
	}

	// ✅ validate parent ownership
	if parentID != nil && *parentID != "" {

		parent, err := s.repo.GetByID(ctx, *parentID, userID)
		if err != nil {
			return nil, fmt.Errorf("folder service validate parent: %w", err)
		}

		if parent == nil {
			return nil, utils.ErrInvalidInput
		}
	}

	folder, err := s.repo.Update(
		ctx,
		folderID,
		userID,
		name,
		parentID,
		department,
	)
	if err != nil {
		return nil, fmt.Errorf("folder service update: %w", err)
	}

	if folder == nil {
		return nil, utils.ErrNotFound
	}

	return folder, nil
}

func (s *FolderService) DeleteFolder(
	ctx context.Context,
	folderID string,
	userID string,
) error {

	exists, err := s.repo.Delete(ctx, folderID, userID)
	if err != nil {
		return fmt.Errorf("folder service delete: %w", err)
	}

	if !exists {
		return utils.ErrNotFound
	}

	return nil
}
