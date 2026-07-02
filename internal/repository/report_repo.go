package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepository struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

type ReportPeriodConfig struct {
	StartExpr   string // SQL expression for the range start (inclusive)
	EndExpr     string // SQL expression for the range end (exclusive)
	BucketTrunc string // generate_series/DATE_TRUNC unit: "hour" | "day"
	BucketLabel string // TO_CHAR format for the timeline label
}

// ReportPeriods are the only period keys GetReport accepts — validated by
// the service layer before this expressions ever reach SQL.
var ReportPeriods = map[string]ReportPeriodConfig{
	"today": {
		StartExpr: "DATE_TRUNC('day', NOW())", EndExpr: "DATE_TRUNC('day', NOW()) + INTERVAL '1 day'",
		BucketTrunc: "hour", BucketLabel: "HH24:00",
	},
	"yesterday": {
		StartExpr: "DATE_TRUNC('day', NOW()) - INTERVAL '1 day'", EndExpr: "DATE_TRUNC('day', NOW())",
		BucketTrunc: "hour", BucketLabel: "HH24:00",
	},
	"week": {
		StartExpr: "DATE_TRUNC('day', NOW()) - INTERVAL '6 days'", EndExpr: "DATE_TRUNC('day', NOW()) + INTERVAL '1 day'",
		BucketTrunc: "day", BucketLabel: "Mon DD",
	},
	"month": {
		StartExpr: "DATE_TRUNC('day', NOW()) - INTERVAL '29 days'", EndExpr: "DATE_TRUNC('day', NOW()) + INTERVAL '1 day'",
		BucketTrunc: "day", BucketLabel: "Mon DD",
	},
}

// GetReport aggregates audit_logs over the given period (validated against
// ReportPeriods by the caller) into per-user breakdowns, action totals, and
// a zero-filled timeline — mirrors the generate_series idiom already used in
// StatsRepository.GetActivity.
//
// userID scopes every query to a single user's own activity when non-empty
// (the normal case — every user generates their own report); pass "" for
// the system-wide, all-users view (admin oversight).
func (r *ReportRepository) GetReport(ctx context.Context, period string, userID string) (map[string]interface{}, error) {

	cfg := ReportPeriods[period]

	var whereUserFilter, joinUserFilter string
	var args []interface{}
	if userID != "" {
		whereUserFilter = "AND user_id = $1"
		joinUserFilter = "AND al.user_id = $1"
		args = []interface{}{userID}
	}

	data := make(map[string]interface{})
	data["period"] = period
	if userID != "" {
		data["scope"] = "own"
	} else {
		data["scope"] = "all"
	}

	// ── Per-user breakdown (a single row when scoped to one user) ──
	userRows, err := r.db.Query(ctx, fmt.Sprintf(`
		WITH daily AS (
			SELECT * FROM audit_logs
			WHERE created_at >= %s AND created_at < %s %s
		),
		by_user_action AS (
			SELECT user_id, action, COUNT(*) AS cnt FROM daily GROUP BY user_id, action
		),
		by_user_resource AS (
			SELECT user_id, resource_type, COUNT(*) AS cnt FROM daily GROUP BY user_id, resource_type
		),
		user_totals AS (
			SELECT user_id, COUNT(*) AS total_actions,
			       MIN(created_at) AS first_action_at, MAX(created_at) AS last_action_at
			FROM daily
			WHERE user_id IS NOT NULL
			GROUP BY user_id
		)
		SELECT
			u.id, u.name, u.email, ut.total_actions,
			COALESCE((SELECT jsonb_object_agg(action, cnt) FROM by_user_action bua WHERE bua.user_id = u.id), '{}'::jsonb),
			COALESCE((SELECT jsonb_object_agg(resource_type, cnt) FROM by_user_resource bur WHERE bur.user_id = u.id), '{}'::jsonb),
			ut.first_action_at, ut.last_action_at
		FROM user_totals ut
		JOIN users u ON u.id = ut.user_id
		ORDER BY ut.total_actions DESC
	`, cfg.StartExpr, cfg.EndExpr, whereUserFilter), args...)
	if err != nil {
		return nil, fmt.Errorf("report repo by-user: %w", err)
	}
	defer userRows.Close()

	byUser := make([]map[string]interface{}, 0)
	totalActions := 0
	for userRows.Next() {
		var userID, userName, userEmail string
		var actions int
		byAction := map[string]int{}
		byResourceType := map[string]int{}
		var firstAt, lastAt time.Time

		if err := userRows.Scan(&userID, &userName, &userEmail, &actions, &byAction, &byResourceType, &firstAt, &lastAt); err != nil {
			return nil, fmt.Errorf("report repo by-user scan: %w", err)
		}

		totalActions += actions
		byUser = append(byUser, map[string]interface{}{
			"user_id":          userID,
			"user_name":        userName,
			"user_email":       userEmail,
			"total_actions":    actions,
			"by_action":        byAction,
			"by_resource_type": byResourceType,
			"first_action_at":  firstAt,
			"last_action_at":   lastAt,
		})
	}
	if err := userRows.Err(); err != nil {
		return nil, fmt.Errorf("report repo by-user rows: %w", err)
	}

	data["by_user"] = byUser
	data["total_actions"] = totalActions
	data["active_users"] = len(byUser)

	// ── Totals by action ─────────────────────────────────────────
	actionRows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT action, COUNT(*)
		FROM audit_logs
		WHERE created_at >= %s AND created_at < %s %s
		GROUP BY action
		ORDER BY COUNT(*) DESC
	`, cfg.StartExpr, cfg.EndExpr, whereUserFilter), args...)
	if err != nil {
		return nil, fmt.Errorf("report repo action totals: %w", err)
	}
	defer actionRows.Close()

	byActionTotals := make(map[string]int)
	for actionRows.Next() {
		var action string
		var count int
		if err := actionRows.Scan(&action, &count); err != nil {
			return nil, fmt.Errorf("report repo action totals scan: %w", err)
		}
		byActionTotals[action] = count
	}
	if err := actionRows.Err(); err != nil {
		return nil, fmt.Errorf("report repo action totals rows: %w", err)
	}
	data["by_action_totals"] = byActionTotals

	// ── Timeline (zero-filled, hourly for today/yesterday, daily otherwise) ──
	timelineRows, err := r.db.Query(ctx, fmt.Sprintf(`
		WITH buckets AS (
			SELECT generate_series(
				DATE_TRUNC('%s', %s),
				DATE_TRUNC('%s', %s - INTERVAL '1 microsecond'),
				'1 %s'::interval
			) AS bucket
		)
		SELECT TO_CHAR(b.bucket, '%s') AS label, COUNT(al.id) AS count
		FROM buckets b
		LEFT JOIN audit_logs al ON DATE_TRUNC('%s', al.created_at) = b.bucket %s
		GROUP BY b.bucket
		ORDER BY b.bucket ASC
	`, cfg.BucketTrunc, cfg.StartExpr, cfg.BucketTrunc, cfg.EndExpr, cfg.BucketTrunc, cfg.BucketLabel, cfg.BucketTrunc, joinUserFilter), args...)
	if err != nil {
		return nil, fmt.Errorf("report repo timeline: %w", err)
	}
	defer timelineRows.Close()

	timeline := make([]map[string]interface{}, 0)
	for timelineRows.Next() {
		var label string
		var count int
		if err := timelineRows.Scan(&label, &count); err != nil {
			return nil, fmt.Errorf("report repo timeline scan: %w", err)
		}
		timeline = append(timeline, map[string]interface{}{"label": label, "count": count})
	}
	if err := timelineRows.Err(); err != nil {
		return nil, fmt.Errorf("report repo timeline rows: %w", err)
	}
	data["timeline"] = timeline

	return data, nil
}
