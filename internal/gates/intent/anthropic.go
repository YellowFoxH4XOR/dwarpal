package intent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// defaultAnthropicModel is used when NewAnthropicProvider is given an empty
// model string.
const defaultAnthropicModel = "claude-sonnet-5"

// anthropicAPIVersion is the Messages API version this provider speaks.
const anthropicAPIVersion = "2023-06-01"

// AnthropicProvider is a thin Provider implementation for the Anthropic
// Messages API (POST /v1/messages). It is deliberately minimal: one request,
// one response, structured verdict fields expected as a JSON object in the
// model's reply text — same contract as OpenAIProvider.
//
// It is never exercised against the real network in this package's tests;
// tests override BaseURL with an httptest.Server.
type AnthropicProvider struct {
	BaseURL string // e.g. "https://api.anthropic.com/v1/messages"
	Model   string
	APIKey  string

	httpClient *http.Client
}

// NewAnthropicProvider builds a provider targeting the Anthropic Messages
// API. If model is empty, it defaults to claude-sonnet-5.
func NewAnthropicProvider(model, apiKey string) *AnthropicProvider {
	if model == "" {
		model = defaultAnthropicModel
	}
	return &AnthropicProvider{
		BaseURL:    "https://api.anthropic.com/v1/messages",
		Model:      model,
		APIKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// Name identifies the provider for logging/diagnostics.
func (p *AnthropicProvider) Name() string { return "anthropic:" + p.Model }

type anthropicMessageRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicMessageResponse struct {
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Verify sends the prompt to the configured Anthropic endpoint and parses
// the model's JSON reply (found in the first text content block) into a
// Verdict. Any network, HTTP, or parse error is returned to the caller,
// which (per Gate.Run) treats it as an infra failure and fails open.
func (p *AnthropicProvider) Verify(ctx context.Context, prompt string) (Verdict, error) {
	fullPrompt := "Reply with a single JSON object: " +
		`{"accomplishes_intent": bool, "only_stated_intent": bool, "surprises": [string]}. No prose.` +
		"\n\n" + prompt

	reqBody := anthropicMessageRequest{
		Model:     p.Model,
		MaxTokens: 1024,
		Messages: []anthropicMessage{
			{Role: "user", Content: fullPrompt},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return Verdict{}, fmt.Errorf("intent: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL, bytes.NewReader(body))
	if err != nil {
		return Verdict{}, fmt.Errorf("intent: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	if p.APIKey != "" {
		req.Header.Set("x-api-key", p.APIKey)
	}

	client := p.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return Verdict{}, fmt.Errorf("intent: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Verdict{}, fmt.Errorf("intent: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return Verdict{}, fmt.Errorf("intent: provider returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var msgResp anthropicMessageResponse
	if err := json.Unmarshal(respBytes, &msgResp); err != nil {
		return Verdict{}, fmt.Errorf("intent: unmarshal response: %w", err)
	}

	var raw string
	for _, block := range msgResp.Content {
		if block.Type == "text" {
			raw = block.Text
			break
		}
	}
	if raw == "" {
		return Verdict{}, fmt.Errorf("intent: provider returned no text content block")
	}

	var vj verdictJSON
	if err := json.Unmarshal([]byte(raw), &vj); err != nil {
		return Verdict{}, fmt.Errorf("intent: unmarshal verdict JSON: %w", err)
	}

	return Verdict{
		AccomplishesIntent: vj.AccomplishesIntent,
		OnlyStatedIntent:   vj.OnlyStatedIntent,
		Surprises:          vj.Surprises,
		Raw:                raw,
	}, nil
}
