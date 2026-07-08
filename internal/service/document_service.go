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
	repo     *repository.DocumentRepository
	userRepo *repository.UserRepository
}

func NewDocumentService(repo *repository.DocumentRepository, userRepo *repository.UserRepository) *DocumentService {
	return &DocumentService{repo: repo, userRepo: userRepo}
}

// resolveScope turns a JWT role into the (isAdmin, department) pair the repo
// layer needs: admins bypass department scoping entirely (department stays
// nil, isAdmin does the work); everyone else gets their own department
// looked up so they can be widened to "own uploads OR same department".
// department stays nil if the user simply has none set.
func (s *DocumentService) resolveScope(ctx context.Context, userID, role string) (isAdmin bool, department *string, err error) {
	if role == "admin" {
		return true, nil, nil
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, nil, fmt.Errorf("document service resolve scope: %w", err)
	}
	if user == nil || user.Department == nil || *user.Department == "" {
		return false, nil, nil
	}

	return false, user.Department, nil
}

// ==============================
// UPLOAD DOCUMENT
// ==============================
type UploadMeta struct {
	Title       string
	Description string
	FolderID    string
	Department  string
	Status      string
}

func (s *DocumentService) Upload(
	ctx context.Context,
	file []byte,
	fileName string,
	fileType string,
	userID string,
	meta UploadMeta,
) (*models.Document, error) {

	fileKey := storage.GenerateFileKey(fileName)

	if err := storage.Upload(userID, fileKey, file, fileType); err != nil {
		return nil, fmt.Errorf("document service upload: storage upload: %w", err)
	}

	title := meta.Title
	if title == "" {
		title = fileName
	}

	doc := &models.Document{
		Title:    title,
		FileName: fileName,
		FileKey:  fileKey,
		FileType: fileType,
		FileSize: int64(len(file)),
		OwnerID:  userID,
		Status:   meta.Status,
	}
	if meta.Description != "" {
		doc.Description = &meta.Description
	}
	if meta.FolderID != "" {
		doc.FolderID = &meta.FolderID
	}
	if meta.Department != "" {
		doc.Department = &meta.Department
	}

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
	role string,
) (*models.Document, error) {

	isAdmin, department, err := s.resolveScope(ctx, userID, role)
	if err != nil {
		return nil, fmt.Errorf("document service get by id: %w", err)
	}

	doc, err := s.repo.GetByID(ctx, docID, userID, isAdmin, department)
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
	role string,
) (string, string, error) {

	isAdmin, department, err := s.resolveScope(ctx, userID, role)
	if err != nil {
		return "", "", fmt.Errorf("document service download: %w", err)
	}

	doc, err := s.repo.GetByIDForDownload(ctx, docID, userID, isAdmin, department)
	if err != nil {
		return "", "", fmt.Errorf("document service download: %w", err)
	}

	if doc == nil {
		return "", "", utils.ErrNotFound
	}

	// The file lives in the OWNER's bucket regardless of who's downloading
	// it (a shared, non-owner user has their own separate bucket).
	url, err := storage.GetDownloadURL(doc.OwnerID, doc.FileKey)
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
	role string,
	title *string,
	status *string,
	isStarred *bool,
	folderID **string,
) (*models.Document, error) {

	isAdmin, department, err := s.resolveScope(ctx, userID, role)
	if err != nil {
		return nil, fmt.Errorf("document service update: %w", err)
	}

	doc, err := s.repo.Update(
		ctx,
		docID,
		userID,
		isAdmin,
		department,
		title,
		status,
		isStarred,
		folderID,
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
// Delete moves the document to Trash. The underlying file is intentionally
// left in storage — it's only removed once the document is permanently
// deleted from Trash (TrashService.Delete/Empty).
func (s *DocumentService) Delete(
	ctx context.Context,
	docID string,
	userID string,
	role string,
) error {

	doc, err := s.repo.Delete(ctx, docID, userID, role == "admin")
	if err != nil {
		return fmt.Errorf("document service delete: %w", err)
	}

	if doc == nil {
		return utils.ErrNotFound
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
	role string,
) (bool, error) {

	isAdmin, department, err := s.resolveScope(ctx, userID, role)
	if err != nil {
		return false, fmt.Errorf("document service toggle star: %w", err)
	}

	isStarred, err := s.repo.ToggleStar(ctx, docID, userID, isAdmin, department)
	if err != nil {
		return false, fmt.Errorf("document service toggle star: %w", err)
	}

	return isStarred, nil
}

func (s *DocumentService) GetAllByFilter(
	ctx context.Context,
	userID string,
	role string,
	query models.DocumentQuery,
) ([]models.DocumentWithMeta, int, error) {

	_, department, err := s.resolveScope(ctx, userID, role)
	if err != nil {
		return nil, 0, fmt.Errorf("document service get all: %w", err)
	}

	docs, total, err := s.repo.GetByUserWithFilter(ctx, userID, department, query)
	if err != nil {
		return nil, 0, fmt.Errorf("document service get all: %w", err)
	}

	return docs, total, nil
}

// ==============================
// ADMIN: BROWSE DOCUMENTS BY DEPARTMENT
// ==============================
func (s *DocumentService) GetByDepartment(
	ctx context.Context,
	department string,
	page, limit int,
) ([]models.DocumentWithMeta, int, error) {

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	} else if limit > 200 {
		limit = 200
	}

	docs, total, err := s.repo.GetByDepartment(ctx, department, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("document service get by department: %w", err)
	}

	return docs, total, nil
}

// CountByDepartment powers the Departments admin page's stat cards.
func (s *DocumentService) CountByDepartment(ctx context.Context) (map[string]int, error) {
	counts, err := s.repo.CountByDepartment(ctx)
	if err != nil {
		return nil, fmt.Errorf("document service count by department: %w", err)
	}
	return counts, nil
}
