package store

import (
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
)

func TestAnalytics(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ws, proj, _, err := s.Bootstrap("")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	const bucket = int64(1000)
	spans := []model.Span{
		// bucket 0
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", SpanID: "a", StartTimeNs: 100, EndTimeNs: 300,
			Model: "gpt-4o", CostTotalMicrocents: 500, TokensInput: 10, TokensOutput: 4},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", SpanID: "b", StartTimeNs: 200, EndTimeNs: 400,
			StatusCode: 2, Model: "gpt-4o", CostTotalMicrocents: 100},
		// bucket 1000
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t2", SpanID: "c", StartTimeNs: 1500, EndTimeNs: 1700,
			Model: "claude-sonnet", CostTotalMicrocents: 900, TokensInput: 20, TokensOutput: 6},
	}
	if err := s.InsertSpans(spans); err != nil {
		t.Fatalf("insert: %v", err)
	}

	a, err := s.Analytics(proj.ID, 0, bucket)
	if err != nil {
		t.Fatalf("analytics: %v", err)
	}
	if len(a.Series) != 2 {
		t.Fatalf("want 2 buckets, got %d: %+v", len(a.Series), a.Series)
	}
	if a.Series[0].BucketStartNs != 0 || a.Series[0].TraceCount != 1 || a.Series[0].ErrorCount != 1 {
		t.Fatalf("bucket0 wrong: %+v", a.Series[0])
	}
	if a.Series[1].BucketStartNs != 1000 || a.Series[1].SpanCount != 1 {
		t.Fatalf("bucket1 wrong: %+v", a.Series[1])
	}
	if a.Totals.TraceCount != 2 || a.Totals.ErrorCount != 1 || a.Totals.CostTotalMicrocents != 1500 {
		t.Fatalf("totals wrong: %+v", a.Totals)
	}
	if len(a.TopModels) != 2 || a.TopModels[0].Model != "claude-sonnet" { // 900 > 600
		t.Fatalf("top models wrong: %+v", a.TopModels)
	}
}
