package models

import "time"

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // never expose
	Name         string     `json:"name"`
	Role         string     `json:"role"`
	Department   *string    `json:"department,omitempty"`
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	Status       string     `json:"status"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
