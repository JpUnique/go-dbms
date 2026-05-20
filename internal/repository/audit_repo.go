package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) GetAll(
	ctx context.Context,
	userID string,
	isAdmin bool,
	userFilter string,
	resourceType string,
	resourceID string,
	action string,
	limit int,
	offset int,
) ([]models.AuditLog, error) {

	conditions := []string{}
	args := []interface{}{}
	idx := 1

	// ✅ restrict non admin
	if !isAdmin {
		conditions = append(conditions, fmt.Sprintf("al.user_id = $%d", idx))
		args = append(args, userID)
		idx++
	} else if userFilter != "" {
		conditions = append(conditions, fmt.Sprintf("al.user_id = $%d", idx))
		args = append(args, userFilter)
		idx++
	}

	if resourceType != "" {
		conditions = append(conditions, fmt.Sprintf("al.resource_type = $%d", idx))
		args = append(args, resourceType)
		idx++
	}

	if resourceID != "" {
		conditions = append(conditions, fmt.Sprintf("al.resource_id = $%d", idx))
		args = append(args, resourceID)
		idx++
	}

	if action != "" {
		conditions = append(conditions, fmt.Sprintf("al.action = $%d", idx))
		args = append(args, action)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	args = append(args, limit, offset)

	query := fmt.Sprintf(`
    SELECT al.id, al.user_id, al.action, al.resource_type,
           al.resource_id, al.created_at,
           u.name, u.email
    FROM audit_logs al
    LEFT JOIN users u ON u.id = al.user_id
    %s
    ORDER BY al.created_at DESC
    LIMIT $%d OFFSET $%d
    `, where, idx, idx+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("audit repo get: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog

	for rows.Next() {
		var l models.AuditLog

		err := rows.Scan(
			&l.ID,
			&l.UserID,
			&l.Action,
			&l.ResourceType,
			&l.ResourceID,
			&l.CreatedAt,
			&l.UserName,
			&l.UserEmail,
		)
		if err != nil {
			return nil, err
		}

		logs = append(logs, l)
	}

	return logs, nil
}

func (r *AuditRepository) Delete(
	ctx context.Context,
	before string,
) error {

	if before != "" {
		_, err := r.db.Exec(ctx,
			"DELETE FROM audit_logs WHERE created_at < $1",
			before,
		)
		return err
	}

	_, err := r.db.Exec(ctx, "TRUNCATE audit_logs")

	return err
}
