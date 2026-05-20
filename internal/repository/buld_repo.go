package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BulkRepository struct {
	db *pgxpool.Pool
}

func NewBulkRepository(db *pgxpool.Pool) *BulkRepository {
	return &BulkRepository{db: db}
}

func (r *BulkRepository) Delete(
	ctx context.Context,
	userID string,
	ids []string,
) (int, error) {

	query := `
    DELETE FROM documents
    WHERE id = ANY($1) AND owner_id = $2
    `

	res, err := r.db.Exec(ctx, query, ids, userID)
	if err != nil {
		return 0, fmt.Errorf("bulk delete: %w", err)
	}

	return int(res.RowsAffected()), nil
}

func (r *BulkRepository) Archive(
	ctx context.Context,
	userID string,
	ids []string,
) (int, error) {

	query := `
    UPDATE documents
    SET status = 'archived'
    WHERE id = ANY($1) AND owner_id = $2
    `

	res, err := r.db.Exec(ctx, query, ids, userID)
	if err != nil {
		return 0, fmt.Errorf("bulk archive: %w", err)
	}

	return int(res.RowsAffected()), nil
}

func (r *BulkRepository) Move(
	ctx context.Context,
	userID string,
	ids []string,
	folderID *string,
) (int, error) {

	query := `
    UPDATE documents
    SET folder_id = $1
    WHERE id = ANY($2) AND owner_id = $3
    `

	res, err := r.db.Exec(ctx, query, folderID, ids, userID)
	if err != nil {
		return 0, fmt.Errorf("bulk move: %w", err)
	}

	return int(res.RowsAffected()), nil
}

func (r *BulkRepository) Update(
	ctx context.Context,
	userID string,
	ids []string,
	status *string,
	department *string,
) (int, error) {

	query := `
    UPDATE documents
    SET
        status = COALESCE($1, status),
        department = COALESCE($2, department)
    WHERE id = ANY($3) AND owner_id = $4
    `

	res, err := r.db.Exec(ctx, query, status, department, ids, userID)
	if err != nil {
		return 0, fmt.Errorf("bulk update: %w", err)
	}

	return int(res.RowsAffected()), nil
}
