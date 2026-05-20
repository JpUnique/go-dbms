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

func (r *TrashRepository) GetAll(ctx context.Context, userID string) ([]models.Document, error) {

	query := `
    SELECT * FROM documents
    WHERE status = 'archived' AND owner_id = $1
    ORDER BY updated_at DESC
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("trash repo get all: %w", err)
	}
	defer rows.Close()

	var docs []models.Document

	for rows.Next() {
		var d models.Document
		rows.Scan(
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
		)
		docs = append(docs, d)
	}

	return docs, nil
}

func (r *TrashRepository) Restore(ctx context.Context, id string) (*models.Document, error) {

	query := `
    UPDATE documents
    SET status = 'draft'
    WHERE id = $1 AND status = 'archived'
    RETURNING id, title
    `

	var d models.Document

	err := r.db.QueryRow(ctx, query, id).Scan(&d.ID, &d.Title)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &d, nil
}

func (r *TrashRepository) Delete(ctx context.Context, id string) (*models.Document, error) {

	query := `
    DELETE FROM documents
    WHERE id = $1 AND status = 'archived'
    RETURNING id, file_key
    `

	var d models.Document

	err := r.db.QueryRow(ctx, query, id).Scan(&d.ID, &d.FileKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &d, nil
}

func (r *TrashRepository) Empty(ctx context.Context) ([]models.Document, error) {
	query := `
    DELETE FROM documents
    WHERE status = 'archived'
    RETURNING file_key
    `

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("trash empty: %w", err)
	}
	defer rows.Close()

	var docs []models.Document

	for rows.Next() {
		var d models.Document
		rows.Scan(&d.FileKey)
		docs = append(docs, d)
	}

	return docs, nil
}
