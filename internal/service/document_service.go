package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
	"github.com/JpUnique/go-dbms/internal/storage"
	"github.com/JpUnique/go-dbms/internal/utils"
)

type DocumentService struct {
	repo *repository.DocumentRepository
}

func NewDocumentService(repo *repository.DocumentRepository) *DocumentService {
	return &DocumentService{repo: repo}
}

// ==============================
// UPLOAD DOCUMENT
// ==============================
func (s *DocumentService) Upload(
	ctx context.Context,
	file []byte,
	fileName string,
	fileType string,
	userID string,
) (*models.Document, error) {

	// ✅ generate unique key for storage
	fileKey := storage.GenerateFileKey(fileName)

	// ✅ upload to MinIO
	if err := storage.Upload(fileKey, file, fileType); err != nil {
		return nil, fmt.Errorf("document service upload: storage upload: %w", err)
	}

	doc := &models.Document{
		Title:    fileName,
		FileName: fileName,
		FileKey:  fileKey,
		FileType: fileType,
		FileSize: int64(len(file)),
		OwnerID:  userID,
	}

	// ✅ save metadata in DB
	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("document service upload: save document: %w", err)
	}

	return doc, nil
}

// ==============================
// GET ALL DOCUMENTS
// ==============================

func (s *DocumentService) GetAll(
	ctx context.Context,
	userID string,
) ([]models.Document, error) {

	docs, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("document service get all: %w", err)
	}

	return docs, nil
}

// ==============================
// GET DOCUMENT BY ID
// ==============================
func (s *DocumentService) GetByID(
	ctx context.Context,
	docID string,
	userID string,
) (*models.Document, error) {

	doc, err := s.repo.GetByID(ctx, docID, userID)
	if err != nil {
		return nil, fmt.Errorf("document service get by id: %w", err)
	}

	if doc == nil {
		return nil, utils.ErrNotFound
	}

	return doc, nil
}

// ==============================
// DOWNLOAD DOCUMENT
// ==============================
func (s *DocumentService) GetDownloadURL(
	ctx context.Context,
	docID string,
	userID string,
) (string, string, error) {

	doc, err := s.repo.GetByID(ctx, docID, userID)
	if err != nil {
		return "", "", fmt.Errorf("document service download: %w", err)
	}

	if doc == nil {
		return "", "", utils.ErrNotFound
	}

	// generate presigned URL (MinIO)
	url, err := storage.GetDownloadURL(doc.FileKey)
	if err != nil {
		return "", "", fmt.Errorf("document service download: generate url: %w", err)
	}

	return url, doc.FileName, nil
}

// ==============================
// DELETE DOCUMENT
// ==============================
func (s *DocumentService) Update(
	ctx context.Context,
	docID string,
	userID string,
	title *string,
	status *string,
	isStarred *bool,
) (*models.Document, error) {

	doc, err := s.repo.Update(
		ctx,
		docID,
		userID,
		title,
		status,
		isStarred,
	)
	if err != nil {
		return nil, fmt.Errorf("document service update: %w", err)
	}

	if doc == nil {
		return nil, utils.ErrNotFound
	}

	return doc, nil
}

// ==============================
// DELETE DOCUMENT
// ==============================
func (s *DocumentService) Delete(
	ctx context.Context,
	docID string,
	userID string,
) error {

	doc, err := s.repo.Delete(ctx, docID, userID)
	if err != nil {
		return fmt.Errorf("document service delete: %w", err)
	}

	if doc == nil {
		return utils.ErrNotFound
	}

	// ✅ delete file from MinIO
	if err := storage.Delete(doc.FileKey); err != nil {
		return fmt.Errorf("document service delete file: %w", err)
	}

	return nil
}

// ==============================
// TOGGLE STAR
// ==============================
func (s *DocumentService) ToggleStar(
	ctx context.Context,
	docID string,
	userID string,
) (bool, error) {

	isStarred, err := s.repo.ToggleStar(ctx, docID, userID)
	if err != nil {
		return false, fmt.Errorf("document service toggle star: %w", err)
	}

	return isStarred, nil
}

func (s *DocumentService) GetAllByFilter(
	ctx context.Context,
	userID string,
	query models.DocumentQuery,
) ([]models.Document, int, error) {

	docs, total, err := s.repo.GetByUserWithFilter(
		ctx,
		userID,
		query,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("document service get all: %w", err)
	}

	return docs, total, nil
}
