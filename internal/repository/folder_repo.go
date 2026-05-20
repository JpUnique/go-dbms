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

type FolderRepository struct {
	db *pgxpool.Pool
}

// constructor
func NewFolderRepository(db *pgxpool.Pool) *FolderRepository {
	return &FolderRepository{db: db}
}

func (r *FolderRepository) GetAllFolders(
	ctx context.Context,
	userID string,
	parentID string,
	limit int,
	offset int,
) ([]models.Folder, error) {

	conditions := []string{"f.owner_id = $1"}
	args := []interface{}{userID}
	argIndex := 2

	// ✅ parent filter
	if parentID == "null" || parentID == "" {
		conditions = append(conditions, "f.parent_id IS NULL")
	} else if parentID != "" {
		conditions = append(conditions, fmt.Sprintf("f.parent_id = $%d", argIndex))
		args = append(args, parentID)
		argIndex++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
    SELECT f.id, f.name, f.owner_id, f.parent_id, f.department,
           f.created_at, f.updated_at,
           (SELECT COUNT(*) FROM documents d WHERE d.folder_id = f.id) AS document_count,
           (SELECT COUNT(*) FROM folders c WHERE c.parent_id = f.id) AS subfolder_count
    FROM folders f
    %s
    ORDER BY f.name ASC
    LIMIT $%d OFFSET $%d
    `, where, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("folder repo get all: %w", err)
	}
	defer rows.Close()

	var folders []models.Folder

	for rows.Next() {
		var f models.Folder

		err := rows.Scan(
			&f.ID,
			&f.Name,
			&f.OwnerID,
			&f.ParentID,
			&f.Department,
			&f.CreatedAt,
			&f.UpdatedAt,
			&f.DocumentCount,
			&f.SubfolderCount,
		)
		if err != nil {
			return nil, fmt.Errorf("folder repo scan: %w", err)
		}

		folders = append(folders, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("folder repo rows error: %w", err)
	}

	return folders, nil
}

func (r *FolderRepository) GetByID(
	ctx context.Context,
	folderID string,
	userID string,
) (*models.Folder, error) {

	query := `
    SELECT id, name, owner_id, parent_id, department, created_at, updated_at
    FROM folders
    WHERE id = $1 AND owner_id = $2
    `

	folder := &models.Folder{}

	err := r.db.QueryRow(ctx, query, folderID, userID).Scan(
		&folder.ID,
		&folder.Name,
		&folder.OwnerID,
		&folder.ParentID,
		&folder.Department,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("folder repo get by id: %w", err)
	}

	return folder, nil
}

func (r *FolderRepository) CreateFolder(
	ctx context.Context,
	folder *models.Folder,
) error {

	query := `
    INSERT INTO folders (name, parent_id, owner_id, department)
    VALUES ($1, $2, $3, $4)
    RETURNING id, created_at, updated_at
    `

	err := r.db.QueryRow(ctx, query,
		folder.Name,
		folder.ParentID,
		folder.OwnerID,
		folder.Department,
	).Scan(
		&folder.ID,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("folder repo create: %w", err)
	}

	return nil
}

func (r *FolderRepository) Update(
	ctx context.Context,
	folderID string,
	userID string,
	name *string,
	parentID *string,
	department *string,
) (*models.Folder, error) {

	query := `
    UPDATE folders
    SET
        name = COALESCE($1, name),
        parent_id = COALESCE($2, parent_id),
        department = COALESCE($3, department),
        updated_at = NOW()
    WHERE id = $4 AND owner_id = $5
    RETURNING id, name, owner_id, parent_id, department, created_at, updated_at
    `

	folder := &models.Folder{}

	err := r.db.QueryRow(ctx, query,
		name,
		parentID,
		department,
		folderID,
		userID,
	).Scan(
		&folder.ID,
		&folder.Name,
		&folder.OwnerID,
		&folder.ParentID,
		&folder.Department,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("folder repo update: %w", err)
	}

	return folder, nil
}

func (r *FolderRepository) Delete(
	ctx context.Context,
	folderID string,
	userID string,
) (bool, error) {

	query := `
    DELETE FROM folders
    WHERE id = $1 AND owner_id = $2
    `

	cmdTag, err := r.db.Exec(ctx, query, folderID, userID)
	if err != nil {
		return false, fmt.Errorf("folder repo delete: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return false, nil
	}

	return true, nil
}
