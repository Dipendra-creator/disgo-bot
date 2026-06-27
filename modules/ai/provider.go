package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// anthropicVersion is the API version header value (see Anthropic API docs).
const anthropicVersion = "2023-06-01"

// Message is one turn in a conversation handed to the provider.
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// Provider abstracts the LLM backend so it can be swapped or mocked in tests.
type Provider interface {
	// Complete returns the assistant's reply to msgs under the given system prompt.
	Complete(ctx context.Context, system string, msgs []Message) (string, error)
	// Model returns the configured model identifier.
	Model() string
}

// anthropicProvider calls the Anthropic Messages API (POST /v1/messages).
type anthropicProvider struct {
	apiKey    string
	model     string
	maxTokens int
	baseURL   string
	http      *http.Client
}

func newAnthropicProvider(apiKey, model string, maxTokens int, baseURL string) *anthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	return &anthropicProvider{
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		baseURL:   strings.TrimRight(baseURL, "/"),
		http:      &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *anthropicProvider) Model() string { return p.model }

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (p *anthropicProvider) Complete(ctx context.Context, system string, msgs []Message) (string, error) {
	reqBody := anthropicRequest{Model: p.model, MaxTokens: p.maxTokens, System: system}
	for _, m := range msgs {
		reqBody.Messages = append(reqBody.Messages, anthropicMessage{Role: m.Role, Content: m.Content})
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var out anthropicResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		if out.Error != nil {
			return "", fmt.Errorf("anthropic %s: %s", out.Error.Type, out.Error.Message)
		}
		return "", fmt.Errorf("anthropic request failed with status %d", resp.StatusCode)
	}
	return extractText(out), nil
}

// extractText concatenates the text blocks of a Messages API response.
func extractText(r anthropicResponse) string {
	var b strings.Builder
	for _, c := range r.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return strings.TrimSpace(b.String())
}
