package models

import "time"

type DocumentVersion struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	Version    int       `json:"version"`
	FileKey    string    `json:"file_key"`
	FileSize   int64     `json:"file_size"`
	UploadedBy *string   `json:"uploaded_by,omitempty"`
	ChangeNote *string   `json:"change_note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type DocumentQuery struct {
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
	Search   string `form:"search"`
	Status   string `form:"status"`
	Starred  string `form:"starred"`   // "true" | "false" | "" (empty = no filter)
	FolderID string `form:"folder_id"` // UUID string; empty = no filter
}
