package repository

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	db *pgxpool.Pool
}

func NewCommentRepository(db *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, documentID, userID, content string) (*models.Comment, error) {
	query := `
		INSERT INTO document_comments (document_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, document_id, user_id, content, created_at, updated_at`

	var c models.Comment
	err := r.db.QueryRow(ctx, query, documentID, userID, content).Scan(
		&c.ID, &c.DocumentID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("comment repo create: %w", err)
	}
	return &c, nil
}

func (r *CommentRepository) GetAll(ctx context.Context, documentID string) ([]*models.Comment, error) {
	query := `
		SELECT c.id, c.document_id, c.user_id, u.name AS user_name,
		       c.content, c.created_at, c.updated_at
		FROM document_comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.document_id = $1
		ORDER BY c.created_at ASC`

	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("comment repo list: %w", err)
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.UserID, &c.UserName,
			&c.Content, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("comment repo scan: %w", err)
		}
		comments = append(comments, &c)
	}
	return comments, rows.Err()
}

// Delete removes a comment; only the owner or an admin may delete.
func (r *CommentRepository) Delete(ctx context.Context, commentID, userID, role string) error {
	var query string
	var args []any

	if role == "admin" {
		query = `DELETE FROM document_comments WHERE id = $1`
		args = []any{commentID}
	} else {
		query = `DELETE FROM document_comments WHERE id = $1 AND user_id = $2`
		args = []any{commentID, userID}
	}

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("comment repo delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("comment not found or not authorised")
	}
	return nil
}
