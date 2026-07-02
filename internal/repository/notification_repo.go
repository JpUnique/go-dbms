package repository

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification) (*models.Notification, error) {
	query := `
		INSERT INTO notifications (user_id, type, title, body, resource_type, resource_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, type, title, body, resource_type, resource_id, is_read, created_at`

	var out models.Notification
	err := r.db.QueryRow(ctx, query,
		n.UserID, n.Type, n.Title, n.Body, n.ResourceType, n.ResourceID,
	).Scan(&out.ID, &out.UserID, &out.Type, &out.Title, &out.Body,
		&out.ResourceType, &out.ResourceID, &out.IsRead, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("notification repo create: %w", err)
	}
	return &out, nil
}

func (r *NotificationRepository) GetForUser(ctx context.Context, userID string, limit int) ([]*models.Notification, error) {
	if limit <= 0 {
		limit = 30
	}
	query := `
		SELECT id, user_id, type, title, body, resource_type, resource_id, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("notification repo list: %w", err)
	}
	defer rows.Close()

	var list []*models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body,
			&n.ResourceType, &n.ResourceID, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("notification repo scan: %w", err)
		}
		list = append(list, &n)
	}
	return list, rows.Err()
}

func (r *NotificationRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID,
	).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`, userID)
	return err
}

// GetUserEmail returns the email and email_notifications preference for a user.
// Used before sending email so we can respect the user's preference.
func (r *NotificationRepository) GetUserEmailPref(ctx context.Context, userID string) (email string, wantsEmail bool, err error) {
	query := `
		SELECT u.email, COALESCE(p.email_notifications, true)
		FROM users u
		LEFT JOIN user_preferences p ON p.user_id = u.id
		WHERE u.id = $1`
	err = r.db.QueryRow(ctx, query, userID).Scan(&email, &wantsEmail)
	return
}
