package llmscrape

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedRequest struct {
	Auth string
	Body map[string]any
}

// fakeLLM answers every chat completion with content (or reasoning only when content is empty).
func fakeLLM(t *testing.T, status int, content, reasoning string, got *capturedRequest) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got != nil {
			got.Auth = r.Header.Get("Authorization")
			json.NewDecoder(r.Body).Decode(&got.Body)
		}
		w.WriteHeader(status)
		msg := map[string]any{"content": nil, "reasoning_content": reasoning}
		if content != "" {
			msg["content"] = content
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": msg, "finish_reason": "stop"}},
		})
	}))
}

var tinySchema = map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}}

func TestCompleteSendsJSONSchema(t *testing.T) {
	var got capturedRequest
	srv := fakeLLM(t, 200, `{"ok":true}`, "", &got)
	defer srv.Close()

	c, _ := NewClient(LLMConfig{BaseURL: srv.URL + "/v1/", Model: "m", APIKey: "k", DisableThinking: true})
	out, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, "t", tinySchema, 10)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"ok":true}` {
		t.Errorf("out = %s", out)
	}
	if got.Auth != "Bearer k" {
		t.Errorf("auth = %q", got.Auth)
	}
	rf := got.Body["response_format"].(map[string]any)
	if rf["type"] != "json_schema" || rf["json_schema"].(map[string]any)["strict"] != true {
		t.Errorf("response_format = %v", rf)
	}
	if kw, ok := got.Body["chat_template_kwargs"].(map[string]any); !ok || kw["thinking"] != false {
		t.Errorf("thinking should be disabled, body = %v", got.Body)
	}
	if got.Body["temperature"] != float64(0) {
		t.Errorf("temperature = %v", got.Body["temperature"])
	}
}

func TestCompleteOmitsOptionalFields(t *testing.T) {
	var got capturedRequest
	srv := fakeLLM(t, 200, `{"ok":true}`, "", &got)
	defer srv.Close()

	c, _ := NewClient(LLMConfig{BaseURL: srv.URL + "/v1", Model: "m"})
	if _, err := c.Complete(context.Background(), nil, "t", tinySchema, 10); err != nil {
		t.Fatal(err)
	}
	if got.Auth != "" {
		t.Errorf("no API key configured, but sent %q", got.Auth)
	}
	if _, ok := got.Body["chat_template_kwargs"]; ok {
		t.Error("chat_template_kwargs sent although thinking was not disabled; strict servers reject unknown fields")
	}
}

func TestCompleteJSONObjectModePutsSchemaInPrompt(t *testing.T) {
	var got capturedRequest
	srv := fakeLLM(t, 200, "```json\n{\"ok\":true}\n```", "", &got)
	defer srv.Close()

	c, _ := NewClient(LLMConfig{BaseURL: srv.URL + "/v1", Model: "m", StructuredOutput: "json_object"})
	out, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}}, "t", tinySchema, 10)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"ok":true}` {
		t.Errorf("code fence not stripped: %s", out)
	}
	if got.Body["response_format"].(map[string]any)["type"] != "json_object" {
		t.Errorf("response_format = %v", got.Body["response_format"])
	}
	msgs := got.Body["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)["content"].(string)
	if !strings.Contains(last, `"ok"`) {
		t.Errorf("schema not included in prompt: %q", last)
	}
}

func TestCompleteExplainsReasoningExhaustion(t *testing.T) {
	srv := fakeLLM(t, 200, "", "thinking about it...", nil)
	defer srv.Close()
	c, _ := NewClient(LLMConfig{BaseURL: srv.URL + "/v1", Model: "m"})
	_, err := c.Complete(context.Background(), nil, "t", tinySchema, 10)
	if err == nil || !strings.Contains(err.Error(), "disable thinking") {
		t.Errorf("expected a hint about thinking mode, got %v", err)
	}
}

func TestCompleteErrors(t *testing.T) {
	srv := fakeLLM(t, 500, "", "", nil)
	defer srv.Close()
	c, _ := NewClient(LLMConfig{BaseURL: srv.URL + "/v1", Model: "m"})
	if _, err := c.Complete(context.Background(), nil, "t", tinySchema, 10); err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 error, got %v", err)
	}

	bad := fakeLLM(t, 200, "not json", "", nil)
	defer bad.Close()
	c2, _ := NewClient(LLMConfig{BaseURL: bad.URL + "/v1", Model: "m"})
	if _, err := c2.Complete(context.Background(), nil, "t", tinySchema, 10); err == nil {
		t.Error("expected invalid JSON to be rejected")
	}

	if _, err := NewClient(LLMConfig{Model: "m"}); err == nil {
		t.Error("expected missing base URL to be rejected")
	}
}
