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

	docWhere := ""
	args := []interface{}{}

	if !isAdmin {
		docWhere = "WHERE owner_id = $1"
		args = append(args, userID)
	}

	data := make(map[string]interface{})

	// ── Document counts ──────────────────────────────────────
	row := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE is_starred),
		       COUNT(*) FILTER (WHERE status = 'published'),
		       COUNT(*) FILTER (WHERE status = 'draft'),
		       COUNT(*) FILTER (WHERE status = 'archived')
		FROM documents %s`, docWhere), args...)

	var total, starred, published, draft, archived int
	row.Scan(&total, &starred, &published, &draft, &archived)
	data["documents"] = map[string]int{
		"total": total, "starred": starred,
		"published": published, "draft": draft, "archived": archived,
	}

	// ── Storage ──────────────────────────────────────────────
	var totalBytes int64
	r.db.QueryRow(ctx, fmt.Sprintf(
		`SELECT COALESCE(SUM(file_size),0) FROM documents %s`, docWhere), args...).Scan(&totalBytes)
	data["storage"] = map[string]interface{}{
		"total_bytes": totalBytes,
		"total_mb":    float64(totalBytes) / (1024 * 1024),
	}

	// ── Users (admin: all; non-admin: just the caller) ───────
	var userTotal, userActive int
	if isAdmin {
		r.db.QueryRow(ctx, `
			SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'active') FROM users`).
			Scan(&userTotal, &userActive)
	} else {
		r.db.QueryRow(ctx, `
			SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'active')
			FROM users WHERE id = $1`, userID).
			Scan(&userTotal, &userActive)
	}
	data["users"] = map[string]int{"total": userTotal, "active": userActive}

	// ── Folders ──────────────────────────────────────────────
	folderWhere := ""
	folderArgs := []interface{}{}
	if !isAdmin {
		folderWhere = "WHERE owner_id = $1"
		folderArgs = append(folderArgs, userID)
	}
	var folderTotal int
	r.db.QueryRow(ctx, fmt.Sprintf(
		`SELECT COUNT(*) FROM folders %s`, folderWhere), folderArgs...).Scan(&folderTotal)
	data["folders"] = map[string]int{"total": folderTotal}

	// ── Recent documents ─────────────────────────────────────
	recentRows, _ := r.db.Query(ctx, fmt.Sprintf(`
		SELECT d.id, d.title, d.file_type, d.updated_at::text,
		       COALESCE(u.name, '') AS owner_name
		FROM documents d
		LEFT JOIN users u ON u.id = d.owner_id
		%s
		ORDER BY d.updated_at DESC LIMIT 10`, docWhere), args...)
	defer recentRows.Close()

	recent := make([]map[string]interface{}, 0)
	for recentRows.Next() {
		var id, title, fileType, updated, ownerName string
		recentRows.Scan(&id, &title, &fileType, &updated, &ownerName)
		recent = append(recent, map[string]interface{}{
			"id": id, "title": title,
			"file_type": fileType, "updated_at": updated,
			"owner_name": ownerName,
		})
	}
	data["recent_documents"] = recent

	// ── By status ────────────────────────────────────────────
	statusRows, _ := r.db.Query(ctx, fmt.Sprintf(`
		SELECT status, COUNT(*) FROM documents %s
		GROUP BY status ORDER BY status`, docWhere), args...)
	defer statusRows.Close()

	byStatus := make([]map[string]interface{}, 0)
	for statusRows.Next() {
		var status string
		var cnt int
		statusRows.Scan(&status, &cnt)
		byStatus = append(byStatus, map[string]interface{}{"status": status, "count": cnt})
	}
	data["by_status"] = byStatus

	// ── By file type ─────────────────────────────────────────
	typeRows, _ := r.db.Query(ctx, fmt.Sprintf(`
		SELECT UPPER(SPLIT_PART(file_name, '.', -1)) AS category, COUNT(*)
		FROM documents %s
		GROUP BY category ORDER BY COUNT(*) DESC LIMIT 8`, docWhere), args...)
	defer typeRows.Close()

	byType := make([]map[string]interface{}, 0)
	for typeRows.Next() {
		var category string
		var cnt int
		typeRows.Scan(&category, &cnt)
		byType = append(byType, map[string]interface{}{"category": category, "count": cnt})
	}
	data["by_type"] = byType

	// ── By department (admin only) ────────────────────────────
	deptRows, _ := r.db.Query(ctx, `
		SELECT COALESCE(department,'Unknown') AS dept, COUNT(*)
		FROM documents GROUP BY dept ORDER BY COUNT(*) DESC LIMIT 6`)
	defer deptRows.Close()

	byDept := make([]map[string]interface{}, 0)
	if isAdmin {
		for deptRows.Next() {
			var dept string
			var cnt int
			deptRows.Scan(&dept, &cnt)
			byDept = append(byDept, map[string]interface{}{"department": dept, "count": cnt})
		}
	}
	data["by_department"] = byDept

	return data, nil
}

func (r *StatsRepository) GetActivity(
	ctx context.Context,
	userID string,
	isAdmin bool,
	period string,
) ([]map[string]interface{}, error) {

	type periodCfg struct {
		interval string
		bucket   string
	}

	cfg := map[string]periodCfg{
		"today":    {interval: "DATE_TRUNC('day', NOW())",    bucket: "TO_CHAR(t.bucket, 'HH24:00')"},
		"day":      {interval: "NOW() - INTERVAL '48 hours'", bucket: "TO_CHAR(t.bucket, 'HH24:00')"},
		"week":     {interval: "NOW() - INTERVAL '7 days'",   bucket: "DATE(t.bucket)::text"},
		"month":    {interval: "NOW() - INTERVAL '30 days'",  bucket: "DATE(t.bucket)::text"},
		"halfyear": {interval: "NOW() - INTERVAL '6 months'", bucket: "TO_CHAR(t.bucket, 'YYYY-MM')"},
		"year":     {interval: "NOW() - INTERVAL '1 year'",   bucket: "TO_CHAR(t.bucket, 'YYYY-MM')"},
	}

	pc, ok := cfg[period]
	if !ok {
		pc = cfg["week"]
	}

	// Combine document uploads (from documents table, always available) with
	// audit_log events (view/download/star/delete etc., logged after the fix).
	// The EXCEPT prevents double-counting uploads that are already in audit_logs.
	var userFilter, auditFilter string
	args := []interface{}{}

	if isAdmin {
		userFilter = ""
		auditFilter = ""
	} else {
		userFilter = "AND d.owner_id = $1"
		auditFilter = "AND al.user_id = $1"
		args = append(args, userID)
	}

	// trunc determines the PostgreSQL DATE_TRUNC unit so generate_series
	// buckets align with the bucket label expression.
	trunc := map[string]string{
		"today": "hour", "day": "hour",
		"week": "day", "month": "day",
		"halfyear": "month", "year": "month",
	}[period]
	if trunc == "" {
		trunc = "day"
	}

	// Use generate_series to produce EVERY time bucket in the range (with zeros
	// where there is no activity) – this gives the spike / trading-chart look.
	query := fmt.Sprintf(`
		WITH
		time_series AS (
			SELECT generate_series(
				DATE_TRUNC('%s', %s),
				DATE_TRUNC('%s', NOW()),
				'1 %s'::interval
			) AS bucket
		),
		events AS (
			SELECT d.created_at AS ts, 'document'::text AS resource_type
			FROM documents d
			WHERE d.created_at >= %s AND d.deleted_at IS NULL %s
			UNION ALL
			SELECT al.created_at AS ts, al.resource_type
			FROM audit_logs al
			WHERE al.created_at >= %s AND al.action != 'upload' %s
		)
		SELECT %s AS date,
		       COUNT(e.ts)                                                  AS count,
		       COUNT(e.ts) FILTER (WHERE e.resource_type = 'document')     AS document_actions,
		       COUNT(e.ts) FILTER (WHERE e.resource_type = 'user')         AS user_actions
		FROM time_series t
		LEFT JOIN events e ON DATE_TRUNC('%s', e.ts) = t.bucket
		GROUP BY t.bucket
		ORDER BY t.bucket ASC
	`,
		trunc, pc.interval,
		trunc, trunc,
		pc.interval, userFilter,
		pc.interval, auditFilter,
		pc.bucket,
		trunc,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("stats repo activity: %w", err)
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var date string
		var count, docActions, userActions int
		rows.Scan(&date, &count, &docActions, &userActions)
		result = append(result, map[string]interface{}{
			"date":             date,
			"count":            count,
			"document_actions": docActions,
			"user_actions":     userActions,
		})
	}

	return result, nil
}
