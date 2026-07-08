package service

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserShareService struct {
	repo         *repository.UserShareRepository
	documentRepo *repository.DocumentRepository
	notifSvc     *NotificationService
	db           *pgxpool.Pool
}

func NewUserShareService(
	repo *repository.UserShareRepository,
	documentRepo *repository.DocumentRepository,
	notifSvc *NotificationService,
	db *pgxpool.Pool,
) *UserShareService {
	return &UserShareService{
		repo:         repo,
		documentRepo: documentRepo,
		notifSvc:     notifSvc,
		db:           db,
	}
}

// requireOwner loads the document and confirms callerID actually owns it.
// DocumentRepository.GetByID also permits shared viewers through, so
// ownership must be checked explicitly here rather than treating "found" as
// "owns it" — only the owner may grant/revoke/list who a document is shared with.
func (s *UserShareService) requireOwner(ctx context.Context, docID, callerID string) (*models.Document, error) {
	doc, err := s.documentRepo.GetByID(ctx, docID, callerID, false, nil)
	if err != nil {
		return nil, fmt.Errorf("user share service: get document: %w", err)
	}
	if doc == nil || doc.OwnerID != callerID {
		return nil, utils.ErrUnauthorized
	}
	return doc, nil
}

// Grant gives recipientID view/download access to docID and notifies them.
func (s *UserShareService) Grant(ctx context.Context, docID, ownerID, sharerName, recipientID, permission string) error {

	if recipientID == ownerID {
		return fmt.Errorf("cannot share a document with yourself")
	}

	doc, err := s.requireOwner(ctx, docID, ownerID)
	if err != nil {
		return err
	}

	if err := s.repo.Grant(ctx, docID, recipientID, permission, ownerID); err != nil {
		return fmt.Errorf("user share service grant: %w", err)
	}

	go s.notifSvc.NotifyDocumentShared(context.Background(), recipientID, sharerName, doc.Title, docID)
	go utils.LogAudit(context.Background(), s.db, utils.AuditEntry{
		UserID: &ownerID, Action: "share_document_user", ResourceType: "document", ResourceID: &docID,
	})

	return nil
}

func (s *UserShareService) Revoke(ctx context.Context, docID, ownerID, recipientID string) error {
	if _, err := s.requireOwner(ctx, docID, ownerID); err != nil {
		return err
	}
	if err := s.repo.Revoke(ctx, docID, recipientID); err != nil {
		return fmt.Errorf("user share service revoke: %w", err)
	}
	return nil
}

func (s *UserShareService) ListRecipients(ctx context.Context, docID, ownerID string) ([]models.ShareRecipient, error) {
	if _, err := s.requireOwner(ctx, docID, ownerID); err != nil {
		return nil, err
	}
	return s.repo.ListRecipients(ctx, docID)
}

func (s *UserShareService) SharedWithMe(ctx context.Context, userID string) ([]models.SharedDocument, error) {
	return s.repo.ListSharedWithUser(ctx, userID)
}
