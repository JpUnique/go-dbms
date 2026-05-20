package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentVersionRepository struct {
	db *pgxpool.Pool
}

// constructor
func NewDocumentVersionRepository(db *pgxpool.Pool) *DocumentVersionRepository {
	return &DocumentVersionRepository{
		db: db,
	}
}

// ==============================
// BEGIN TRANSACTION
// ==============================
func (r *DocumentVersionRepository) BeginTx(
	ctx context.Context,
) (pgx.Tx, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("version repo begin tx: %w", err)
	}

	return tx, nil
}

// ==============================
// LOCK DOCUMENT (FOR UPDATE)
// ==============================
func (r *DocumentVersionRepository) GetCurrentVersionForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	docID string,
	userID string,
) (int, error) {

	query := `
    SELECT version
    FROM documents
    WHERE id = $1 AND owner_id = $2
    FOR UPDATE
    `

	var version int

	err := tx.QueryRow(ctx, query, docID, userID).Scan(&version)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("version repo lock document: %w", err)
	}

	return version, nil
}

// ==============================
// CREATE NEW VERSION
// ==============================
func (r *DocumentVersionRepository) Create(
	ctx context.Context,
	tx pgx.Tx,
	version *models.DocumentVersion,
) error {

	query := `
    INSERT INTO document_versions
    (document_id, version, file_key, file_size, uploaded_by, change_note)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id, created_at
    `

	err := tx.QueryRow(ctx, query,
		version.DocumentID,
		version.Version,
		version.FileKey,
		version.FileSize,
		version.UploadedBy,
		version.ChangeNote,
	).Scan(
		&version.ID,
		&version.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("version repo create: %w", err)
	}

	return nil
}

// ==============================
// GET VERSIONS BY DOCUMENT
// ==============================
func (r *DocumentVersionRepository) GetByDocument(
	ctx context.Context,
	docID string,
	userID string,
) ([]models.DocumentVersion, error) {

	query := `
    SELECT v.id, v.document_id, v.version, v.file_key,
           v.file_size, v.uploaded_by, v.change_note, v.created_at
    FROM document_versions v
    JOIN documents d ON v.document_id = d.id
    WHERE v.document_id = $1 AND d.owner_id = $2
    ORDER BY v.version DESC
    `

	rows, err := r.db.Query(ctx, query, docID, userID)
	if err != nil {
		return nil, fmt.Errorf("version repo get by document: %w", err)
	}
	defer rows.Close()

	var versions []models.DocumentVersion

	for rows.Next() {
		var v models.DocumentVersion

		err := rows.Scan(
			&v.ID,
			&v.DocumentID,
			&v.Version,
			&v.FileKey,
			&v.FileSize,
			&v.UploadedBy,
			&v.ChangeNote,
			&v.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("version repo scan: %w", err)
		}

		versions = append(versions, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("version repo rows error: %w", err)
	}

	return versions, nil
}

// ==============================
// GET SINGLE VERSION (FOR DOWNLOAD)
// ==============================
func (r *DocumentVersionRepository) GetByID(
	ctx context.Context,
	docID string,
	versionID string,
	userID string,
) (*models.DocumentVersion, string, error) {

	query := `
    SELECT v.id, v.file_key, d.file_name
    FROM document_versions v
    JOIN documents d ON v.document_id = d.id
    WHERE v.id = $1 AND v.document_id = $2 AND d.owner_id = $3
    `

	var version models.DocumentVersion
	var fileName string

	err := r.db.QueryRow(
		ctx,
		query,
		versionID,
		docID,
		userID,
	).Scan(
		&version.ID,
		&version.FileKey,
		&fileName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("version repo get by id: %w", err)
	}

	return &version, fileName, nil
}
