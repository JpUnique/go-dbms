package rag

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// groqChat uses Groq's OpenAI-compatible API for chat completion.
// Sign up free at console.groq.com — no credit card required.
// Note: Groq does not provide an embedding API; use ollama or gemini for embeddings.
type groqChat struct {
	apiKey string
	model  string
}

func (g *groqChat) newClient() *openai.Client {
	cfg := openai.DefaultConfig(g.apiKey)
	cfg.BaseURL = "https://api.groq.com/openai/v1"
	return openai.NewClientWithConfig(cfg)
}

func (g *groqChat) Chat(ctx context.Context, system string, history []Turn) (string, error) {
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

	resp, err := g.newClient().CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    g.model,
		Messages: msgs,
	})
	if err != nil {
		return "", fmt.Errorf("groq chat: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("groq chat: empty response")
	}
	return resp.Choices[0].Message.Content, nil
}
