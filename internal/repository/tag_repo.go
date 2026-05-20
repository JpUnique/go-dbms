package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagRepository struct {
	db *pgxpool.Pool
}

// constructor
func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) GetAll(
	ctx context.Context,
) ([]models.Tag, error) {

	query := `
    SELECT id, name, color, created_at
    FROM tags
    ORDER BY name ASC
    `

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("tag repo get all: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag

	for rows.Next() {
		var t models.Tag

		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Color,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("tag repo scan: %w", err)
		}

		tags = append(tags, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tag repo rows error: %w", err)
	}

	return tags, nil
}

func (r *TagRepository) Create(
	ctx context.Context,
	name string,
	color string,
) (*models.Tag, error) {

	query := `
    INSERT INTO tags (name, color)
    VALUES ($1, $2)
    RETURNING id, name, color, created_at
    `

	var t models.Tag

	err := r.db.QueryRow(ctx, query, name, color).Scan(
		&t.ID,
		&t.Name,
		&t.Color,
		&t.CreatedAt,
	)

	if err != nil {

		// ✅ Unique constraint (Postgres 23505)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return nil, fmt.Errorf("tag name already exists")
		}

		return nil, fmt.Errorf("tag repo create: %w", err)
	}

	return &t, nil
}

func (r *TagRepository) Update(
	ctx context.Context,
	id string,
	name *string,
	color *string,
) (*models.Tag, error) {

	query := `
    UPDATE tags
    SET
        name = COALESCE($1, name),
        color = COALESCE($2, color)
    WHERE id = $3
    RETURNING id, name, color, created_at
    `

	var t models.Tag

	err := r.db.QueryRow(ctx, query,
		name,
		color,
		id,
	).Scan(
		&t.ID,
		&t.Name,
		&t.Color,
		&t.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("tag repo update: %w", err)
	}

	return &t, nil
}

func (r *TagRepository) Delete(
	ctx context.Context,
	id string,
) error {

	query := `
    DELETE FROM tags WHERE id = $1
    `

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("tag repo delete: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("tag not found")
	}

	return nil
}

func (r *TagRepository) Attach(
	ctx context.Context,
	docID string,
	tagID string,
) error {

	query := `
    INSERT INTO document_tags (document_id, tag_id)
    VALUES ($1, $2)
    ON CONFLICT DO NOTHING
    `

	_, err := r.db.Exec(ctx, query, docID, tagID)
	if err != nil {
		return fmt.Errorf("tag repo attach: %w", err)
	}

	return nil
}

func (r *TagRepository) Detach(
	ctx context.Context,
	docID string,
	tagID string,
) error {

	query := `
	DELETE FROM document_tags
	WHERE document_id = $1 AND tag_id = $2
	`
	_, err := r.db.Exec(ctx, query, docID, tagID)
	if err != nil {
		return fmt.Errorf("tag repo detach: %w", err)
	}

	return nil
}
func (r *TagRepository) GetByDocument(
	ctx context.Context,
	docID string,
) ([]models.Tag, error) {

	query := `
    SELECT t.id, t.name, t.color, t.created_at
    FROM tags t
    JOIN document_tags dt ON dt.tag_id = t.id
    WHERE dt.document_id = $1
    ORDER BY t.name ASC
    `

	rows, err := r.db.Query(ctx, query, docID)
	if err != nil {
		return nil, fmt.Errorf("tag repo get by document: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag

	for rows.Next() {
		var t models.Tag

		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Color,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("tag repo scan: %w", err)
		}

		tags = append(tags, t)
	}

	return tags, nil
}
