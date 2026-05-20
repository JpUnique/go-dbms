package models

import "time"

type Folder struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	OwnerID    string  `json:"owner_id"`
	ParentID   *string `json:"parent_id,omitempty"`
	Department *string `json:"department,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// computed fields
	DocumentCount  int `json:"document_count,omitempty"`
	SubfolderCount int `json:"subfolder_count,omitempty"`
}
