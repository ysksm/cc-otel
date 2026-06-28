package otlp

import (
	"encoding/json"
	"strings"

	"github.com/ysksm/cc-otel/internal/model"
	"github.com/ysksm/cc-otel/internal/pricing"
)

// IngestContext carries the tenancy resolved from the API key / project lookup.
type IngestContext struct {
	WorkspaceID string
	APIKeyID    string
	// ResolveProject maps a project slug (from span/resource attrs) to a project
	// id. An empty slug returns the default project id. It returns ok=false when
	// the slug cannot be resolved.
	ResolveProject func(slug string) (projectID string, ok bool)
}

// ToSpans converts decoded OTLP resource spans into enriched model.Span rows.
// Spans whose project cannot be resolved are skipped and counted as rejected.
func ToSpans(resourceSpans []ResourceSpans, ctx IngestContext) (spans []model.Span, rejected int) {
	for _, rs := range resourceSpans {
		serviceName := rs.Resource.Str("service.name")
		for _, sc := range rs.Scopes {
			for _, sp := range sc.Spans {
				slug := firstNonEmpty(sp.Attrs.Str("latitude.project"), rs.Resource.Str("latitude.project"))
				projectID, ok := ctx.ResolveProject(slug)
				if !ok {
					rejected++
					continue
				}
				m := buildSpan(sp, rs.Resource, sc.Name, sc.Version, serviceName)
				m.WorkspaceID = ctx.WorkspaceID
				m.ProjectID = projectID
				m.APIKeyID = ctx.APIKeyID
				spans = append(spans, m)
			}
		}
	}
	return spans, rejected
}

var providerAliases = map[string]string{
	"bedrock": "amazon-bedrock", "aws_bedrock": "amazon-bedrock", "aws.bedrock": "amazon-bedrock",
	"gemini": "google", "vertexai": "google-vertex", "vertex_ai": "google-vertex", "gcp.vertex_ai": "google-vertex",
}

func buildSpan(sp Span, resource Attrs, scopeName, scopeVersion, serviceName string) model.Span {
	a := sp.Attrs
	m := model.Span{
		SessionID:     resolveSessionID(a),
		TraceID:       sp.TraceID,
		SpanID:        sp.SpanID,
		ParentSpanID:  sp.ParentSpanID,
		StartTimeNs:   int64(sp.StartNs),
		EndTimeNs:     int64(sp.EndNs),
		Name:          sp.Name,
		ServiceName:   serviceName,
		Kind:          int8(sp.Kind),
		StatusCode:    int16(sp.StatusCode),
		StatusMessage: sp.StatusMessage,
		ScopeName:     scopeName,
		ScopeVersion:  scopeVersion,
		Provider:      resolveProvider(a, scopeName, sp.Name),
		Model:         a.Str("gen_ai.request.model", "llm.model_name", "ai.model.id", "embedding.model_name", "model"),
		ResponseModel: a.Str("gen_ai.response.model", "ai.response.model"),
		ResponseID:    a.Str("gen_ai.response.id", "ai.response.id"),
		UserID:        a.Str("user.id", "enduser.id", "gen_ai.request.user", "langfuse.user.id"),
		UserEmail:     a.Str("user.email", "enduser.email"),
		Attributes:    toJSON(map[string]any(a)),
		Resource:      toJSON(map[string]any(resource)),
		FinishReasons: resolveFinishReasons(a),
		Tags:          resolveTags(a),
	}
	if m.StatusCode == 2 {
		m.ErrorType = a.Str("error.type")
	}
	m.Operation = resolveOperation(a, scopeName, sp.Name, m.Model)

	// Tokens (normalized additive) + cost.
	m.TokensInput, m.TokensOutput, m.TokensCacheRead, m.TokensCacheCreate, m.TokensReasoning = resolveTokens(a)
	m.CostInputMicrocents, m.CostOutputMicrocents, m.CostTotalMicrocents, m.CostIsEstimated = resolveCost(a, m)

	// Streaming / TTFT.
	m.TimeToFirstTokenNs = resolveTTFT(a)
	if b, ok := a.Bool("gen_ai.request.stream"); ok {
		m.IsStreaming = b
	} else if m.TimeToFirstTokenNs > 0 {
		m.IsStreaming = true
	}

	// Messages.
	m.InputMessages = resolveMessages(a, "gen_ai.input.messages", "gen_ai.prompt", "ai.prompt.messages", "ai.prompt", "input.value", "user_prompt")
	m.OutputMessages = resolveMessages(a, "gen_ai.output.messages", "gen_ai.completion", "ai.response.text", "output.value")
	m.SystemInstructions = resolveMessages(a, "gen_ai.system_instructions", "ai.prompt.system")

	// Tool execution.
	if m.Operation == "execute_tool" {
		m.ToolCallID = a.Str("gen_ai.tool.call.id", "ai.toolCall.id", "tool_call.id")
		m.ToolName = a.Str("gen_ai.tool.name", "ai.toolCall.name", "tool.name", "traceloop.entity.name")
		m.ToolInput = anyToJSON(a, "gen_ai.tool.call.arguments", "ai.toolCall.args", "input.value")
		m.ToolOutput = anyToJSON(a, "gen_ai.tool.call.result", "ai.toolCall.result", "output.value")
	}
	return m
}

func resolveProvider(a Attrs, scopeName, spanName string) string {
	p := a.Str("gen_ai.provider.name", "gen_ai.system", "llm.system", "llm.provider", "metadata.ls_provider", "ai.model.provider")
	if p == "" {
		if strings.HasPrefix(scopeName, "claude_code") || strings.HasPrefix(spanName, "claude_code") {
			return "anthropic"
		}
		return ""
	}
	p = strings.ToLower(p)
	// Vercel ai.model.provider looks like "anthropic.messages"; keep the head.
	if i := strings.IndexByte(p, '.'); i > 0 {
		if alias, ok := providerAliases[p]; ok {
			return alias
		}
		p = p[:i]
	}
	if alias, ok := providerAliases[p]; ok {
		return alias
	}
	return p
}

func resolveOperation(a Attrs, scopeName, spanName, modelName string) string {
	if op := a.Str("gen_ai.operation.name"); op != "" {
		op = strings.ToLower(op)
		if op == "rerank" {
			return "reranker"
		}
		return op
	}
	switch strings.ToLower(a.Str("openinference.span.kind")) {
	case "llm":
		return "chat"
	case "embedding":
		return "embeddings"
	case "tool":
		return "execute_tool"
	case "agent":
		return "invoke_agent"
	case "chain":
		return "chain"
	case "retriever":
		return "retrieval"
	case "reranker":
		return "reranker"
	}
	switch a.Str("ai.operationId") {
	case "ai.generateText", "ai.streamText", "ai.generateText.doGenerate", "ai.streamText.doStream":
		return "chat"
	case "ai.embed", "ai.embedMany":
		return "embeddings"
	case "ai.toolCall":
		return "execute_tool"
	}
	switch a.Str("span.type") {
	case "llm_request":
		return "chat"
	case "tool_execution":
		return "execute_tool"
	case "interaction":
		return "chain"
	}
	if a.Has("gen_ai.tool.name", "ai.toolCall.name", "tool.name") {
		return "execute_tool"
	}
	if modelName != "" {
		return "chat"
	}
	return "unspecified"
}

// inclusiveInputKeys are conventions where input token counts include cache tokens.
var inclusiveInputKeys = map[string]bool{
	"gen_ai.usage.input_tokens":  true,
	"gen_ai.usage.prompt_tokens": true,
	"ai.usage.promptTokens":      true,
	"ai.usage.inputTokens":       true,
	"llm.token_count.prompt":     true,
}

func resolveTokens(a Attrs) (input, output, cacheRead, cacheCreate, reasoning int64) {
	rawInput, inputKey := firstI64(a, "gen_ai.usage.input_tokens", "gen_ai.usage.prompt_tokens",
		"llm.token_count.prompt", "ai.usage.promptTokens", "ai.usage.inputTokens", "input_tokens")
	output, _ = firstI64(a, "gen_ai.usage.output_tokens", "gen_ai.usage.completion_tokens",
		"llm.token_count.completion", "ai.usage.completionTokens", "ai.usage.outputTokens", "output_tokens")
	cacheRead, _ = firstI64(a, "gen_ai.usage.cache_read.input_tokens", "gen_ai.usage.cache_read_input_tokens",
		"ai.usage.cachedInputTokens", "llm.token_count.prompt_details.cache_read", "cache_read_tokens")
	cacheCreate, _ = firstI64(a, "gen_ai.usage.cache_creation.input_tokens", "gen_ai.usage.cache_creation_input_tokens",
		"llm.token_count.prompt_details.cache_write", "cache_creation_tokens")
	reasoning, _ = firstI64(a, "gen_ai.usage.reasoning_tokens", "llm.token_count.completion_details.reasoning")

	input = rawInput
	// Normalize to additive: if input was reported inclusively, subtract cache.
	if inclusiveInputKeys[inputKey] {
		input = input - cacheRead - cacheCreate
		if input < 0 {
			input = 0
		}
	}
	return input, output, cacheRead, cacheCreate, reasoning
}

func resolveCost(a Attrs, m model.Span) (in, out, total int64, estimated bool) {
	inUSD, hasIn := a.F64("gen_ai.usage.input_cost", "llm.cost.prompt")
	outUSD, hasOut := a.F64("gen_ai.usage.output_cost", "llm.cost.completion")
	totalUSD, hasTotal := a.F64("gen_ai.usage.total_cost", "gen_ai.usage.cost", "llm.cost.total")
	if hasIn || hasOut {
		in = pricing.USDToMicrocents(inUSD)
		out = pricing.USDToMicrocents(outUSD)
		total = in + out
		return in, out, total, false
	}
	if hasTotal {
		return 0, 0, pricing.USDToMicrocents(totalUSD), false
	}
	if ein, eout, etotal, ok := pricing.Estimate(m.Model, m.TokensInput, m.TokensOutput, m.TokensCacheRead, m.TokensCacheCreate, m.TokensReasoning); ok {
		return ein, eout, etotal, true
	}
	return 0, 0, 0, false
}

func resolveTTFT(a Attrs) int64 {
	if v, ok := a.I64("gen_ai.server.time_to_first_token", "llm.latency.time_to_first_token"); ok {
		return v
	}
	if ms, ok := a.F64("ttft_ms"); ok {
		return int64(ms * 1_000_000)
	}
	return 0
}

func resolveSessionID(a Attrs) string {
	return a.Str("session.id", "gen_ai.session.id", "langfuse.session.id",
		"traceloop.association.properties.session_id", "langsmith.trace.session_id",
		"session_id", "gen_ai.conversation.id")
}

func resolveFinishReasons(a Attrs) string {
	if v, ok := a.Raw("gen_ai.response.finish_reasons"); ok {
		if arr, isArr := v.([]any); isArr {
			return toJSON(arr)
		}
		return toJSON([]any{v})
	}
	if s := a.Str("ai.response.finishReason", "llm.finish_reason"); s != "" {
		s = strings.ReplaceAll(strings.ReplaceAll(s, "tool-calls", "tool_calls"), "content-filter", "content_filter")
		return toJSON([]any{s})
	}
	return "[]"
}

func resolveTags(a Attrs) string {
	if v, ok := a.Raw("langfuse.trace.tags", "braintrust.tags", "tag.tags"); ok {
		if arr, isArr := v.([]any); isArr {
			return toJSON(arr)
		}
	}
	if s := a.Str("latitude.tags"); s != "" {
		// latitude.tags is a JSON string
		var arr []any
		if json.Unmarshal([]byte(s), &arr) == nil {
			return toJSON(arr)
		}
	}
	return "[]"
}

func resolveMessages(a Attrs, keys ...string) string {
	for _, k := range keys {
		if v, ok := a.Raw(k); ok && v != nil {
			switch t := v.(type) {
			case string:
				if t == "" {
					continue
				}
				// If it already parses as JSON, keep it; else wrap as a text message.
				var probe any
				if json.Unmarshal([]byte(t), &probe) == nil {
					return t
				}
				return toJSON([]any{map[string]any{"role": "user", "content": t}})
			default:
				return toJSON(t)
			}
		}
	}
	return "[]"
}

func anyToJSON(a Attrs, keys ...string) string {
	if v, ok := a.Raw(keys...); ok && v != nil {
		if s, isStr := v.(string); isStr {
			return s
		}
		return toJSON(v)
	}
	return ""
}

func firstI64(a Attrs, keys ...string) (int64, string) {
	for _, k := range keys {
		if v, ok := a.I64(k); ok {
			return v, k
		}
	}
	return 0, ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}
