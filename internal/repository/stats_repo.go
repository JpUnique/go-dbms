package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepository struct {
	db *pgxpool.Pool
}

func NewStatsRepository(db *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{db: db}
}

func (r *StatsRepository) GetDashboard(
	ctx context.Context,
	userID string,
	isAdmin bool,
) (map[string]interface{}, error) {

	where := ""
	args := []interface{}{}

	if !isAdmin {
		where = "WHERE owner_id = $1"
		args = append(args, userID)
	}

	data := make(map[string]interface{})

	// Documents stats
	documentsQuery := fmt.Sprintf(`
    SELECT COUNT(*) AS total,
           COUNT(*) FILTER (WHERE is_starred = true) AS starred,
           COUNT(*) FILTER (WHERE status = 'published') AS published,
           COUNT(*) FILTER (WHERE status = 'draft') AS draft,
           COUNT(*) FILTER (WHERE status = 'archived') AS archived
    FROM documents %s
    `, where)

	row := r.db.QueryRow(ctx, documentsQuery, args...)

	var total, starred, published, draft, archived int
	row.Scan(&total, &starred, &published, &draft, &archived)

	data["documents"] = map[string]int{
		"total":     total,
		"starred":   starred,
		"published": published,
		"draft":     draft,
		"archived":  archived,
	}

	// Storage
	storageQuery := fmt.Sprintf(`
    SELECT COALESCE(SUM(file_size),0) FROM documents %s
    `, where)

	var totalBytes int64
	r.db.QueryRow(ctx, storageQuery, args...).Scan(&totalBytes)

	data["storage"] = map[string]interface{}{
		"total_bytes": totalBytes,
		"total_mb":    float64(totalBytes) / (1024 * 1024),
	}

	// Recent
	recentQuery := fmt.Sprintf(`
    SELECT id, title, file_type, updated_at
    FROM documents %s
    ORDER BY updated_at DESC LIMIT 10
    `, where)

	rows, _ := r.db.Query(ctx, recentQuery, args...)
	defer rows.Close()

	var recent []map[string]interface{}

	for rows.Next() {
		var id, title, fileType string
		var updated string
		rows.Scan(&id, &title, &fileType, &updated)

		recent = append(recent, map[string]interface{}{
			"id":      id,
			"title":   title,
			"type":    fileType,
			"updated": updated,
		})
	}

	data["recent_documents"] = recent

	return data, nil
}

func (r *StatsRepository) GetActivity(
	ctx context.Context,
	userID string,
	isAdmin bool,
) ([]map[string]interface{}, error) {

	query := `
    SELECT DATE(created_at) AS date,
           COUNT(*) AS count
    FROM audit_logs
    `

	args := []interface{}{}

	if !isAdmin {
		query += " WHERE user_id = $1"
		args = append(args, userID)
	}

	query += " GROUP BY date ORDER BY date ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("stats repo activity: %w", err)
	}
	defer rows.Close()

	var result []map[string]interface{}

	for rows.Next() {
		var date string
		var count int

		rows.Scan(&date, &count)

		result = append(result, map[string]interface{}{
			"date":  date,
			"count": count,
		})
	}

	return result, nil
}
