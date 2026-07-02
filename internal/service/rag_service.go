package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/rag"
	"github.com/JpUnique/go-dbms/internal/repository"
)

type RAGService struct {
	repo     *repository.RAGRepository
	embedder rag.EmbedProvider
	chat     rag.ChatProvider
}

func NewRAGService(repo *repository.RAGRepository) *RAGService {
	return &RAGService{
		repo:     repo,
		embedder: rag.NewEmbedProvider(),
		chat:     rag.NewChatProvider(),
	}
}

// ─── Indexing ─────────────────────────────────────────────────────────────────

// IndexDocument extracts text, chunks it, embeds each chunk, and stores vectors.
// Safe to call in a goroutine — logs errors instead of propagating them.
func (s *RAGService) IndexDocument(ctx context.Context, documentID, fileExt string, fileData []byte) {
	text, err := rag.ExtractText(fileData, fileExt)
	if err != nil {
		log.Printf("rag index [%s]: extract: %v", documentID, err)
		return
	}

	chunks := rag.Chunk(text)
	if len(chunks) == 0 {
		log.Printf("rag index [%s]: no chunks produced", documentID)
		return
	}

	embeddings, err := s.embedder.EmbedBatch(ctx, chunks)
	if err != nil {
		log.Printf("rag index [%s]: embed: %v", documentID, err)
		return
	}

	if err := s.repo.DeleteChunks(ctx, documentID); err != nil {
		log.Printf("rag index [%s]: delete old chunks: %v", documentID, err)
	}

	for i, chunk := range chunks {
		if i >= len(embeddings) || embeddings[i] == nil {
			continue
		}
		if err := s.repo.SaveChunk(ctx, documentID, i, chunk, embeddings[i]); err != nil {
			log.Printf("rag index [%s]: save chunk %d: %v", documentID, i, err)
		}
	}

	log.Printf("rag index [%s]: indexed %d chunks", documentID, len(chunks))
}

// ─── Chat ─────────────────────────────────────────────────────────────────────

type AskRequest struct {
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
}

type AskResponse struct {
	SessionID string               `json:"session_id"`
	Answer    string               `json:"answer"`
	Sources   []models.ChunkResult `json:"sources"`
}

// Ask runs the full RAG pipeline:
//  1. Embed the question
//  2. Retrieve the most relevant document chunks
//  3. Send them + conversation history to the LLM
//  4. Persist both messages and return the answer with citations
func (s *RAGService) Ask(ctx context.Context, userID string, req AskRequest) (*AskResponse, error) {
	if strings.TrimSpace(req.Question) == "" {
		return nil, fmt.Errorf("question cannot be empty")
	}

	// 1 — Ensure a session exists
	sessionID := req.SessionID
	if sessionID == "" {
		title := req.Question
		if len(title) > 60 {
			title = title[:60] + "…"
		}
		sess, err := s.repo.CreateSession(ctx, userID, title)
		if err != nil {
			return nil, fmt.Errorf("create session: %w", err)
		}
		sessionID = sess.ID
	}

	// 2 — Embed the question
	qEmbedding, err := s.embedder.Embed(ctx, req.Question)
	if err != nil {
		return nil, fmt.Errorf("embed question: %w", err)
	}

	// 3 — Retrieve relevant chunks (top 5)
	chunks, err := s.repo.SearchChunks(ctx, userID, qEmbedding, 5)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}

	// 4 — Build context block from retrieved chunks
	var contextParts []string
	for _, c := range chunks {
		contextParts = append(contextParts, fmt.Sprintf("[Document: %s]\n%s", c.DocumentTitle, c.Content))
	}
	contextBlock := strings.Join(contextParts, "\n\n---\n\n")

	// 5 — Load recent conversation history (last 10 turns)
	history, _ := s.repo.ListMessages(ctx, sessionID)
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	// 6 — Build turns for the LLM
	var turns []rag.Turn
	for _, m := range history {
		turns = append(turns, rag.Turn{Role: m.Role, Content: m.Content})
	}

	userContent := req.Question
	if contextBlock != "" {
		userContent = fmt.Sprintf(
			"Use the following document excerpts to answer the question. "+
				"If the answer is not in the documents, say so clearly.\n\n"+
				"DOCUMENT CONTEXT:\n%s\n\nQUESTION: %s",
			contextBlock, req.Question,
		)
	}
	turns = append(turns, rag.Turn{Role: "user", Content: userContent})

	// 7 — Call LLM via the configured chat provider
	systemPrompt := "You are a helpful assistant for PETRODATA, a document management system. " +
		"Answer questions based on the provided document context. " +
		"Always cite which document your answer comes from. " +
		"If the context does not contain enough information, say so clearly."

	answer, err := s.chat.Chat(ctx, systemPrompt, turns)
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}

	// 8 — Persist user message
	if _, err := s.repo.SaveMessage(ctx, &models.ChatMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   req.Question,
	}); err != nil {
		log.Printf("rag ask: save user message: %v", err)
	}

	// 9 — Persist assistant reply with sources
	sourcesJSON, _ := json.Marshal(chunks)
	if _, err := s.repo.SaveMessage(ctx, &models.ChatMessage{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   answer,
		Sources:   sourcesJSON,
	}); err != nil {
		log.Printf("rag ask: save assistant message: %v", err)
	}

	return &AskResponse{
		SessionID: sessionID,
		Answer:    answer,
		Sources:   chunks,
	}, nil
}

// ─── Session helpers ──────────────────────────────────────────────────────────

func (s *RAGService) CreateSession(ctx context.Context, userID, title string) (*models.ChatSession, error) {
	if title == "" {
		title = "New Chat"
	}
	return s.repo.CreateSession(ctx, userID, title)
}

func (s *RAGService) ListSessions(ctx context.Context, userID string) ([]models.ChatSession, error) {
	return s.repo.ListSessions(ctx, userID)
}

func (s *RAGService) GetSession(ctx context.Context, sessionID, userID string) (*models.ChatSession, []models.ChatMessage, error) {
	sess, err := s.repo.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, nil, err
	}
	msgs, err := s.repo.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}
	return sess, msgs, nil
}

func (s *RAGService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	return s.repo.DeleteSession(ctx, sessionID, userID)
}
