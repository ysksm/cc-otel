package otlp

import "testing"

const sampleJSON = `{
  "resourceSpans": [{
    "resource": {"attributes":[{"key":"service.name","value":{"stringValue":"svc"}}]},
    "scopeSpans": [{
      "scope": {"name":"openai","version":"1"},
      "spans": [{
        "traceId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1",
        "spanId":"bbbbbbbbbbbbbbb1",
        "name":"chat",
        "kind":3,
        "startTimeUnixNano":"1000",
        "endTimeUnixNano":"2000",
        "status":{"code":1},
        "attributes":[
          {"key":"gen_ai.system","value":{"stringValue":"openai"}},
          {"key":"gen_ai.request.model","value":{"stringValue":"gpt-4o"}},
          {"key":"gen_ai.operation.name","value":{"stringValue":"chat"}},
          {"key":"session.id","value":{"stringValue":"s1"}},
          {"key":"gen_ai.usage.input_tokens","value":{"intValue":"1000"}},
          {"key":"gen_ai.usage.output_tokens","value":{"intValue":"500"}}
        ]
      }]
    }]
  }]
}`

func TestDecodeJSONAndTransform(t *testing.T) {
	rspans, err := DecodeJSON([]byte(sampleJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	spans, rejected := ToSpans(rspans, IngestContext{
		WorkspaceID:    "ws",
		APIKeyID:       "key",
		ResolveProject: func(slug string) (string, bool) { return "proj", true },
	})
	if rejected != 0 || len(spans) != 1 {
		t.Fatalf("want 1 span 0 rejected, got %d spans %d rejected", len(spans), rejected)
	}
	s := spans[0]
	if s.Provider != "openai" || s.Model != "gpt-4o" || s.Operation != "chat" {
		t.Fatalf("enrichment wrong: provider=%s model=%s op=%s", s.Provider, s.Model, s.Operation)
	}
	if s.SessionID != "s1" || s.TraceID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1" {
		t.Fatalf("ids wrong: session=%s trace=%s", s.SessionID, s.TraceID)
	}
	if s.TokensInput != 1000 || s.TokensOutput != 500 {
		t.Fatalf("tokens wrong: in=%d out=%d", s.TokensInput, s.TokensOutput)
	}
	// gpt-4o estimate: in 1000/1e6*2.5 USD, out 500/1e6*10 USD -> >0 and estimated.
	if s.CostTotalMicrocents <= 0 || !s.CostIsEstimated {
		t.Fatalf("cost wrong: total=%d estimated=%v", s.CostTotalMicrocents, s.CostIsEstimated)
	}
	if s.WorkspaceID != "ws" || s.ProjectID != "proj" || s.APIKeyID != "key" {
		t.Fatalf("tenancy wrong: %+v", s)
	}
}

func TestProviderInferenceClaudeCode(t *testing.T) {
	a := Attrs{}
	if got := resolveProvider(a, "claude_code.telemetry", "claude_code.llm_request"); got != "anthropic" {
		t.Fatalf("claude code provider inference failed: %q", got)
	}
}

func TestTokenNormalizationInclusive(t *testing.T) {
	a := Attrs{
		"gen_ai.usage.input_tokens":             int64(1000),
		"gen_ai.usage.cache_read.input_tokens":  int64(200),
		"gen_ai.usage.output_tokens":            int64(50),
	}
	in, out, cr, cc, _ := resolveTokens(a)
	if in != 800 || cr != 200 || out != 50 || cc != 0 {
		t.Fatalf("normalization wrong: in=%d out=%d cr=%d cc=%d", in, out, cr, cc)
	}
}
