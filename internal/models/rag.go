package models

import "time"

// ChunkResult is returned from a vector similarity search.
type ChunkResult struct {
	ChunkID       string  `json:"chunk_id"`
	DocumentID    string  `json:"document_id"`
	DocumentTitle string  `json:"document_title"`
	FileName      string  `json:"file_name"`
	ChunkIndex    int     `json:"chunk_index"`
	Content       string  `json:"content"`
	Similarity    float64 `json:"similarity"`
}

// ChatSession groups a conversation between a user and the RAG assistant.
type ChatSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatMessage is a single turn in a chat session.
type ChatMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"` // "user" | "assistant"
	Content   string    `json:"content"`
	Sources   []byte    `json:"sources,omitempty"` // JSONB: []ChunkResult
	CreatedAt time.Time `json:"created_at"`
}
