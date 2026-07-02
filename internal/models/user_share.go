package models

import "time"

// UserShare is a raw row in document_user_shares.
type UserShare struct {
	DocumentID string    `json:"document_id"`
	UserID     string    `json:"user_id"`
	Permission string    `json:"permission"`
	SharedBy   *string   `json:"shared_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ShareRecipient is a user a document has been shared with — used to render
// "who this document is shared with" on the document's Share page.
type ShareRecipient struct {
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Permission string    `json:"permission"`
	SharedAt   time.Time `json:"shared_at"`
}

// SharedDocument is a document shared with the current user — used to
// render the "Shared with Me" list.
type SharedDocument struct {
	Document
	SharedByName string    `json:"shared_by_name"`
	Permission   string    `json:"permission"`
	SharedAt     time.Time `json:"shared_at"`
}
