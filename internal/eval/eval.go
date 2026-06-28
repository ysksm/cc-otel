// Package eval runs LLM-as-judge evaluations against a trace's conversation.
package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ysksm/cc-otel/internal/llm"
	"github.com/ysksm/cc-otel/internal/model"
)

// Verdict is the parsed judge result.
type Verdict struct {
	Value     float64 `json:"value"`
	Passed    bool    `json:"passed"`
	Reasoning string  `json:"reasoning"`
}

// Run evaluates a trace's conversation with the given evaluator and returns the
// verdict plus token usage. An error is returned if the LLM call or parsing fails.
func Run(ctx context.Context, client *llm.Client, e model.Evaluation, spans []model.Span) (Verdict, llm.Usage, error) {
	conv := BuildConversation(spans)
	if strings.TrimSpace(conv) == "" {
		return Verdict{}, llm.Usage{}, fmt.Errorf("trace has no LLM conversation to evaluate")
	}
	system, msgs := BuildJudgePrompt(e, conv)
	text, usage, err := client.Chat(ctx, system, msgs)
	if err != nil {
		return Verdict{}, usage, err
	}
	v, err := ParseVerdict(text)
	if err != nil {
		return Verdict{}, usage, fmt.Errorf("could not parse judge output: %w (raw: %s)", err, truncate(text, 200))
	}
	return v, usage, nil
}

// BuildJudgePrompt builds the system + user messages for the judge.
func BuildJudgePrompt(e model.Evaluation, conversation string) (string, []llm.Message) {
	instructions := strings.TrimSpace(e.Prompt)
	if instructions == "" {
		instructions = "Evaluate the quality and correctness of the assistant's response."
	}
	system := "You are a strict, fair evaluator of LLM/agent conversations. " + instructions +
		"\n\nRespond with ONLY a single JSON object and nothing else, in this exact shape:\n" +
		`{"value": <number between 0 and 1>, "passed": <true|false>, "reasoning": "<one or two short sentences>"}`
	user := "Evaluate the following conversation:\n\n" + conversation
	return system, []llm.Message{{Role: "user", Content: user}}
}

// ParseVerdict extracts the verdict JSON from the model output (tolerant of code
// fences and surrounding prose).
func ParseVerdict(text string) (Verdict, error) {
	raw := extractJSONObject(text)
	if raw == "" {
		return Verdict{}, fmt.Errorf("no JSON object found")
	}
	// Decode leniently: passed/value may arrive as strings or numbers.
	var loose struct {
		Value     json.RawMessage `json:"value"`
		Passed    json.RawMessage `json:"passed"`
		Reasoning string          `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(raw), &loose); err != nil {
		return Verdict{}, err
	}
	v := Verdict{Reasoning: strings.TrimSpace(loose.Reasoning)}
	v.Value = clamp01(parseNumber(loose.Value))
	if p, ok := parseBool(loose.Passed); ok {
		v.Passed = p
	} else {
		v.Passed = v.Value >= 0.5
	}
	return v, nil
}

// SpanText extracts a plain-text representation of a single span's content
// (name + input/output message text), used for embedding/indexing.
func SpanText(s model.Span) string {
	var parts []string
	if s.Name != "" {
		parts = append(parts, s.Name)
	}
	for _, m := range parseMsgs(s.InputMessages) {
		if m.Content != "" {
			parts = append(parts, m.Content)
		}
	}
	for _, m := range parseMsgs(s.OutputMessages) {
		if m.Content != "" {
			parts = append(parts, m.Content)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// BuildConversation formats the trace's most relevant LLM conversation as text.
func BuildConversation(spans []model.Span) string {
	span := pickConversationSpan(spans)
	if span == nil {
		return ""
	}
	var sb strings.Builder
	if sys := joinMessages(span.SystemInstructions); sys != "" {
		sb.WriteString("System:\n")
		sb.WriteString(sys)
		sb.WriteString("\n\n")
	}
	for _, m := range parseMsgs(span.InputMessages) {
		writeMsg(&sb, m)
	}
	for _, m := range parseMsgs(span.OutputMessages) {
		writeMsg(&sb, m)
	}
	return strings.TrimSpace(sb.String())
}

func pickConversationSpan(spans []model.Span) *model.Span {
	sorted := append([]model.Span(nil), spans...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartTimeNs < sorted[j].StartTimeNs })
	for i := len(sorted) - 1; i >= 0; i-- {
		if len(parseMsgs(sorted[i].OutputMessages)) > 0 {
			return &sorted[i]
		}
	}
	for i := len(sorted) - 1; i >= 0; i-- {
		if len(parseMsgs(sorted[i].InputMessages)) > 0 || sorted[i].Operation == "chat" {
			return &sorted[i]
		}
	}
	return nil
}

type msg struct {
	Role    string
	Content string
}

func parseMsgs(jsonStr string) []msg {
	if strings.TrimSpace(jsonStr) == "" {
		return nil
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &arr); err != nil {
		return nil
	}
	var out []msg
	for _, m := range arr {
		role, _ := m["role"].(string)
		if role == "" {
			role = "unknown"
		}
		out = append(out, msg{Role: role, Content: contentToText(m["content"])})
	}
	return out
}

func joinMessages(jsonStr string) string {
	var parts []string
	for _, m := range parseMsgs(jsonStr) {
		parts = append(parts, m.Content)
	}
	if len(parts) == 0 {
		// systemInstructions may be a plain JSON string or array of strings.
		var s string
		if json.Unmarshal([]byte(jsonStr), &s) == nil {
			return s
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func writeMsg(sb *strings.Builder, m msg) {
	if strings.TrimSpace(m.Content) == "" {
		return
	}
	sb.WriteString(titleCase(m.Role))
	sb.WriteString(":\n")
	sb.WriteString(m.Content)
	sb.WriteString("\n\n")
}

func contentToText(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []any:
		var parts []string
		for _, p := range t {
			if pm, ok := p.(map[string]any); ok {
				if s, ok := pm["text"].(string); ok {
					parts = append(parts, s)
					continue
				}
				b, _ := json.Marshal(pm)
				parts = append(parts, string(b))
			} else if s, ok := p.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n")
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// ---- small helpers ---------------------------------------------------------

func extractJSONObject(text string) string {
	// strip code fences
	text = strings.ReplaceAll(text, "```json", "```")
	if i := strings.Index(text, "```"); i >= 0 {
		if j := strings.Index(text[i+3:], "```"); j >= 0 {
			text = text[i+3 : i+3+j]
		}
	}
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start < 0 || end < 0 || end < start {
		return ""
	}
	return text[start : end+1]
}

func parseNumber(raw json.RawMessage) float64 {
	if len(raw) == 0 {
		return 0
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return f
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		var f2 float64
		_, _ = fmt.Sscanf(strings.TrimSpace(s), "%g", &f2)
		return f2
	}
	return 0
}

func parseBool(raw json.RawMessage) (bool, bool) {
	if len(raw) == 0 {
		return false, false
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b, true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "yes", "pass", "passed", "1":
			return true, true
		case "false", "no", "fail", "failed", "0":
			return false, true
		}
	}
	return false, false
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
