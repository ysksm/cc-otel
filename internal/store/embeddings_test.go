package store

import (
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
)

func TestEmbeddingsAndSearch(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ws, proj, _, err := s.Bootstrap("")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// Two spans with content (embeddable).
	spans := []model.Span{
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", SpanID: "a", StartTimeNs: 1, EndTimeNs: 2,
			Name: "chat", InputMessages: `[{"role":"user","content":"weather in Tokyo"}]`},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t2", SpanID: "b", StartTimeNs: 3, EndTimeNs: 4,
			Name: "chat", InputMessages: `[{"role":"user","content":"summarize the report"}]`},
	}
	if err := s.InsertSpans(spans); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if n, _ := s.CountEmbeddableSpans(proj.ID); n != 2 {
		t.Fatalf("embeddable spans: want 2, got %d", n)
	}
	need, err := s.SpansNeedingEmbedding(proj.ID, 10)
	if err != nil || len(need) != 2 {
		t.Fatalf("needing embedding: %d err=%v", len(need), err)
	}

	// Embed: spanA -> [1,0], spanB -> [0,1].
	if err := s.UpsertEmbedding(proj.ID, "t1", "a", "mock", []float32{1, 0}, "chat", "weather in Tokyo"); err != nil {
		t.Fatalf("upsert a: %v", err)
	}
	if err := s.UpsertEmbedding(proj.ID, "t2", "b", "mock", []float32{0, 1}, "chat", "summarize the report"); err != nil {
		t.Fatalf("upsert b: %v", err)
	}
	if n, _ := s.CountEmbeddings(proj.ID); n != 2 {
		t.Fatalf("embeddings: want 2, got %d", n)
	}
	// Now nothing needs embedding.
	need2, _ := s.SpansNeedingEmbedding(proj.ID, 10)
	if len(need2) != 0 {
		t.Fatalf("expected 0 needing embedding, got %d", len(need2))
	}

	// Query close to spanA -> spanA ranks first.
	hits, err := s.SearchSemantic(proj.ID, []float32{0.9, 0.1}, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 2 || hits[0].SpanID != "a" {
		t.Fatalf("ranking wrong: %+v", hits)
	}
	if hits[0].Score <= hits[1].Score {
		t.Fatalf("scores not ordered: %+v", hits)
	}
	if hits[0].Snippet == "" {
		t.Fatalf("expected snippet")
	}
}
