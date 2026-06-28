package store

import (
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
)

func TestScoresAndEvaluations(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ws, proj, _, err := s.Bootstrap("")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// Evaluation create + get + list.
	e, err := s.CreateEvaluation(model.Evaluation{
		WorkspaceID: ws.ID, ProjectID: proj.ID, Name: "Correctness", Slug: "correctness",
		Prompt: "Is it correct?", Provider: "openai", Model: "gpt-4o", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create eval: %v", err)
	}
	got, err := s.GetEvaluation(proj.ID, e.ID)
	if err != nil || got.Slug != "correctness" || !got.Enabled {
		t.Fatalf("get eval: %+v err=%v", got, err)
	}
	evals, err := s.ListEvaluations(proj.ID)
	if err != nil || len(evals) != 1 {
		t.Fatalf("list evals: %d err=%v", len(evals), err)
	}

	// Annotation score + evaluation score on a trace.
	if _, err := s.CreateScore(model.Score{
		WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", Source: "annotation", SourceID: "UI",
		Name: "helpfulness", Value: 0.9, Passed: true, Reasoning: "great",
	}); err != nil {
		t.Fatalf("create annotation: %v", err)
	}
	if _, err := s.CreateScore(model.Score{
		WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", Source: "evaluation", SourceID: e.ID,
		Name: e.Name, Value: 0.4, Passed: false, Tokens: 17,
	}); err != nil {
		t.Fatalf("create eval score: %v", err)
	}

	scores, err := s.ListScoresForTrace(proj.ID, "t1")
	if err != nil {
		t.Fatalf("list scores: %v", err)
	}
	if len(scores) != 2 {
		t.Fatalf("want 2 scores, got %d", len(scores))
	}
	var hasAnn, hasEval bool
	for _, sc := range scores {
		if sc.Source == "annotation" && sc.Value == 0.9 && sc.Passed {
			hasAnn = true
		}
		if sc.Source == "evaluation" && sc.SourceID == e.ID && sc.Tokens == 17 {
			hasEval = true
		}
	}
	if !hasAnn || !hasEval {
		t.Fatalf("scores wrong: %+v", scores)
	}
}
