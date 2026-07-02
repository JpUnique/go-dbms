package rag

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// ── Ollama Embeddings ─────────────────────────────────────────────────────────
// Uses Ollama's OpenAI-compatible /v1/embeddings endpoint.

type ollamaEmbed struct {
	baseURL string
	model   string
}

func (o *ollamaEmbed) newClient() *openai.Client {
	cfg := openai.DefaultConfig("ollama")
	cfg.BaseURL = o.baseURL + "/v1"
	return openai.NewClientWithConfig(cfg)
}

func (o *ollamaEmbed) Embed(ctx context.Context, text string) ([]float32, error) {
	results, err := o.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (o *ollamaEmbed) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	resp, err := o.newClient().CreateEmbeddings(ctx, openai.EmbeddingRequestStrings{
		Input: texts,
		Model: openai.EmbeddingModel(o.model),
	})
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	result := make([][]float32, len(resp.Data))
	for _, d := range resp.Data {
		result[d.Index] = d.Embedding
	}
	return result, nil
}

// ── Ollama Chat ───────────────────────────────────────────────────────────────
// Uses Ollama's OpenAI-compatible /v1/chat/completions endpoint.

type ollamaChat struct {
	baseURL string
	model   string
}

func (o *ollamaChat) newClient() *openai.Client {
	cfg := openai.DefaultConfig("ollama")
	cfg.BaseURL = o.baseURL + "/v1"
	return openai.NewClientWithConfig(cfg)
}

func (o *ollamaChat) Chat(ctx context.Context, system string, history []Turn) (string, error) {
	var msgs []openai.ChatCompletionMessage
	if system != "" {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: system,
		})
	}
	for _, t := range history {
		role := openai.ChatMessageRoleUser
		if t.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}
		msgs = append(msgs, openai.ChatCompletionMessage{Role: role, Content: t.Content})
	}

	resp, err := o.newClient().CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    o.model,
		Messages: msgs,
	})
	if err != nil {
		return "", fmt.Errorf("ollama chat: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("ollama chat: empty response")
	}
	return resp.Choices[0].Message.Content, nil
}
