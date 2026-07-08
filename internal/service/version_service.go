package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
	"github.com/JpUnique/go-dbms/internal/storage"
	"github.com/JpUnique/go-dbms/internal/utils"
)

type DocumentVersionService struct {
	versionRepo  *repository.DocumentVersionRepository
	documentRepo *repository.DocumentRepository
	userRepo     *repository.UserRepository
}

func NewDocumentVersionService(
	versionRepo *repository.DocumentVersionRepository,
	documentRepo *repository.DocumentRepository,
	userRepo *repository.UserRepository,
) *DocumentVersionService {

	return &DocumentVersionService{
		versionRepo:  versionRepo,
		documentRepo: documentRepo,
		userRepo:     userRepo,
	}
}

// resolveScope mirrors DocumentService.resolveScope — admins bypass
// department scoping entirely, everyone else gets their own department
// looked up so version access can be widened to "own uploads OR same
// department", matching document view/edit access.
func (s *DocumentVersionService) resolveScope(ctx context.Context, userID, role string) (isAdmin bool, department *string, err error) {
	if role == "admin" {
		return true, nil, nil
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, nil, fmt.Errorf("version service resolve scope: %w", err)
	}
	if user == nil || user.Department == nil || *user.Department == "" {
		return false, nil, nil
	}

	return false, user.Department, nil
}

// ======================================
// GET VERSIONS ✅
func (s *DocumentVersionService) GetVersions(
	ctx context.Context,
	docID string,
	userID string,
	role string,
) ([]models.DocumentVersion, error) {

	isAdmin, department, err := s.resolveScope(ctx, userID, role)
	if err != nil {
		return nil, err
	}

	return s.versionRepo.GetByDocument(ctx, docID, userID, isAdmin, department)
}

// ======================================
// UPLOAD NEW VERSION ✅ FIXED
func (s *DocumentVersionService) UploadVersion(
	ctx context.Context,
	docID string,
	userID string,
	role string,
	username string,
	file []byte,
	fileName string,
	fileType string,
	changeNote string,
) (*models.DocumentVersion, error) {

	isAdmin, department, err := s.resolveScope(ctx, userID, role)
	if err != nil {
		return nil, err
	}

	tx, err := s.versionRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	// ✅ SAFE ROLLBACK HANDLING
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// ✅ LOCK DOCUMENT
	currentVersion, err := s.versionRepo.GetCurrentVersionForUpdate(ctx, tx, docID, userID, isAdmin, department)
	if err != nil {
		return nil, err
	}

	newVersion := currentVersion + 1

	// ✅ Upload to MinIO (correct bucket)
	fileKey := storage.GenerateFileKey(fileName)

	if err = storage.Upload(username, fileKey, file, fileType); err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	// ✅ CREATE NEW VERSION ROW
	version := &models.DocumentVersion{
		DocumentID: docID,
		Version:    newVersion,
		FileKey:    fileKey,
		FileSize:   int64(len(file)),
		UploadedBy: utils.StrPtr(userID),
	}

	if changeNote != "" {
		version.ChangeNote = &changeNote
	}

	if err = s.versionRepo.Create(ctx, tx, version); err != nil {
		return nil, err
	}

	// ✅ UPDATE MAIN DOCUMENT
	if err = s.documentRepo.UpdateLatestVersion(
		ctx,
		tx,
		docID,
		userID,
		newVersion,
		fileKey,
		fileName,
		fileType,
		int64(len(file)),
	); err != nil {
		return nil, err
	}

	// ✅ COMMIT
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return version, nil
}

// ======================================
// DOWNLOAD VERSION ✅ FIXED
func (s *DocumentVersionService) DownloadVersion(
	ctx context.Context,
	docID string,
	versionID string,
	userID string,
	role string,
) (string, string, error) {

	isAdmin, department, err := s.resolveScope(ctx, userID, role)
	if err != nil {
		return "", "", err
	}

	version, fileName, ownerName, err :=
		s.versionRepo.GetByID(ctx, docID, versionID, userID, isAdmin, department)

	if err != nil {
		return "", "", fmt.Errorf("download version: %w", err)
	}

	if version == nil {
		return "", "", utils.ErrNotFound
	}

	// ✅ Use correct bucket owner
	url, err := storage.GetDownloadURL(ownerName, version.FileKey)
	if err != nil {
		return "", "", fmt.Errorf("generate url: %w", err)
	}

	return url, fileName, nil
}
