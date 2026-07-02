package models

import "time"

type UserTwoFactor struct {
	UserID    string    `json:"user_id"`
	Secret    string    `json:"-"` // do not expose this in JSON responses
	Enabled   bool      `json:"enabled"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
