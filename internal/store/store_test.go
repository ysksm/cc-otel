package store

import (
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
)

func TestStoreEndToEnd(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	ws, proj, token, err := s.Bootstrap("")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if ws.Slug != "local" || proj.Slug != "default" {
		t.Fatalf("unexpected defaults: ws=%+v proj=%+v", ws, proj)
	}
	if token == "" {
		t.Fatalf("expected a dev token on first bootstrap")
	}

	// API key round-trips.
	wsID, keyID, err := s.ResolveAPIKey(token)
	if err != nil || wsID != ws.ID || keyID == "" {
		t.Fatalf("resolve api key: ws=%s key=%s err=%v", wsID, keyID, err)
	}

	spans := []model.Span{
		{
			WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "trace1", SpanID: "spanA",
			SessionID: "sess1", StartTimeNs: 1_000, EndTimeNs: 2_000, Name: "chat",
			Operation: "chat", Provider: "anthropic", Model: "claude-opus",
			TokensInput: 10, TokensOutput: 5, CostTotalMicrocents: 1234,
			InputMessages: `[{"role":"user","content":"hi"}]`,
		},
		{
			WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "trace1", SpanID: "spanB",
			ParentSpanID: "spanA", SessionID: "sess1", StartTimeNs: 1_200, EndTimeNs: 1_800,
			Name: "tool", Operation: "execute_tool", StatusCode: 2, ToolName: "search",
		},
	}
	if err := s.InsertSpans(spans); err != nil {
		t.Fatalf("insert spans: %v", err)
	}

	traces, err := s.ListTraces(proj.ID, 50, 0)
	if err != nil {
		t.Fatalf("list traces: %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("want 1 trace, got %d", len(traces))
	}
	tr := traces[0]
	if tr.SpanCount != 2 || tr.ErrorCount != 1 {
		t.Fatalf("trace agg wrong: %+v", tr)
	}
	if tr.TokensInput != 10 || tr.CostTotalMicrocents != 1234 {
		t.Fatalf("trace tokens/cost wrong: %+v", tr)
	}
	if tr.Models != `["claude-opus"]` {
		t.Fatalf("trace models wrong: %q", tr.Models)
	}

	got, err := s.ListTraceSpans(proj.ID, "trace1")
	if err != nil {
		t.Fatalf("list trace spans: %v", err)
	}
	if len(got) != 2 || got[0].SpanID != "spanA" || got[1].ParentSpanID != "spanA" {
		t.Fatalf("spans wrong: %+v", got)
	}

	sessions, err := s.ListSessions(proj.ID, 50)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].TraceCount != 1 || sessions[0].SpanCount != 2 {
		t.Fatalf("session agg wrong: %+v", sessions)
	}
}
