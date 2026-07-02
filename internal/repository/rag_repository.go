package repository

import (
	"context"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type RAGRepository struct {
	db *pgxpool.Pool
}

func NewRAGRepository(db *pgxpool.Pool) *RAGRepository {
	return &RAGRepository{db: db}
}

// ─── Chunks ──────────────────────────────────────────────────────────────────

// DeleteChunks removes all existing chunks for a document (before re-indexing).
func (r *RAGRepository) DeleteChunks(ctx context.Context, documentID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM document_chunks WHERE document_id = $1`, documentID)
	return err
}

// SaveChunk inserts a single chunk with its embedding.
func (r *RAGRepository) SaveChunk(ctx context.Context, documentID string, index int, content string, embedding []float32) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO document_chunks (document_id, chunk_index, content, embedding)
		 VALUES ($1, $2, $3, $4)`,
		documentID, index, content, pgvector.NewVector(embedding),
	)
	if err != nil {
		return fmt.Errorf("save chunk: %w", err)
	}
	return nil
}

// SearchChunks finds the topK most similar chunks to the query embedding.
// It also joins the documents table to return document metadata.
func (r *RAGRepository) SearchChunks(ctx context.Context, userID string, embedding []float32, topK int) ([]models.ChunkResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			dc.id,
			dc.document_id,
			d.title,
			d.file_name,
			dc.chunk_index,
			dc.content,
			1 - (dc.embedding <=> $1) AS similarity
		FROM document_chunks dc
		JOIN documents d ON d.id = dc.document_id
		WHERE d.owner_id = $2
		  AND d.deleted_at IS NULL
		  AND dc.embedding IS NOT NULL
		ORDER BY dc.embedding <=> $1
		LIMIT $3
	`, pgvector.NewVector(embedding), userID, topK)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()

	var results []models.ChunkResult
	for rows.Next() {
		var cr models.ChunkResult
		if err := rows.Scan(
			&cr.ChunkID, &cr.DocumentID, &cr.DocumentTitle,
			&cr.FileName, &cr.ChunkIndex, &cr.Content, &cr.Similarity,
		); err != nil {
			return nil, err
		}
		results = append(results, cr)
	}
	return results, nil
}

// ─── Chat Sessions ────────────────────────────────────────────────────────────

func (r *RAGRepository) CreateSession(ctx context.Context, userID, title string) (*models.ChatSession, error) {
	var s models.ChatSession
	err := r.db.QueryRow(ctx,
		`INSERT INTO chat_sessions (user_id, title)
		 VALUES ($1, $2)
		 RETURNING id, user_id, title, created_at, updated_at`,
		userID, title,
	).Scan(&s.ID, &s.UserID, &s.Title, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &s, nil
}

func (r *RAGRepository) ListSessions(ctx context.Context, userID string) ([]models.ChatSession, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, title, created_at, updated_at
		 FROM chat_sessions
		 WHERE user_id = $1
		 ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.ChatSession
	for rows.Next() {
		var s models.ChatSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.Title, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *RAGRepository) GetSession(ctx context.Context, sessionID, userID string) (*models.ChatSession, error) {
	var s models.ChatSession
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, title, created_at, updated_at
		 FROM chat_sessions WHERE id = $1 AND user_id = $2`,
		sessionID, userID,
	).Scan(&s.ID, &s.UserID, &s.Title, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *RAGRepository) DeleteSession(ctx context.Context, sessionID, userID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM chat_sessions WHERE id = $1 AND user_id = $2`,
		sessionID, userID)
	return err
}

func (r *RAGRepository) UpdateSessionTitle(ctx context.Context, sessionID, title string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE chat_sessions SET title = $1 WHERE id = $2`, title, sessionID)
	return err
}

// ─── Chat Messages ────────────────────────────────────────────────────────────

func (r *RAGRepository) SaveMessage(ctx context.Context, msg *models.ChatMessage) (*models.ChatMessage, error) {
	var m models.ChatMessage
	err := r.db.QueryRow(ctx,
		`INSERT INTO chat_messages (session_id, role, content, sources)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, session_id, role, content, sources, created_at`,
		msg.SessionID, msg.Role, msg.Content, msg.Sources,
	).Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.Sources, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("save message: %w", err)
	}
	return &m, nil
}

func (r *RAGRepository) ListMessages(ctx context.Context, sessionID string) ([]models.ChatMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, session_id, role, content, sources, created_at
		 FROM chat_messages WHERE session_id = $1 ORDER BY created_at ASC`,
		sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.Sources, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}
