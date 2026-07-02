package repository

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserShareRepository struct {
	db *pgxpool.Pool
}

func NewUserShareRepository(db *pgxpool.Pool) *UserShareRepository {
	return &UserShareRepository{db: db}
}

// Grant gives userID access to docID, or updates the permission level if a
// grant already exists.
func (r *UserShareRepository) Grant(ctx context.Context, docID, userID, permission, sharedBy string) error {
	_, err := r.db.Exec(ctx, `
    INSERT INTO document_user_shares (document_id, user_id, permission, shared_by)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT (document_id, user_id)
    DO UPDATE SET permission = EXCLUDED.permission, shared_by = EXCLUDED.shared_by
  `, docID, userID, permission, sharedBy)
	if err != nil {
		return fmt.Errorf("user share repo grant: %w", err)
	}
	return nil
}

func (r *UserShareRepository) Revoke(ctx context.Context, docID, userID string) error {
	cmdTag, err := r.db.Exec(ctx, `
    DELETE FROM document_user_shares WHERE document_id = $1 AND user_id = $2
  `, docID, userID)
	if err != nil {
		return fmt.Errorf("user share repo revoke: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListRecipients returns everyone docID has been shared with.
func (r *UserShareRepository) ListRecipients(ctx context.Context, docID string) ([]models.ShareRecipient, error) {
	rows, err := r.db.Query(ctx, `
    SELECT u.id, u.name, u.email, s.permission, s.created_at
    FROM document_user_shares s
    JOIN users u ON u.id = s.user_id
    WHERE s.document_id = $1
    ORDER BY s.created_at DESC
  `, docID)
	if err != nil {
		return nil, fmt.Errorf("user share repo list recipients: %w", err)
	}
	defer rows.Close()

	recipients := make([]models.ShareRecipient, 0)
	for rows.Next() {
		var rec models.ShareRecipient
		if err := rows.Scan(&rec.UserID, &rec.Name, &rec.Email, &rec.Permission, &rec.SharedAt); err != nil {
			return nil, fmt.Errorf("user share repo list recipients scan: %w", err)
		}
		recipients = append(recipients, rec)
	}
	return recipients, rows.Err()
}

// ListSharedWithUser returns every document shared directly with userID.
func (r *UserShareRepository) ListSharedWithUser(ctx context.Context, userID string) ([]models.SharedDocument, error) {
	rows, err := r.db.Query(ctx, `
    SELECT d.id, d.title, d.description, d.file_name, d.file_key, d.file_type, d.file_size,
           d.folder_id, d.owner_id, d.department, d.status, d.version, d.is_starred,
           d.last_accessed, d.created_at, d.updated_at,
           COALESCE(u.name, 'Unknown'), s.permission, s.created_at
    FROM document_user_shares s
    JOIN documents d ON d.id = s.document_id AND d.status != 'archived'
    LEFT JOIN users u ON u.id = s.shared_by
    WHERE s.user_id = $1
    ORDER BY s.created_at DESC
  `, userID)
	if err != nil {
		return nil, fmt.Errorf("user share repo list shared with user: %w", err)
	}
	defer rows.Close()

	docs := make([]models.SharedDocument, 0)
	for rows.Next() {
		var d models.SharedDocument
		if err := rows.Scan(
			&d.ID, &d.Title, &d.Description, &d.FileName, &d.FileKey, &d.FileType, &d.FileSize,
			&d.FolderID, &d.OwnerID, &d.Department, &d.Status, &d.Version, &d.IsStarred,
			&d.LastAccess, &d.CreatedAt, &d.UpdatedAt,
			&d.SharedByName, &d.Permission, &d.SharedAt,
		); err != nil {
			return nil, fmt.Errorf("user share repo list shared with user scan: %w", err)
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}
