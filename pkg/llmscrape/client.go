// Package llmscrape turns an arbitrary scene web page into an XBVR scene using a language model
// behind any OpenAI-compatible chat completions API, and finds candidate pages for unmatched files
// with a web search.
package llmscrape

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLMConfig describes an OpenAI-compatible chat completions endpoint.
type LLMConfig struct {
	BaseURL          string // e.g. http://localhost:8000/v1
	Model            string
	APIKey           string // optional
	DisableThinking  bool
	StructuredOutput string // "json_schema" (default) or "json_object"
	Timeout          time.Duration
}

// Message is one chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Client calls a chat completions endpoint and returns schema-shaped JSON.
type Client struct {
	cfg  LLMConfig
	http *http.Client
}

// NewClient validates the configuration and returns a client.
func NewClient(cfg LLMConfig) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("LLM base URL is not configured")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("LLM model is not configured")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 180 * time.Second
	}
	if cfg.StructuredOutput == "" {
		cfg.StructuredOutput = "json_schema"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}, nil
}

// Model returns the configured model name.
func (c *Client) Model() string { return c.cfg.Model }

type chatRequest struct {
	Model              string          `json:"model"`
	Messages           []Message       `json:"messages"`
	Temperature        float64         `json:"temperature"`
	MaxTokens          int             `json:"max_tokens,omitempty"`
	ResponseFormat     json.RawMessage `json:"response_format,omitempty"`
	ChatTemplateKwargs map[string]any  `json:"chat_template_kwargs,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content          *string `json:"content"`
			ReasoningContent *string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Complete sends the messages and returns the model's JSON answer, constrained to schema where
// the server supports it. The result is only guaranteed to be valid JSON; callers must still
// validate the values.
func (c *Client) Complete(ctx context.Context, messages []Message, schemaName string, schema map[string]any, maxTokens int) (json.RawMessage, error) {
	req := chatRequest{
		Model:       c.cfg.Model,
		Messages:    messages,
		Temperature: 0,
		MaxTokens:   maxTokens,
	}

	switch c.cfg.StructuredOutput {
	case "json_object":
		// No decoding constraint beyond valid JSON, so the schema has to travel in the prompt.
		schemaText, _ := json.Marshal(schema)
		req.Messages = append([]Message{}, messages...)
		req.Messages = append(req.Messages, Message{
			Role:    "system",
			Content: "Answer with a single JSON object that conforms to this JSON schema and nothing else:\n" + string(schemaText),
		})
		req.ResponseFormat = json.RawMessage(`{"type":"json_object"}`)
	default:
		rf, err := json.Marshal(map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   schemaName,
				"strict": true,
				"schema": schema,
			},
		})
		if err != nil {
			return nil, err
		}
		req.ResponseFormat = rf
	}

	if c.cfg.DisableThinking {
		// Both spellings are in use across chat templates; templates ignore keys they don't use.
		req.ChatTemplateKwargs = map[string]any{"thinking": false, "enable_thinking": false}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("reading LLM response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM returned HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 300))
	}

	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return nil, fmt.Errorf("LLM response is not a chat completion: %w", err)
	}
	if len(cr.Choices) == 0 {
		return nil, errors.New("LLM returned no choices")
	}
	choice := cr.Choices[0]
	if choice.Message.Content == nil || strings.TrimSpace(*choice.Message.Content) == "" {
		if choice.Message.ReasoningContent != nil && *choice.Message.ReasoningContent != "" {
			return nil, fmt.Errorf("LLM spent its whole budget reasoning (finish_reason=%s); enable \"disable thinking\" or raise the token limit", choice.FinishReason)
		}
		return nil, fmt.Errorf("LLM returned an empty answer (finish_reason=%s)", choice.FinishReason)
	}

	content := stripCodeFence(*choice.Message.Content)
	if !json.Valid([]byte(content)) {
		return nil, fmt.Errorf("LLM answer is not valid JSON (finish_reason=%s): %s", choice.FinishReason, truncate(content, 200))
	}
	return json.RawMessage(content), nil
}

// stripCodeFence removes a ```json ... ``` wrapper, which some servers add in json_object mode.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	if i := strings.Index(s, "\n"); i >= 0 {
		s = s[i+1:]
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
}

// truncate shortens s to at most n characters without splitting a multi-byte character.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
