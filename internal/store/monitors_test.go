package store

import (
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
)

func TestMonitorsAndSignals(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ws, proj, _, err := s.Bootstrap("")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// Monitors: one for errors, one for cost > $0.005.
	if _, err := s.CreateMonitor(model.Monitor{
		WorkspaceID: ws.ID, ProjectID: proj.ID, Name: "errors", ConditionType: CondTraceError, Enabled: true,
	}); err != nil {
		t.Fatalf("create monitor: %v", err)
	}
	if _, err := s.CreateMonitor(model.Monitor{
		WorkspaceID: ws.ID, ProjectID: proj.ID, Name: "pricey", ConditionType: CondCostGt, Threshold: 0.005, Enabled: true,
	}); err != nil {
		t.Fatalf("create monitor2: %v", err)
	}

	// t1: error span. t2: expensive (cost 0.0065 USD = 650000 microcents).
	spans := []model.Span{
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", SpanID: "a", StartTimeNs: 100, EndTimeNs: 200,
			StatusCode: 2, Model: "gpt-4o"},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t2", SpanID: "b", StartTimeNs: 300, EndTimeNs: 400,
			Model: "gpt-4o", CostTotalMicrocents: 650000},
	}
	if err := s.InsertSpans(spans); err != nil {
		t.Fatalf("insert: %v", err)
	}

	created, err := s.EvaluateMonitorsForTraces(proj.ID, []string{"t1", "t2"})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if created != 2 { // errors->t1, pricey->t2
		t.Fatalf("want 2 signals created, got %d", created)
	}

	// Idempotent: re-evaluating creates no new signals.
	again, err := s.EvaluateMonitorsForTraces(proj.ID, []string{"t1", "t2"})
	if err != nil {
		t.Fatalf("re-evaluate: %v", err)
	}
	if again != 0 {
		t.Fatalf("expected no new signals on re-eval, got %d", again)
	}

	signals, err := s.ListSignals(proj.ID, 100)
	if err != nil {
		t.Fatalf("list signals: %v", err)
	}
	if len(signals) != 2 {
		t.Fatalf("want 2 signals, got %d: %+v", len(signals), signals)
	}
	var hasErr, hasCost bool
	for _, sig := range signals {
		if sig.MonitorName == "errors" && sig.TraceID == "t1" {
			hasErr = true
		}
		if sig.MonitorName == "pricey" && sig.TraceID == "t2" {
			hasCost = true
		}
	}
	if !hasErr || !hasCost {
		t.Fatalf("signals wrong: %+v", signals)
	}
}
