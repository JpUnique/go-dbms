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
	notifSvc     *NotificationService
}

func NewShareService(repo *repository.ShareRepository, docRepo *repository.DocumentRepository, notifSvc *NotificationService) *ShareService {
	return &ShareService{
		repo:         repo,
		documentRepo: docRepo,
		notifSvc:     notifSvc,
	}
}

// ======================================
// CREATE SHARE LINK
// ======================================
func (s *ShareService) Create(
	ctx context.Context,
	userID string,
	docID string,
	permission string,
	password *string,
	expiresAt *string,
) (*models.DocumentShare, error) {

	// ✅ validate permission
	if permission != "view" && permission != "download" {
		return nil, fmt.Errorf("invalid permission")
	}

	// ✅ verify ownership (fast check)
	doc, err := s.documentRepo.GetByID(ctx, docID, userID, false, nil)
	if err != nil || doc == nil {
		return nil, fmt.Errorf("not owner")
	}

	// ✅ generate secure token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("token generation failed: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// ✅ optional password hash
	var passwordHash *string
	if password != nil && *password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), 10)
		if err != nil {
			return nil, fmt.Errorf("password hash failed: %w", err)
		}
		str := string(hash)
		passwordHash = &str
	}

	// ✅ optional expiry
	var expiry *time.Time
	if expiresAt != nil && *expiresAt != "" {
		t, err := time.Parse(time.RFC3339, *expiresAt)
		if err != nil {
			return nil, fmt.Errorf("invalid expiry format")
		}
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

	// ✅ save
	if err := s.repo.Create(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// ======================================
// PUBLIC ACCESS (FAST CHECK)
// ======================================
func (s *ShareService) PublicAccess(
	ctx context.Context,
	token string,
) (map[string]interface{}, error) {

	share, doc, err := s.repo.GetByToken(ctx, token)
	if err != nil || share == nil {
		return nil, fmt.Errorf("not found")
	}

	// ✅ expiry check
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, fmt.Errorf("expired")
	}

	// track view
	firstAccess := share.AccessCount == 0
	go func() {
		_ = s.repo.IncrementAccessCount(context.Background(), share.ID)
		if firstAccess {
			s.notifSvc.NotifyShareAccessed(context.Background(), doc.OwnerID, doc.Title, doc.ID)
		}
	}()

	return map[string]interface{}{
		"document":   doc,
		"permission": share.Permission,
	}, nil
}

// ======================================
// DOWNLOAD (OPTIMIZED + SAFE)
// ======================================
func (s *ShareService) Download(
	ctx context.Context,
	token string,
	password string,
) (string, string, error) {

	// ✅ single DB call (good performance)
	share, doc, err := s.repo.GetByToken(ctx, token)
	if err != nil || share == nil {
		return "", "", fmt.Errorf("not found")
	}

	// ✅ expiry check
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return "", "", fmt.Errorf("link expired")
	}

	// ✅ permission check
	if share.Permission == "view" {
		return "", "", fmt.Errorf("download not allowed")
	}

	// ✅ password check (only if set)
	if share.PasswordHash != nil {
		if err := bcrypt.CompareHashAndPassword(
			[]byte(*share.PasswordHash),
			[]byte(password),
		); err != nil {
			return "", "", fmt.Errorf("invalid password")
		}
	}

	// ✅ generate signed URL
	url, err := storage.GetDownloadURL(share.OwnerName, doc.FileKey)
	if err != nil {
		return "", "", fmt.Errorf("generate download url: %w", err)
	}

	// track download
	go func() { _ = s.repo.IncrementAccessCount(context.Background(), share.ID) }()

	return url, doc.FileName, nil
}

// ======================================
// GET ALL SHARES
// ======================================
func (s *ShareService) GetAll(
	ctx context.Context,
	userID string,
) ([]models.DocumentShare, error) {

	return s.repo.GetAll(ctx, userID)
}

// ======================================
// DELETE SHARE
// ======================================
func (s *ShareService) Delete(
	ctx context.Context,
	shareID string,
	userID string,
) error {

	deleted, err := s.repo.Delete(ctx, shareID, userID)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	if !deleted {
		return fmt.Errorf("share not found")
	}

	return nil
}
