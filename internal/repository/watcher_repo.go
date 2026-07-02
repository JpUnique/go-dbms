package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WatcherRepository struct {
	db *pgxpool.Pool
}

func NewWatcherRepository(db *pgxpool.Pool) *WatcherRepository {
	return &WatcherRepository{db: db}
}

// Toggle adds or removes a watcher. Returns true if now watching.
func (r *WatcherRepository) Toggle(ctx context.Context, documentID, userID string) (bool, error) {
	// Try insert; if conflict (already watching) → delete
	tag, err := r.db.Exec(ctx,
		`INSERT INTO document_watchers (document_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		documentID, userID,
	)
	if err != nil {
		return false, fmt.Errorf("watcher toggle insert: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return true, nil // now watching
	}
	// Was already watching → remove
	_, err = r.db.Exec(ctx,
		`DELETE FROM document_watchers WHERE document_id = $1 AND user_id = $2`,
		documentID, userID,
	)
	return false, err
}

func (r *WatcherRepository) IsWatching(ctx context.Context, documentID, userID string) (bool, error) {
	var watching bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM document_watchers WHERE document_id = $1 AND user_id = $2)`,
		documentID, userID,
	).Scan(&watching)
	return watching, err
}

func (r *WatcherRepository) WatcherCount(ctx context.Context, documentID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM document_watchers WHERE document_id = $1`, documentID,
	).Scan(&n)
	return n, err
}

// WatcherUserIDs returns IDs of all users watching a document.
func (r *WatcherRepository) WatcherUserIDs(ctx context.Context, documentID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id FROM document_watchers WHERE document_id = $1`, documentID)
	if err != nil {
		return nil, fmt.Errorf("watcher ids: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
