package store

import (
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
)

func TestContentSearch(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ws, proj, _, err := s.Bootstrap("")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	spans := []model.Span{
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", SpanID: "a", StartTimeNs: 100, EndTimeNs: 200,
			Name: "chat", Operation: "chat",
			InputMessages:  `[{"role":"user","content":"What is the weather in Tokyo?"}]`,
			OutputMessages: `[{"role":"assistant","content":"It is sunny."}]`},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t2", SpanID: "b", StartTimeNs: 300, EndTimeNs: 400,
			Name: "chat", Operation: "chat",
			InputMessages: `[{"role":"user","content":"Summarize this report."}]`},
	}
	if err := s.InsertSpans(spans); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Search by message content should find only t1.
	got, err := s.ListTraces(proj.ID, 50, 0, model.TraceFilter{Search: "Tokyo"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 1 || got[0].TraceID != "t1" {
		t.Fatalf("content search wrong: %+v", got)
	}

	// A term in the other trace finds only t2.
	got2, err := s.ListTraces(proj.ID, 50, 0, model.TraceFilter{Search: "Summarize"})
	if err != nil {
		t.Fatalf("search2: %v", err)
	}
	if len(got2) != 1 || got2[0].TraceID != "t2" {
		t.Fatalf("content search2 wrong: %+v", got2)
	}

	// A non-matching term returns nothing.
	none, err := s.ListTraces(proj.ID, 50, 0, model.TraceFilter{Search: "zzz-no-match"})
	if err != nil {
		t.Fatalf("search3: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no matches, got %d", len(none))
	}
}
