package store

import (
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
)

func TestTraceFiltersAndScoreRollup(t *testing.T) {
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
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "ok1", SpanID: "a", StartTimeNs: 100, EndTimeNs: 1100,
			Name: "chat openai", Provider: "openai", Model: "gpt-4o"},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "err1", SpanID: "b", StartTimeNs: 200, EndTimeNs: 5200,
			Name: "chat claude", Provider: "anthropic", Model: "claude-sonnet", StatusCode: 2},
	}
	if err := s.InsertSpans(spans); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// Add scores to ok1.
	for _, sc := range []model.Score{
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "ok1", Source: "annotation", Value: 0.8, Passed: true},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "ok1", Source: "evaluation", Value: 0.2, Passed: false},
	} {
		if _, err := s.CreateScore(sc); err != nil {
			t.Fatalf("score: %v", err)
		}
	}

	all, _ := s.ListTraces(proj.ID, 50, 0, model.TraceFilter{})
	if len(all) != 2 {
		t.Fatalf("want 2 traces, got %d", len(all))
	}

	// errorsOnly
	errs, _ := s.ListTraces(proj.ID, 50, 0, model.TraceFilter{ErrorsOnly: true})
	if len(errs) != 1 || errs[0].TraceID != "err1" {
		t.Fatalf("errorsOnly wrong: %+v", errs)
	}

	// provider filter
	op, _ := s.ListTraces(proj.ID, 50, 0, model.TraceFilter{Provider: "openai"})
	if len(op) != 1 || op[0].TraceID != "ok1" {
		t.Fatalf("provider filter wrong: %+v", op)
	}

	// model filter
	cl, _ := s.ListTraces(proj.ID, 50, 0, model.TraceFilter{Model: "claude-sonnet"})
	if len(cl) != 1 || cl[0].TraceID != "err1" {
		t.Fatalf("model filter wrong: %+v", cl)
	}

	// search by name
	se, _ := s.ListTraces(proj.ID, 50, 0, model.TraceFilter{Search: "openai"})
	if len(se) != 1 || se[0].TraceID != "ok1" {
		t.Fatalf("search wrong: %+v", se)
	}

	// minDuration (err1 is 5000ns, ok1 is 1000ns)
	md, _ := s.ListTraces(proj.ID, 50, 0, model.TraceFilter{MinDuration: 2000})
	if len(md) != 1 || md[0].TraceID != "err1" {
		t.Fatalf("minDuration wrong: %+v", md)
	}

	// score rollup on ok1
	var ok1 *model.TraceSummary
	for i := range all {
		if all[i].TraceID == "ok1" {
			ok1 = &all[i]
		}
	}
	if ok1 == nil || ok1.ScoreCount != 2 || ok1.PassedCount != 1 || ok1.FailedCount != 1 {
		t.Fatalf("score rollup wrong: %+v", ok1)
	}
	if ok1.AvgScore == nil || *ok1.AvgScore != 0.5 {
		t.Fatalf("avg score wrong: %v", ok1.AvgScore)
	}
}
