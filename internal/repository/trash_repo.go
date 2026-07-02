package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TrashRepository struct {
	db *pgxpool.Pool
}

func NewTrashRepository(db *pgxpool.Pool) *TrashRepository {
	return &TrashRepository{db: db}
}

// ======================================
// GET ALL TRASH (ALREADY GOOD ✅)
// ======================================
func (r *TrashRepository) GetAll(
	ctx context.Context,
	userID string,
) ([]models.Document, error) {

	query := `
    SELECT id, title, file_name, file_key, file_type, file_size,
           owner_id, status, version, is_starred,
           created_at, updated_at
    FROM documents
    WHERE status = 'archived' AND owner_id = $1
    ORDER BY updated_at DESC
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("trash repo get all: %w", err)
	}
	defer rows.Close()

	docs := make([]models.Document, 0)

	for rows.Next() {
		var d models.Document

		if err := rows.Scan(
			&d.ID,
			&d.Title,
			&d.FileName,
			&d.FileKey,
			&d.FileType,
			&d.FileSize,
			&d.OwnerID,
			&d.Status,
			&d.Version,
			&d.IsStarred,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, err
		}

		docs = append(docs, d)
	}

	return docs, nil
}

// ======================================
// RESTORE DOCUMENT ✅ FIXED
// ======================================
func (r *TrashRepository) Restore(
	ctx context.Context,
	id string,
	userID string,
) (*models.Document, error) {

	query := `
    UPDATE documents
    SET status = 'draft'
    WHERE id = $1 AND owner_id = $2 AND status = 'archived'
    RETURNING id, title
    `

	var d models.Document

	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&d.ID,
		&d.Title,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &d, nil
}

// ======================================
// DELETE DOCUMENT ✅ FIXED
// ======================================
func (r *TrashRepository) Delete(
	ctx context.Context,
	id string,
	userID string,
) (*models.Document, error) {

	query := `
    DELETE FROM documents
    WHERE id = $1 AND owner_id = $2 AND status = 'archived'
    RETURNING id, file_key
    `

	var d models.Document

	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&d.ID,
		&d.FileKey,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &d, nil
}

// ======================================
// EMPTY TRASH ✅ FIXED
// ======================================
func (r *TrashRepository) Empty(
	ctx context.Context,
	userID string,
) ([]models.Document, error) {

	query := `
    DELETE FROM documents
    WHERE owner_id = $1 AND status = 'archived'
    RETURNING file_key
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("trash empty: %w", err)
	}
	defer rows.Close()

	docs := make([]models.Document, 0)

	for rows.Next() {
		var d models.Document

		if err := rows.Scan(&d.FileKey); err != nil {
			return nil, err
		}

		docs = append(docs, d)
	}

	return docs, nil
}
