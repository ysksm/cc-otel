package store

import (
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
)

func TestRetentionAndPurge(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ws, proj, _, err := s.Bootstrap("")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	if _, err := s.CreateMonitor(model.Monitor{
		WorkspaceID: ws.ID, ProjectID: proj.ID, Name: "errors", ConditionType: CondTraceError, Enabled: true,
	}); err != nil {
		t.Fatalf("monitor: %v", err)
	}
	// Old error trace + a newer trace.
	if err := s.InsertSpans([]model.Span{
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "old", SpanID: "a", StartTimeNs: 1000, EndTimeNs: 1100,
			StatusCode: 2, Model: "gpt-4o", InputMessages: `[{"role":"user","content":"hi"}]`},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "new", SpanID: "b", StartTimeNs: 10000, EndTimeNs: 10100,
			Model: "gpt-4o"},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// Generate a signal, an embedding and a score on the old trace.
	if _, err := s.EvaluateMonitorsForTraces(proj.ID, []string{"old", "new"}); err != nil {
		t.Fatalf("eval monitors: %v", err)
	}
	if err := s.UpsertEmbedding(proj.ID, "old", "a", "m", []float32{1, 0}, "n", "hi"); err != nil {
		t.Fatalf("embed: %v", err)
	}
	if _, err := s.CreateScore(model.Score{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "old",
		Source: "annotation", SourceID: "UI", Name: "x", Value: 1, Passed: true}); err != nil {
		t.Fatalf("score: %v", err)
	}

	st, err := s.StorageStats()
	if err != nil || st.Spans != 2 || st.Traces != 2 || st.OldestNs != 1000 || st.NewestNs != 10000 {
		t.Fatalf("stats wrong: %+v err=%v", st, err)
	}
	if st.Embeddings != 1 || st.Scores != 1 || st.Signals != 1 {
		t.Fatalf("aux counts wrong: %+v", st)
	}

	// Delete everything older than 5000ns: removes the old trace and its orphans.
	deleted, err := s.DeleteOlderThan(5000)
	if err != nil || deleted != 1 {
		t.Fatalf("delete older: n=%d err=%v", deleted, err)
	}
	st2, _ := s.StorageStats()
	if st2.Spans != 1 || st2.Embeddings != 0 || st2.Scores != 0 || st2.Signals != 0 {
		t.Fatalf("after retention wrong: %+v", st2)
	}

	// Purge removes the rest.
	if n, err := s.PurgeTelemetry(); err != nil || n != 1 {
		t.Fatalf("purge: n=%d err=%v", n, err)
	}
	st3, _ := s.StorageStats()
	if st3.Spans != 0 {
		t.Fatalf("after purge spans=%d", st3.Spans)
	}
}
