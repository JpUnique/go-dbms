package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const geminiBase = "https://generativelanguage.googleapis.com/v1beta"

// ── Gemini Embeddings ─────────────────────────────────────────────────────────

type geminiEmbed struct {
	apiKey string
	model  string
}

func (g *geminiEmbed) Embed(ctx context.Context, text string) ([]float32, error) {
	results, err := g.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (g *geminiEmbed) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Parts []part `json:"parts"`
	}
	type embedReq struct {
		Model   string  `json:"model"`
		Content content `json:"content"`
	}
	type batchReq struct {
		Requests []embedReq `json:"requests"`
	}
	type embedValues struct {
		Values []float32 `json:"values"`
	}
	type batchResp struct {
		Embeddings []embedValues `json:"embeddings"`
		Error      *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	reqs := make([]embedReq, len(texts))
	for i, t := range texts {
		reqs[i] = embedReq{
			Model:   "models/" + g.model,
			Content: content{Parts: []part{{Text: t}}},
		}
	}

	body, _ := json.Marshal(batchReq{Requests: reqs})
	url := fmt.Sprintf("%s/models/%s:batchEmbedContents?key=%s", geminiBase, g.model, g.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini embed: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini embed: http: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var br batchResp
	if err := json.Unmarshal(raw, &br); err != nil {
		return nil, fmt.Errorf("gemini embed: unmarshal: %w", err)
	}
	if br.Error != nil {
		return nil, fmt.Errorf("gemini embed API error: %s", br.Error.Message)
	}

	result := make([][]float32, len(br.Embeddings))
	for i, e := range br.Embeddings {
		result[i] = e.Values
	}
	return result, nil
}

// ── Gemini Chat ───────────────────────────────────────────────────────────────

type geminiChat struct {
	apiKey string
	model  string
}

func (g *geminiChat) Chat(ctx context.Context, system string, history []Turn) (string, error) {
	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Role  string `json:"role,omitempty"`
		Parts []part `json:"parts"`
	}
	type systemInstruction struct {
		Parts []part `json:"parts"`
	}
	type reqBody struct {
		SystemInstruction *systemInstruction `json:"system_instruction,omitempty"`
		Contents          []content          `json:"contents"`
	}
	type candidate struct {
		Content content `json:"content"`
	}
	type respBody struct {
		Candidates []candidate `json:"candidates"`
		Error      *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	var body reqBody
	if system != "" {
		body.SystemInstruction = &systemInstruction{Parts: []part{{Text: system}}}
	}
	for _, t := range history {
		role := "user"
		if t.Role == "assistant" {
			role = "model"
		}
		body.Contents = append(body.Contents, content{
			Role:  role,
			Parts: []part{{Text: t.Content}},
		})
	}

	raw, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", geminiBase, g.model, g.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("gemini chat: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini chat: http: %w", err)
	}
	defer resp.Body.Close()

	respRaw, _ := io.ReadAll(resp.Body)
	var rb respBody
	if err := json.Unmarshal(respRaw, &rb); err != nil {
		return "", fmt.Errorf("gemini chat: unmarshal: %w", err)
	}
	if rb.Error != nil {
		return "", fmt.Errorf("gemini chat API error: %s", rb.Error.Message)
	}
	if len(rb.Candidates) == 0 || len(rb.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini chat: empty response")
	}
	return rb.Candidates[0].Content.Parts[0].Text, nil
}
