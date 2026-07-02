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

	if doc.Status == "" {
		doc.Status = "draft"
	}

	query := `
    INSERT INTO documents (
        title, file_name, file_key, file_type, file_size, owner_id,
        description, folder_id, department, status
    )
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    RETURNING id, status, created_at, updated_at
    `

	err := r.db.QueryRow(ctx, query,
		doc.Title,
		doc.FileName,
		doc.FileKey,
		doc.FileType,
		doc.FileSize,
		doc.OwnerID,
		doc.Description,
		doc.FolderID,
		doc.Department,
		doc.Status,
	).Scan(
		&doc.ID,
		&doc.Status,
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

	docs := make([]models.Document, 0)

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

// GetByID returns the document if userID owns it OR has been directly
// shared it (any permission level) — callers that require true ownership
// (e.g. granting/revoking shares, deleting) must additionally compare
// doc.OwnerID == userID themselves.
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
    WHERE id = $1 AND (
      owner_id = $2
      OR EXISTS (
        SELECT 1 FROM document_user_shares
        WHERE document_id = documents.id AND user_id = $2
      )
    )
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

// GetByIDForDownload is like GetByID, but a share only grants access here
// if its permission is specifically "download" — a "view"-only share must
// not be able to download the file.
func (r *DocumentRepository) GetByIDForDownload(
	ctx context.Context,
	docID string,
	userID string,
) (*models.Document, error) {

	query := `
    SELECT id, title, file_name, file_key, file_type, file_size,
           owner_id, status, version, is_starred,
           created_at, updated_at
    FROM documents
    WHERE id = $1 AND (
      owner_id = $2
      OR EXISTS (
        SELECT 1 FROM document_user_shares
        WHERE document_id = documents.id AND user_id = $2 AND permission = 'download'
      )
    )
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
		return nil, fmt.Errorf("document repo get by id for download: %w", err)
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
	folderID **string, // double-pointer: nil = don't change, &nil = move to root, &id = move to folder
) (*models.Document, error) {

	query := `
    UPDATE documents
    SET
        title      = COALESCE($1, title),
        status     = COALESCE($2, status),
        is_starred = COALESCE($3, is_starred),
        folder_id  = CASE WHEN $4 THEN $5 ELSE folder_id END,
        updated_at = NOW()
    WHERE id = $6 AND owner_id = $7
    RETURNING id, title, file_name, file_key, file_type, file_size,
              folder_id, owner_id, status, version, is_starred,
              created_at, updated_at
    `

	changeFolder := folderID != nil
	var newFolderID *string
	if changeFolder {
		newFolderID = *folderID
	}

	doc := &models.Document{}

	err := r.db.QueryRow(ctx, query,
		title,
		status,
		isStarred,
		changeFolder,
		newFolderID,
		docID,
		userID,
	).Scan(
		&doc.ID,
		&doc.Title,
		&doc.FileName,
		&doc.FileKey,
		&doc.FileType,
		&doc.FileSize,
		&doc.FolderID,
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

// UpdateStatus forcefully sets the document status regardless of owner (used by approval workflow).
func (r *DocumentRepository) UpdateStatus(ctx context.Context, docID, _ string, status string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE documents SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, docID,
	)
	if err != nil {
		return fmt.Errorf("document repo update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("document not found")
	}
	return nil
}

// Delete moves a document to Trash (status = 'archived') — it does not
// remove the row or the underlying file. Permanent removal only happens via
// TrashRepository.Delete/Empty once the user empties the trash.
func (r *DocumentRepository) Delete(
	ctx context.Context,
	docID string,
	userID string,
) (*models.Document, error) {

	query := `
    UPDATE documents
    SET status = 'archived'
    WHERE id = $1 AND owner_id = $2 AND status != 'archived'
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
) ([]models.DocumentWithMeta, int, error) {

	conditions := []string{"d.owner_id = $1", "d.status != 'archived'"}
	args := []interface{}{userID}
	argIndex := 2

	// SEARCH
	if query.Search != "" {
		conditions = append(conditions,
			fmt.Sprintf("(d.title ILIKE $%d OR d.file_name ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+query.Search+"%")
		argIndex++
	}

	// STATUS FILTER
	if query.Status != "" {
		conditions = append(conditions, fmt.Sprintf("d.status = $%d", argIndex))
		args = append(args, query.Status)
		argIndex++
	}

	// STARRED FILTER
	if query.Starred == "true" || query.Starred == "false" {
		conditions = append(conditions, fmt.Sprintf("d.is_starred = $%d", argIndex))
		args = append(args, query.Starred == "true")
		argIndex++
	}

	// FOLDER FILTER
	if query.FolderID != "" {
		conditions = append(conditions, fmt.Sprintf("d.folder_id = $%d", argIndex))
		args = append(args, query.FolderID)
		argIndex++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// COUNT QUERY — no joins needed, use plain table name
	countQuery := fmt.Sprintf(`
    SELECT count(*) FROM documents d %s
    `, where)

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("document repo count: %w", err)
	}

	// PAGINATION
	offset := (query.Page - 1) * query.Limit
	args = append(args, query.Limit, offset)

	queryStr := fmt.Sprintf(`
    SELECT d.id, d.title, d.description, d.file_name, d.file_key, d.file_type, d.file_size,
           d.folder_id, d.owner_id, d.department, d.status, d.version, d.is_starred,
           d.last_accessed, d.created_at, d.updated_at,
           u.name  AS owner_name,
           f.name  AS folder_name
    FROM documents d
    LEFT JOIN users   u ON u.id = d.owner_id
    LEFT JOIN folders f ON f.id = d.folder_id
    %s
    ORDER BY d.updated_at DESC
    LIMIT $%d OFFSET $%d
    `, where, argIndex, argIndex+1)

	rows, err := r.db.Query(ctx, queryStr, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("document repo query: %w", err)
	}
	defer rows.Close()

	docs := make([]models.DocumentWithMeta, 0)

	for rows.Next() {
		var d models.DocumentWithMeta
		err := rows.Scan(
			&d.ID, &d.Title, &d.Description, &d.FileName, &d.FileKey, &d.FileType, &d.FileSize,
			&d.FolderID, &d.OwnerID, &d.Department, &d.Status, &d.Version, &d.IsStarred,
			&d.LastAccess, &d.CreatedAt, &d.UpdatedAt,
			&d.OwnerName, &d.FolderName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("document repo scan: %w", err)
		}
		docs = append(docs, d)
	}

	return docs, total, nil
}
