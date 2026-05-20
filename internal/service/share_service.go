package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
	"github.com/JpUnique/go-dbms/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type ShareService struct {
	repo         *repository.ShareRepository
	documentRepo *repository.DocumentRepository
}

func NewShareService(repo *repository.ShareRepository, docRepo *repository.DocumentRepository) *ShareService {
	return &ShareService{
		repo:         repo,
		documentRepo: docRepo,
	}
}

func (s *ShareService) Create(
	ctx context.Context,
	userID string,
	docID string,
	permission string,
	password *string,
	expiresAt *string,
) (*models.DocumentShare, error) {

	// ✅ verify ownership
	doc, err := s.documentRepo.GetByID(ctx, docID, userID)
	if err != nil || doc == nil {
		return nil, fmt.Errorf("not owner")
	}

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	var passwordHash *string
	if password != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte(*password), 10)
		str := string(hash)
		passwordHash = &str
	}

	var expiry *time.Time
	if expiresAt != nil {
		t, _ := time.Parse(time.RFC3339, *expiresAt)
		expiry = &t
	}

	share := &models.DocumentShare{
		DocumentID:   docID,
		ShareToken:   token,
		SharedBy:     userID,
		Permission:   permission,
		PasswordHash: passwordHash,
		ExpiresAt:    expiry,
	}

	if err := s.repo.Create(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

func (s *ShareService) PublicAccess(ctx context.Context, token string) (map[string]interface{}, error) {

	share, doc, err := s.repo.GetByToken(ctx, token)
	if err != nil || share == nil {
		return nil, fmt.Errorf("not found")
	}

	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, fmt.Errorf("expired")
	}

	return map[string]interface{}{
		"document":   doc,
		"permission": share.Permission,
	}, nil
}

func (s *ShareService) Download(
	ctx context.Context,
	token string,
	password string,
) (string, string, error) {

	share, doc, err := s.repo.GetByToken(ctx, token)
	if err != nil || share == nil {
		return "", "", fmt.Errorf("not found")
	}

	if share.Permission == "view" {
		return "", "", fmt.Errorf("download not allowed")
	}

	if share.PasswordHash != nil {
		if err := bcrypt.CompareHashAndPassword(
			[]byte(*share.PasswordHash),
			[]byte(password),
		); err != nil {
			return "", "", fmt.Errorf("invalid password")
		}
	}

	url, _ := storage.GetDownloadURL(doc.FileKey)

	return url, doc.FileName, nil
}

func (s *ShareService) GetAll(
	ctx context.Context,
	userID string,
) ([]models.DocumentShare, error) {

	shares, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("share service get all: %w", err)
	}

	return shares, nil
}

func (s *ShareService) Delete(
	ctx context.Context,
	shareID string,
	userID string,
) error {

	deleted, err := s.repo.Delete(ctx, shareID, userID)
	if err != nil {
		return fmt.Errorf("share service delete: %w", err)
	}

	if !deleted {
		return fmt.Errorf("share not found")
	}

	return nil
}
