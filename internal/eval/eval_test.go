package eval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ysksm/cc-otel/internal/llm"
	"github.com/ysksm/cc-otel/internal/model"
)

func TestParseVerdict(t *testing.T) {
	cases := []struct {
		in        string
		wantVal   float64
		wantPass  bool
	}{
		{`{"value":0.8,"passed":true,"reasoning":"good"}`, 0.8, true},
		{"```json\n{\"value\": 0.2, \"passed\": false, \"reasoning\": \"bad\"}\n```", 0.2, false},
		{`Here is my verdict: {"value":"0.9","passed":"yes","reasoning":"ok"} done`, 0.9, true},
		{`{"value":0.4}`, 0.4, false}, // passed derived from value<0.5
	}
	for i, c := range cases {
		v, err := ParseVerdict(c.in)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if v.Value != c.wantVal || v.Passed != c.wantPass {
			t.Fatalf("case %d: got value=%v passed=%v want %v/%v", i, v.Value, v.Passed, c.wantVal, c.wantPass)
		}
	}
}

func TestBuildConversation(t *testing.T) {
	spans := []model.Span{{
		StartTimeNs:    100,
		Operation:      "chat",
		InputMessages:  `[{"role":"user","content":"What is 2+2?"}]`,
		OutputMessages: `[{"role":"assistant","content":"4"}]`,
	}}
	conv := BuildConversation(spans)
	if conv == "" || !contains(conv, "What is 2+2?") || !contains(conv, "4") {
		t.Fatalf("conversation missing content: %q", conv)
	}
}

func TestRunWithMockLLM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		resp := map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"role": "assistant",
				"content": `{"value":0.75,"passed":true,"reasoning":"correct answer"}`}}},
			"usage": map[string]any{"prompt_tokens": 12, "completion_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := llm.New(llm.Config{Provider: "openai", BaseURL: srv.URL, Model: "mock"})
	e := model.Evaluation{Name: "correctness", Prompt: "Is the answer correct?"}
	spans := []model.Span{{
		StartTimeNs:    1,
		Operation:      "chat",
		InputMessages:  `[{"role":"user","content":"2+2?"}]`,
		OutputMessages: `[{"role":"assistant","content":"4"}]`,
	}}
	v, usage, err := Run(context.Background(), client, e, spans)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if v.Value != 0.75 || !v.Passed || v.Reasoning != "correct answer" {
		t.Fatalf("verdict wrong: %+v", v)
	}
	if usage.InputTokens != 12 || usage.OutputTokens != 5 {
		t.Fatalf("usage wrong: %+v", usage)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
