package rag

import (
	"context"
	"os"
)

// Turn is a single message in a conversation history.
type Turn struct {
	Role    string // "user" | "assistant"
	Content string
}

// EmbedProvider converts text into float32 vectors.
type EmbedProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// ChatProvider sends a prompt to an LLM and returns the reply.
type ChatProvider interface {
	Chat(ctx context.Context, system string, history []Turn) (string, error)
}

// NewEmbedProvider reads RAG_EMBED_PROVIDER and returns the right implementation.
//
//	RAG_EMBED_PROVIDER=ollama  (default) — local Ollama, no key needed
//	RAG_EMBED_PROVIDER=gemini            — Google Gemini free tier
func NewEmbedProvider() EmbedProvider {
	switch os.Getenv("RAG_EMBED_PROVIDER") {
	case "gemini":
		return &geminiEmbed{
			apiKey: os.Getenv("GEMINI_API_KEY"),
			model:  envOr("GEMINI_EMBED_MODEL", "text-embedding-004"),
		}
	default: // "ollama" or unset
		return &ollamaEmbed{
			baseURL: envOr("OLLAMA_BASE_URL", "http://localhost:11434"),
			model:   envOr("OLLAMA_EMBED_MODEL", "nomic-embed-text"),
		}
	}
}

// NewChatProvider reads RAG_CHAT_PROVIDER and returns the right implementation.
//
//	RAG_CHAT_PROVIDER=ollama     (default) — local Ollama, no key needed
//	RAG_CHAT_PROVIDER=gemini               — Google Gemini free tier
//	RAG_CHAT_PROVIDER=groq                 — Groq free tier (fast)
//	RAG_CHAT_PROVIDER=anthropic            — Anthropic Claude (paid)
func NewChatProvider() ChatProvider {
	switch os.Getenv("RAG_CHAT_PROVIDER") {
	case "gemini":
		return &geminiChat{
			apiKey: os.Getenv("GEMINI_API_KEY"),
			model:  envOr("GEMINI_CHAT_MODEL", "gemini-1.5-flash"),
		}
	case "groq":
		return &groqChat{
			apiKey: os.Getenv("GROQ_API_KEY"),
			model:  envOr("GROQ_CHAT_MODEL", "llama-3.1-8b-instant"),
		}
	case "anthropic":
		return &anthropicChat{
			apiKey: os.Getenv("ANTHROPIC_API_KEY"),
			model:  envOr("ANTHROPIC_CHAT_MODEL", "claude-haiku-4-5-20251001"),
		}
	default: // "ollama" or unset
		return &ollamaChat{
			baseURL: envOr("OLLAMA_BASE_URL", "http://localhost:11434"),
			model:   envOr("OLLAMA_CHAT_MODEL", "llama3.2"),
		}
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
