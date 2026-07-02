package models

import "time"

type DocumentReview struct {
	ID           string     `json:"id"`
	DocumentID   string     `json:"document_id"`
	DocumentTitle string    `json:"document_title,omitempty"`
	SubmitterID  string     `json:"submitter_id"`
	SubmitterName string    `json:"submitter_name,omitempty"`
	ReviewerID   *string    `json:"reviewer_id,omitempty"`
	ReviewerName *string    `json:"reviewer_name,omitempty"`
	Decision     string     `json:"decision"`
	Note         *string    `json:"note,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
}
