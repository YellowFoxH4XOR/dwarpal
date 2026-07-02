package intent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAIProvider is a thin Provider implementation for OpenAI-compatible
// chat-completions endpoints (OpenAI itself, and any BYO-key gateway that
// mirrors its API shape). It is deliberately minimal: one request, one
// response, structured verdict fields expected as a JSON object in the
// model's reply.
//
// It is never exercised by this package's tests (no network in tests) —
// tests use mockProvider instead.
type OpenAIProvider struct {
	Endpoint string // e.g. "https://api.openai.com/v1/chat/completions"
	Model    string
	APIKey   string

	httpClient *http.Client
}

// NewOpenAIProvider builds a provider targeting an OpenAI-compatible endpoint.
func NewOpenAIProvider(endpoint, model, apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		Endpoint:   endpoint,
		Model:      model,
		APIKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// Name identifies the provider for logging/diagnostics.
func (p *OpenAIProvider) Name() string { return "openai:" + p.Model }

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
}

// verdictJSON is the structured shape we ask the model to reply with.
type verdictJSON struct {
	AccomplishesIntent bool     `json:"accomplishes_intent"`
	OnlyStatedIntent   bool     `json:"only_stated_intent"`
	Surprises          []string `json:"surprises"`
}

// Verify sends the prompt to the configured endpoint and parses the model's
// JSON reply into a Verdict. Any network, HTTP, or parse error is returned to
// the caller, which (per Gate.Run) treats it as an infra failure and fails
// open.
func (p *OpenAIProvider) Verify(ctx context.Context, prompt string) (Verdict, error) {
	reqBody := openAIChatRequest{
		Model: p.Model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: "Reply with a single JSON object: " +
				`{"accomplishes_intent": bool, "only_stated_intent": bool, "surprises": [string]}. No prose.`},
			{Role: "user", Content: prompt},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return Verdict{}, fmt.Errorf("intent: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(body))
	if err != nil {
		return Verdict{}, fmt.Errorf("intent: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
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

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return Verdict{}, fmt.Errorf("intent: unmarshal response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return Verdict{}, fmt.Errorf("intent: provider returned no choices")
	}

	raw := chatResp.Choices[0].Message.Content
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
