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

func NewDocumentVersionRepository(db *pgxpool.Pool) *DocumentVersionRepository {
	return &DocumentVersionRepository{db: db}
}

// ======================================
// BEGIN TRANSACTION ✅ IMPROVED
// ======================================
func (r *DocumentVersionRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("version repo begin tx: %w", err)
	}

	return tx, nil
}

// ======================================
// LOCK DOCUMENT (FOR UPDATE) ✅ FIXED
// ======================================
func (r *DocumentVersionRepository) GetCurrentVersionForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	docID string,
	userID string,
	isAdmin bool,
	department *string,
) (int, error) {

	query := `
    SELECT version
    FROM documents
    WHERE id = $1 AND (
      owner_id = $2
      OR $3
      OR ($4::text IS NOT NULL AND department = $4)
    )
    FOR UPDATE
    `

	var version int

	err := tx.QueryRow(ctx, query, docID, userID, isAdmin, department).Scan(&version)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// ✅ better error than silent 0
			return 0, fmt.Errorf("document not found or unauthorized")
		}
		return 0, fmt.Errorf("lock document failed: %w", err)
	}

	return version, nil
}

// ======================================
// CREATE NEW VERSION ✅ IMPROVED
// ======================================
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
		return fmt.Errorf("create version failed: %w", err)
	}

	return nil
}

// ======================================
// GET VERSIONS ✅ CLEAN + SAFE
// ======================================
func (r *DocumentVersionRepository) GetByDocument(
	ctx context.Context,
	docID string,
	userID string,
	isAdmin bool,
	department *string,
) ([]models.DocumentVersion, error) {

	query := `
    SELECT
        v.id, v.document_id, v.version, v.file_key,
        v.file_size, v.uploaded_by, v.change_note, v.created_at
    FROM document_versions v
    JOIN documents d ON v.document_id = d.id
    WHERE v.document_id = $1 AND (
      d.owner_id = $2
      OR $3
      OR ($4::text IS NOT NULL AND d.department = $4)
    )
    ORDER BY v.version DESC
    `

	rows, err := r.db.Query(ctx, query, docID, userID, isAdmin, department)
	if err != nil {
		return nil, fmt.Errorf("get versions failed: %w", err)
	}
	defer rows.Close()

	var versions []models.DocumentVersion

	for rows.Next() {
		var v models.DocumentVersion

		if err := rows.Scan(
			&v.ID,
			&v.DocumentID,
			&v.Version,
			&v.FileKey,
			&v.FileSize,
			&v.UploadedBy,
			&v.ChangeNote,
			&v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan version failed: %w", err)
		}

		versions = append(versions, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return versions, nil
}

// ======================================
// GET SINGLE VERSION ✅ FIXED + EXTENDED
// ======================================
func (r *DocumentVersionRepository) GetByID(
	ctx context.Context,
	docID string,
	versionID string,
	userID string,
	isAdmin bool,
	department *string,
) (*models.DocumentVersion, string, string, error) {

	query := `
    SELECT
        v.id, v.file_key,
        d.file_name,
        u.name
    FROM document_versions v
    JOIN documents d ON v.document_id = d.id
    JOIN users u ON d.owner_id = u.id
    WHERE v.id = $1 AND v.document_id = $2 AND (
      d.owner_id = $3
      OR $4
      OR ($5::text IS NOT NULL AND d.department = $5)
    )
    `

	var version models.DocumentVersion
	var fileName string
	var ownerName string

	err := r.db.QueryRow(
		ctx,
		query,
		versionID,
		docID,
		userID,
		isAdmin,
		department,
	).Scan(
		&version.ID,
		&version.FileKey,
		&fileName,
		&ownerName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", "", nil
		}
		return nil, "", "", fmt.Errorf("get version failed: %w", err)
	}

	return &version, fileName, ownerName, nil
}
