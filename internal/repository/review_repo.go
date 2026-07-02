package repository

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Submit(ctx context.Context, documentID, submitterID string) (*models.DocumentReview, error) {
	query := `
		INSERT INTO document_reviews (document_id, submitter_id)
		VALUES ($1, $2)
		RETURNING id, document_id, submitter_id, reviewer_id, decision, note, created_at, reviewed_at`

	var rev models.DocumentReview
	err := r.db.QueryRow(ctx, query, documentID, submitterID).Scan(
		&rev.ID, &rev.DocumentID, &rev.SubmitterID,
		&rev.ReviewerID, &rev.Decision, &rev.Note,
		&rev.CreatedAt, &rev.ReviewedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("review repo submit: %w", err)
	}
	return &rev, nil
}

func (r *ReviewRepository) Decide(ctx context.Context, reviewID, reviewerID, decision string, note *string) error {
	query := `
		UPDATE document_reviews
		SET decision = $1, reviewer_id = $2, note = $3, reviewed_at = NOW()
		WHERE id = $4`
	tag, err := r.db.Exec(ctx, query, decision, reviewerID, note, reviewID)
	if err != nil {
		return fmt.Errorf("review repo decide: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("review not found")
	}
	return nil
}

func (r *ReviewRepository) GetByDocument(ctx context.Context, documentID string) ([]*models.DocumentReview, error) {
	query := `
		SELECT r.id, r.document_id, d.title,
		       r.submitter_id, su.name,
		       r.reviewer_id,  CASE WHEN rv.name IS NOT NULL THEN rv.name END,
		       r.decision, r.note, r.created_at, r.reviewed_at
		FROM document_reviews r
		JOIN documents d  ON d.id  = r.document_id
		JOIN users     su ON su.id = r.submitter_id
		LEFT JOIN users rv ON rv.id = r.reviewer_id
		WHERE r.document_id = $1
		ORDER BY r.created_at DESC`

	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("review repo by doc: %w", err)
	}
	defer rows.Close()

	var list []*models.DocumentReview
	for rows.Next() {
		var rev models.DocumentReview
		if err := rows.Scan(
			&rev.ID, &rev.DocumentID, &rev.DocumentTitle,
			&rev.SubmitterID, &rev.SubmitterName,
			&rev.ReviewerID, &rev.ReviewerName,
			&rev.Decision, &rev.Note, &rev.CreatedAt, &rev.ReviewedAt,
		); err != nil {
			return nil, fmt.Errorf("review repo scan: %w", err)
		}
		list = append(list, &rev)
	}
	return list, rows.Err()
}

// PendingReviews returns all documents currently in pending_review state (admin queue).
func (r *ReviewRepository) PendingQueue(ctx context.Context) ([]*models.DocumentReview, error) {
	query := `
		SELECT r.id, r.document_id, d.title,
		       r.submitter_id, su.name,
		       r.reviewer_id, NULL,
		       r.decision, r.note, r.created_at, r.reviewed_at
		FROM document_reviews r
		JOIN documents d  ON d.id  = r.document_id
		JOIN users     su ON su.id = r.submitter_id
		WHERE r.decision = 'pending'
		ORDER BY r.created_at ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("review repo pending: %w", err)
	}
	defer rows.Close()

	var list []*models.DocumentReview
	for rows.Next() {
		var rev models.DocumentReview
		var reviewerName *string
		if err := rows.Scan(
			&rev.ID, &rev.DocumentID, &rev.DocumentTitle,
			&rev.SubmitterID, &rev.SubmitterName,
			&rev.ReviewerID, &reviewerName,
			&rev.Decision, &rev.Note, &rev.CreatedAt, &rev.ReviewedAt,
		); err != nil {
			return nil, fmt.Errorf("review repo pending scan: %w", err)
		}
		rev.ReviewerName = reviewerName
		list = append(list, &rev)
	}
	return list, rows.Err()
}

func (r *ReviewRepository) GetPendingReviewID(ctx context.Context, documentID string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx,
		`SELECT id FROM document_reviews WHERE document_id = $1 AND decision = 'pending' ORDER BY created_at DESC LIMIT 1`,
		documentID,
	).Scan(&id)
	return id, err
}
