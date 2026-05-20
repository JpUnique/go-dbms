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
}

func NewDocumentVersionService(
	versionRepo *repository.DocumentVersionRepository,
	documentRepo *repository.DocumentRepository,
) *DocumentVersionService {

	return &DocumentVersionService{
		versionRepo:  versionRepo,
		documentRepo: documentRepo,
	}
}

func (s *DocumentVersionService) GetVersions(
	ctx context.Context,
	docID string,
	userID string,
) ([]models.DocumentVersion, error) {

	versions, err := s.versionRepo.GetByDocument(ctx, docID, userID)
	if err != nil {
		return nil, fmt.Errorf("version service get versions: %w", err)
	}

	return versions, nil
}

func (s *DocumentVersionService) UploadVersion(
	ctx context.Context,
	docID string,
	userID string,
	file []byte,
	fileName string,
	fileType string,
	changeNote string,
) (*models.DocumentVersion, error) {

	//  start transaction
	tx, err := s.versionRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("version service begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// lock document row (FOR UPDATE)
	currentVersion, err := s.versionRepo.GetCurrentVersionForUpdate(ctx, tx, docID, userID)
	if err != nil {
		return nil, fmt.Errorf("version service lock document: %w", err)
	}

	if currentVersion == 0 {
		return nil, utils.ErrNotFound
	}

	newVersion := currentVersion + 1

	//  upload new file to MinIO
	fileKey := storage.GenerateFileKey(fileName)

	if err := storage.Upload(fileKey, file, fileType); err != nil {
		return nil, fmt.Errorf("version service upload file: %w", err)
	}

	//  insert new version
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

	err = s.versionRepo.Create(ctx, tx, version)
	if err != nil {
		return nil, fmt.Errorf("version service insert version: %w", err)
	}

	// update main document
	err = s.documentRepo.UpdateLatestVersion(
		ctx,
		tx,
		docID,
		userID,
		newVersion,
		fileKey,
		fileName,
		fileType,
		int64(len(file)),
	)
	if err != nil {
		return nil, fmt.Errorf("version service update document: %w", err)
	}

	// commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("version service commit: %w", err)
	}

	return version, nil
}

func (s *DocumentVersionService) DownloadVersion(
	ctx context.Context,
	docID string,
	versionID string,
	userID string,
) (string, string, error) {

	version, fileName, err :=
		s.versionRepo.GetByID(ctx, docID, versionID, userID)

	if err != nil {
		return "", "", fmt.Errorf("version service download: %w", err)
	}

	if version == nil {
		return "", "", utils.ErrNotFound
	}

	// generate presigned URL
	url, err := storage.GetDownloadURL(version.FileKey)
	if err != nil {
		return "", "", fmt.Errorf("version service generate url: %w", err)
	}

	return url, fileName, nil
}
