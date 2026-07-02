package models

import "time"

type UserPreferences struct {
	UserID             string    `json:"user_id"`
	DarkMode           bool      `json:"dark_mode"`
	EmailNotifications bool      `json:"email_notifications"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
