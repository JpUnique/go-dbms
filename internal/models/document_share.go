package models

import "time"

type DocumentShare struct {
	ID           string     `json:"id"`
	DocumentID   string     `json:"document_id"`
	ShareToken   string     `json:"share_token"`
	SharedBy     string     `json:"shared_by"`
	Permission   string     `json:"permission"`
	PasswordHash *string    `json:"password_hash,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	AccessCount  int        `json:"access_count"`

	CreatedAt time.Time `json:"created_at"`
}
