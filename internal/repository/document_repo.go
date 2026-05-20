package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentRepository struct {
	db *pgxpool.Pool
}

func NewDocumentRepository(db *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// ==============================
// CREATE DOCUMENT
// ==============================
func (r *DocumentRepository) Create(
	ctx context.Context,
	doc *models.Document,
) error {

	query := `
    INSERT INTO documents (
        title, file_name, file_key, file_type, file_size, owner_id
    )
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id, created_at, updated_at
    `

	err := r.db.QueryRow(ctx, query,
		doc.Title,
		doc.FileName,
		doc.FileKey,
		doc.FileType,
		doc.FileSize,
		doc.OwnerID,
	).Scan(
		&doc.ID,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("document repo create: %w", err)
	}

	return nil
}

func (r *DocumentRepository) GetByUser(
	ctx context.Context,
	userID string,
) ([]models.Document, error) {

	query := `
    SELECT id, title, file_name, file_key, file_type, file_size,
           owner_id, status, version, is_starred,
           created_at, updated_at
    FROM documents
    WHERE owner_id = $1
    ORDER BY updated_at DESC
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("document repo get by user: %w", err)
	}
	defer rows.Close()

	var docs []models.Document

	for rows.Next() {
		var doc models.Document

		err := rows.Scan(
			&doc.ID,
			&doc.Title,
			&doc.FileName,
			&doc.FileKey,
			&doc.FileType,
			&doc.FileSize,
			&doc.OwnerID,
			&doc.Status,
			&doc.Version,
			&doc.IsStarred,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("document repo scan: %w", err)
		}

		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document repo rows error: %w", err)
	}

	return docs, nil
}

func (r *DocumentRepository) GetByID(
	ctx context.Context,
	docID string,
	userID string,
) (*models.Document, error) {

	query := `
    SELECT id, title, file_name, file_key, file_type, file_size,
           owner_id, status, version, is_starred,
           created_at, updated_at
    FROM documents
    WHERE id = $1 AND owner_id = $2
    `

	doc := &models.Document{}

	err := r.db.QueryRow(ctx, query, docID, userID).Scan(
		&doc.ID,
		&doc.Title,
		&doc.FileName,
		&doc.FileKey,
		&doc.FileType,
		&doc.FileSize,
		&doc.OwnerID,
		&doc.Status,
		&doc.Version,
		&doc.IsStarred,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("document repo get by id: %w", err)
	}

	return doc, nil
}

func (r *DocumentRepository) Update(
	ctx context.Context,
	docID string,
	userID string,
	title *string,
	status *string,
	isStarred *bool,
) (*models.Document, error) {

	query := `
    UPDATE documents
    SET
        title = COALESCE($1, title),
        status = COALESCE($2, status),
        is_starred = COALESCE($3, is_starred),
        updated_at = NOW()
    WHERE id = $4 AND owner_id = $5
    RETURNING id, title, file_name, file_key, file_type, file_size,
              owner_id, status, version, is_starred,
              created_at, updated_at
    `

	doc := &models.Document{}

	err := r.db.QueryRow(ctx, query,
		title,
		status,
		isStarred,
		docID,
		userID,
	).Scan(
		&doc.ID,
		&doc.Title,
		&doc.FileName,
		&doc.FileKey,
		&doc.FileType,
		&doc.FileSize,
		&doc.OwnerID,
		&doc.Status,
		&doc.Version,
		&doc.IsStarred,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("document repo update: %w", err)
	}

	return doc, nil
}

func (r *DocumentRepository) Delete(
	ctx context.Context,
	docID string,
	userID string,
) (*models.Document, error) {

	query := `
    DELETE FROM documents
    WHERE id = $1 AND owner_id = $2
    RETURNING id, file_key
    `

	doc := &models.Document{}

	err := r.db.QueryRow(ctx, query, docID, userID).
		Scan(&doc.ID, &doc.FileKey)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("document repo delete: %w", err)
	}

	return doc, nil
}

func (r *DocumentRepository) ToggleStar(
	ctx context.Context,
	docID string,
	userID string,
) (bool, error) {

	query := `
    UPDATE documents
    SET is_starred = NOT is_starred
    WHERE id = $1 AND owner_id = $2
    RETURNING is_starred
    `

	var isStarred bool

	err := r.db.QueryRow(ctx, query, docID, userID).
		Scan(&isStarred)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("document repo toggle star: not found")
		}
		return false, fmt.Errorf("document repo toggle star: %w", err)
	}

	return isStarred, nil
}

func (r *DocumentRepository) UpdateLatestVersion(
	ctx context.Context,
	tx pgx.Tx,
	docID string,
	userID string,
	version int,
	fileKey string,
	fileName string,
	fileType string,
	fileSize int64,
) error {

	query := `
    UPDATE documents
    SET version = $1,
        file_key = $2,
        file_name = $3,
        file_type = $4,
        file_size = $5,
        updated_at = NOW()
    WHERE id = $6 AND owner_id = $7
    `

	_, err := tx.Exec(ctx, query,
		version,
		fileKey,
		fileName,
		fileType,
		fileSize,
		docID,
		userID,
	)

	if err != nil {
		return fmt.Errorf("document repo update latest version: %w", err)
	}

	return nil
}

func (r *DocumentRepository) GetByUserWithFilter(
	ctx context.Context,
	userID string,
	query models.DocumentQuery,
) ([]models.Document, int, error) {

	conditions := []string{"owner_id = $1"}
	args := []interface{}{userID}
	argIndex := 2

	// ✅ SEARCH
	if query.Search != "" {
		conditions = append(conditions,
			fmt.Sprintf("(title ILIKE $%d OR file_name ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+query.Search+"%")
		argIndex++
	}

	// ✅ STATUS FILTER
	if query.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, query.Status)
		argIndex++
	}

	// ✅ STARRED FILTER
	if query.Starred != nil {
		conditions = append(conditions, fmt.Sprintf("is_starred = $%d", argIndex))
		args = append(args, *query.Starred)
		argIndex++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// ✅ COUNT QUERY
	countQuery := fmt.Sprintf(`
    SELECT count(*) FROM documents %s
    `, where)

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("document repo count: %w", err)
	}

	// ✅ PAGINATION
	offset := (query.Page - 1) * query.Limit

	args = append(args, query.Limit, offset)

	queryStr := fmt.Sprintf(`
    SELECT id, title, file_name, file_key, file_type, file_size,
           owner_id, status, version, is_starred,
           created_at, updated_at
    FROM documents
    %s
    ORDER BY updated_at DESC
    LIMIT $%d OFFSET $%d
    `, where, argIndex, argIndex+1)

	rows, err := r.db.Query(ctx, queryStr, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("document repo query: %w", err)
	}
	defer rows.Close()

	var docs []models.Document

	for rows.Next() {
		var d models.Document

		err := rows.Scan(
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
		if err != nil {
			return nil, 0, fmt.Errorf("document repo scan: %w", err)
		}

		docs = append(docs, d)
	}

	return docs, total, nil
}
