package models

import "time"

type Document struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`

	FileName string `json:"file_name"`
	FileKey  string `json:"file_key"`
	FileType string `json:"file_type"`
	FileSize int64  `json:"file_size"`

	FolderID *string `json:"folder_id,omitempty"`
	OwnerID  string  `json:"owner_id"`

	Status     string     `json:"status"`
	IsStarred  bool       `json:"is_starred"`
	Version    int        `json:"version"`
	LastAccess *time.Time `json:"last_accessed,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
