package utils

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditEntry represents an audit log entry
type AuditEntry struct {
	UserID       *string
	Action       string
	ResourceType string
	ResourceID   *string
	Details      map[string]interface{}
	IPAddress    *string
	UserAgent    *string
}

// LogAudit inserts audit log into database
func LogAudit(ctx context.Context, db *pgxpool.Pool, entry AuditEntry) {

	var detailsJSON []byte
	var err error

	if entry.Details != nil {
		detailsJSON, err = json.Marshal(entry.Details)
		if err != nil {
			log.Println("[AUDIT] failed to marshal details:", err)
			return
		}
	}

	_, err = db.Exec(ctx,
		`INSERT INTO audit_logs
        (user_id, action, resource_type, resource_id, details, ip_address, user_agent)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.UserID,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		detailsJSON,
		entry.IPAddress,
		entry.UserAgent,
	)

	if err != nil {
		log.Println("[AUDIT] insert failed:", err)
	}
}
