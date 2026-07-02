package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ShareRepository struct {
	db *pgxpool.Pool
}

// constructor
func NewShareRepository(db *pgxpool.Pool) *ShareRepository {
	return &ShareRepository{db: db}
}

func (r *ShareRepository) Create(
	ctx context.Context,
	share *models.DocumentShare,
) error {

	query := `
    INSERT INTO document_shares (
        document_id, share_token, shared_by,
        permission, password_hash, expires_at
    )
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id, created_at
    `

	err := r.db.QueryRow(ctx, query,
		share.DocumentID,
		share.ShareToken,
		share.SharedBy,
		share.Permission,
		share.PasswordHash,
		share.ExpiresAt,
	).Scan(
		&share.ID,
		&share.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("share repo create: %w", err)
	}

	return nil
}

func (r *ShareRepository) GetAll(
	ctx context.Context,
	userID string,
) ([]models.DocumentShare, error) {

	query := `
    SELECT id, document_id, share_token, shared_by,
           permission, password_hash, expires_at,
           access_count, created_at
    FROM document_shares
    WHERE shared_by = $1
    ORDER BY created_at DESC
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("share repo get all: %w", err)
	}
	defer rows.Close()

	shares := []models.DocumentShare{}

	for rows.Next() {
		var s models.DocumentShare

		err := rows.Scan(
			&s.ID,
			&s.DocumentID,
			&s.ShareToken,
			&s.SharedBy,
			&s.Permission,
			&s.PasswordHash,
			&s.ExpiresAt,
			&s.AccessCount,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("share repo scan: %w", err)
		}

		shares = append(shares, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("share repo rows error: %w", err)
	}

	return shares, nil
}

func (r *ShareRepository) Delete(
	ctx context.Context,
	shareID string,
	userID string,
) (bool, error) {

	query := `
    DELETE FROM document_shares
    WHERE id = $1 AND shared_by = $2
    `

	cmdTag, err := r.db.Exec(ctx, query, shareID, userID)
	if err != nil {
		return false, fmt.Errorf("share repo delete: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return false, nil
	}

	return true, nil
}

func (r *ShareRepository) GetByToken(
	ctx context.Context,
	token string,
) (*models.DocumentShare, *models.Document, error) {

	query := `
    SELECT
        s.id, s.document_id, s.share_token, s.shared_by,
        s.permission, s.password_hash, s.expires_at,
        s.access_count, s.created_at,

        d.id, d.title, d.file_name, d.file_type, d.file_size, d.file_key, d.owner_id,

        u.name

    FROM document_shares s
    JOIN documents d ON s.document_id = d.id
    JOIN users u ON d.owner_id = u.id

    WHERE s.share_token = $1
    `

	var share models.DocumentShare
	var doc models.Document

	err := r.db.QueryRow(ctx, query, token).Scan(
		&share.ID,
		&share.DocumentID,
		&share.ShareToken,
		&share.SharedBy,
		&share.Permission,
		&share.PasswordHash,
		&share.ExpiresAt,
		&share.AccessCount,
		&share.CreatedAt,

		&doc.ID,
		&doc.Title,
		&doc.FileName,
		&doc.FileType,
		&doc.FileSize,
		&doc.FileKey,
		&doc.OwnerID,

		&share.OwnerName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("share repo get by token: %w", err)
	}

	return &share, &doc, nil
}

func (r *ShareRepository) IncrementAccessCount(
	ctx context.Context,
	shareID string,
) error {

	query := `
    UPDATE document_shares
    SET access_count = access_count + 1
    WHERE id = $1
    `

	_, err := r.db.Exec(ctx, query, shareID)
	if err != nil {
		return fmt.Errorf("share repo increment access: %w", err)
	}

	return nil
}
