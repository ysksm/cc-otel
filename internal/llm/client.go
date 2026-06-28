// Package llm is a minimal chat client for LLM-as-judge evaluations. It supports
// any OpenAI-compatible Chat Completions endpoint (OpenAI, Ollama, LM Studio,
// vLLM, OpenRouter, ...) and the Anthropic Messages API. This keeps evaluations
// runnable fully locally (point CCOTEL_LLM_BASE_URL at a local Ollama) or against
// a hosted provider.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Message is a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage reports token counts when the provider returns them.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
}

// Config selects the provider and endpoint.
type Config struct {
	Provider string // "openai" (default, openai-compatible) or "anthropic"
	BaseURL  string
	APIKey   string
	Model    string
}

// ConfigFromEnv reads default LLM settings from the environment.
func ConfigFromEnv() Config {
	return Config{
		Provider: getenv("CCOTEL_LLM_PROVIDER", "openai"),
		BaseURL:  os.Getenv("CCOTEL_LLM_BASE_URL"),
		APIKey:   os.Getenv("CCOTEL_LLM_API_KEY"),
		Model:    os.Getenv("CCOTEL_LLM_MODEL"),
	}
}

// EmbedConfigFromEnv reads embedding settings (CCOTEL_EMBED_*). Embeddings are
// considered configured only when a model is set.
func EmbedConfigFromEnv() Config {
	return Config{
		Provider: getenv("CCOTEL_EMBED_PROVIDER", "openai"),
		BaseURL:  os.Getenv("CCOTEL_EMBED_BASE_URL"),
		APIKey:   os.Getenv("CCOTEL_EMBED_API_KEY"),
		Model:    os.Getenv("CCOTEL_EMBED_MODEL"),
	}
}

// Embed returns embedding vectors for the given inputs via an OpenAI-compatible
// /embeddings endpoint (works with OpenAI, Ollama, LM Studio, ...).
func (c *Client) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if c.cfg.Model == "" {
		return nil, fmt.Errorf("no embedding model configured (set CCOTEL_EMBED_MODEL)")
	}
	if len(inputs) == 0 {
		return nil, nil
	}
	base := c.cfg.BaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	reqBody := map[string]any{"model": c.cfg.Model, "input": inputs}
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	headers := http.Header{}
	if c.cfg.APIKey != "" {
		headers.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	if err := c.postJSON(ctx, strings.TrimRight(base, "/")+"/embeddings", headers, reqBody, &out); err != nil {
		return nil, err
	}
	vecs := make([][]float32, 0, len(out.Data))
	for _, d := range out.Data {
		vecs = append(vecs, d.Embedding)
	}
	if len(vecs) != len(inputs) {
		return nil, fmt.Errorf("embedding count mismatch: got %d for %d inputs", len(vecs), len(inputs))
	}
	return vecs, nil
}

// Merge overlays non-empty fields of o onto c (used for per-evaluation overrides).
func (c Config) Merge(o Config) Config {
	if o.Provider != "" {
		c.Provider = o.Provider
	}
	if o.BaseURL != "" {
		c.BaseURL = o.BaseURL
	}
	if o.APIKey != "" {
		c.APIKey = o.APIKey
	}
	if o.Model != "" {
		c.Model = o.Model
	}
	return c
}

// Client calls a chat LLM.
type Client struct {
	cfg  Config
	http *http.Client
}

// New builds a client.
func New(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 120 * time.Second}}
}

// Chat sends a system prompt + messages and returns the assistant text + usage.
func (c *Client) Chat(ctx context.Context, system string, msgs []Message) (string, Usage, error) {
	if c.cfg.Model == "" {
		return "", Usage{}, fmt.Errorf("no LLM model configured (set CCOTEL_LLM_MODEL or the evaluation's model)")
	}
	switch strings.ToLower(c.cfg.Provider) {
	case "anthropic":
		return c.anthropic(ctx, system, msgs)
	default:
		return c.openai(ctx, system, msgs)
	}
}

func (c *Client) openai(ctx context.Context, system string, msgs []Message) (string, Usage, error) {
	base := c.cfg.BaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	all := make([]Message, 0, len(msgs)+1)
	if system != "" {
		all = append(all, Message{Role: "system", Content: system})
	}
	all = append(all, msgs...)
	reqBody := map[string]any{"model": c.cfg.Model, "messages": all, "temperature": 0}
	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	headers := http.Header{}
	if c.cfg.APIKey != "" {
		headers.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	if err := c.postJSON(ctx, strings.TrimRight(base, "/")+"/chat/completions", headers, reqBody, &out); err != nil {
		return "", Usage{}, err
	}
	if len(out.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("LLM returned no choices")
	}
	return out.Choices[0].Message.Content, Usage{out.Usage.PromptTokens, out.Usage.CompletionTokens}, nil
}

func (c *Client) anthropic(ctx context.Context, system string, msgs []Message) (string, Usage, error) {
	base := c.cfg.BaseURL
	if base == "" {
		base = "https://api.anthropic.com"
	}
	reqBody := map[string]any{"model": c.cfg.Model, "max_tokens": 1024, "messages": msgs}
	if system != "" {
		reqBody["system"] = system
	}
	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	headers := http.Header{}
	headers.Set("x-api-key", c.cfg.APIKey)
	headers.Set("anthropic-version", "2023-06-01")
	if err := c.postJSON(ctx, strings.TrimRight(base, "/")+"/v1/messages", headers, reqBody, &out); err != nil {
		return "", Usage{}, err
	}
	var sb strings.Builder
	for _, p := range out.Content {
		sb.WriteString(p.Text)
	}
	return sb.String(), Usage{out.Usage.InputTokens, out.Usage.OutputTokens}, nil
}

func (c *Client) postJSON(ctx context.Context, url string, headers http.Header, body, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		snippet := string(data)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return fmt.Errorf("LLM endpoint returned %s: %s", resp.Status, snippet)
	}
	return json.Unmarshal(data, out)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
