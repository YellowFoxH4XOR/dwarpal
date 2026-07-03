package intent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAnthropicProvider_DefaultModel(t *testing.T) {
	p := NewAnthropicProvider("", "key")
	if p.Model != defaultAnthropicModel {
		t.Fatalf("Model = %q, want default %q", p.Model, defaultAnthropicModel)
	}

	p2 := NewAnthropicProvider("claude-opus-4-8", "key")
	if p2.Model != "claude-opus-4-8" {
		t.Fatalf("Model = %q, want %q", p2.Model, "claude-opus-4-8")
	}
}

func TestAnthropicProvider_Name(t *testing.T) {
	p := NewAnthropicProvider("claude-sonnet-5", "key")
	if got, want := p.Name(), "anthropic:claude-sonnet-5"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
}

func TestAnthropicProvider_Verify_RequestShapeAndParsing(t *testing.T) {
	var gotPath string
	var gotHeaders http.Header
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeaders = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		resp := map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": `{"accomplishes_intent": true, "only_stated_intent": false, "surprises": ["renamed unrelated function"]}`,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	p := NewAnthropicProvider("claude-sonnet-5", "test-api-key")
	p.BaseURL = srv.URL + "/v1/messages"

	v, err := p.Verify(context.Background(), "does this diff do the thing?")
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	// Request shape assertions.
	if gotPath != "/v1/messages" {
		t.Errorf("path = %q, want /v1/messages", gotPath)
	}
	if got := gotHeaders.Get("x-api-key"); got != "test-api-key" {
		t.Errorf("x-api-key header = %q, want %q", got, "test-api-key")
	}
	if got := gotHeaders.Get("anthropic-version"); got != anthropicAPIVersion {
		t.Errorf("anthropic-version header = %q, want %q", got, anthropicAPIVersion)
	}
	if got := gotHeaders.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type header = %q, want application/json", got)
	}

	if gotBody["model"] != "claude-sonnet-5" {
		t.Errorf("body model = %v, want claude-sonnet-5", gotBody["model"])
	}
	if maxTokens, ok := gotBody["max_tokens"].(float64); !ok || maxTokens != 1024 {
		t.Errorf("body max_tokens = %v, want 1024", gotBody["max_tokens"])
	}
	messages, ok := gotBody["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("body messages = %v, want a single-element array", gotBody["messages"])
	}
	msg, ok := messages[0].(map[string]any)
	if !ok || msg["role"] != "user" {
		t.Fatalf("messages[0] = %v, want role=user", messages[0])
	}
	content, _ := msg["content"].(string)
	if !strings.Contains(content, "does this diff do the thing?") {
		t.Errorf("message content = %q, want it to contain the prompt", content)
	}

	// Verdict parsing assertions.
	if !v.AccomplishesIntent {
		t.Error("AccomplishesIntent = false, want true")
	}
	if v.OnlyStatedIntent {
		t.Error("OnlyStatedIntent = true, want false")
	}
	if len(v.Surprises) != 1 || v.Surprises[0] != "renamed unrelated function" {
		t.Errorf("Surprises = %v, want [\"renamed unrelated function\"]", v.Surprises)
	}
	if v.Raw == "" {
		t.Error("Raw = \"\", want the raw text block")
	}
}

func TestAnthropicProvider_Verify_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"boom"}}`))
	}))
	defer srv.Close()

	p := NewAnthropicProvider("claude-sonnet-5", "test-api-key")
	p.BaseURL = srv.URL + "/v1/messages"

	_, err := p.Verify(context.Background(), "prompt")
	if err == nil {
		t.Fatal("Verify returned nil error, want non-nil for a 500 response")
	}
}

func TestAnthropicProvider_Verify_NoTextBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"content": []map[string]any{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewAnthropicProvider("claude-sonnet-5", "test-api-key")
	p.BaseURL = srv.URL + "/v1/messages"

	_, err := p.Verify(context.Background(), "prompt")
	if err == nil {
		t.Fatal("Verify returned nil error, want non-nil when no text content block is present")
	}
}
