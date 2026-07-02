package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// anthropicChat calls the Anthropic Messages API.
// This is a paid provider — kept here for when you have API credits.
type anthropicChat struct {
	apiKey string
	model  string
}

func (a *anthropicChat) Chat(ctx context.Context, system string, history []Turn) (string, error) {
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type reqBody struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		System    string `json:"system,omitempty"`
		Messages  []msg  `json:"messages"`
	}
	type contentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type respBody struct {
		Content []contentBlock `json:"content"`
		Error   *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	var msgs []msg
	for _, t := range history {
		role := "user"
		if t.Role == "assistant" {
			role = "assistant"
		}
		msgs = append(msgs, msg{Role: role, Content: t.Content})
	}

	payload := reqBody{
		Model:     a.model,
		MaxTokens: 1024,
		System:    system,
		Messages:  msgs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("anthropic chat: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("anthropic chat: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic chat: http: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var rb respBody
	if err := json.Unmarshal(raw, &rb); err != nil {
		return "", fmt.Errorf("anthropic chat: unmarshal: %w", err)
	}
	if rb.Error != nil {
		return "", fmt.Errorf("anthropic API error: %s", rb.Error.Message)
	}
	for _, block := range rb.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("anthropic chat: no text content in response")
}
