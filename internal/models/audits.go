package models

import "time"

type AuditLog struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"user_id,omitempty"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   *string   `json:"resource_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`

	// joined fields
	UserName  *string `json:"user_name,omitempty"`
	UserEmail *string `json:"user_email,omitempty"`
}
