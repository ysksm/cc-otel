package store

import (
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
)

func TestUsersAndToolsAggregation(t *testing.T) {
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
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", SpanID: "a", StartTimeNs: 100, EndTimeNs: 300,
			UserID: "alice", Model: "gpt-4o", TokensInput: 10, TokensOutput: 5},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", SpanID: "b", StartTimeNs: 150, EndTimeNs: 250,
			Operation: "execute_tool", ToolName: "search", StatusCode: 2},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t2", SpanID: "c", StartTimeNs: 400, EndTimeNs: 600,
			UserID: "alice", Model: "gpt-4o", TokensInput: 20, TokensOutput: 8},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t2", SpanID: "d", StartTimeNs: 420, EndTimeNs: 720,
			Operation: "execute_tool", ToolName: "search"},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t3", SpanID: "e", StartTimeNs: 700, EndTimeNs: 900,
			UserID: "bob", Model: "claude-sonnet"},
	}
	if err := s.InsertSpans(spans); err != nil {
		t.Fatalf("insert: %v", err)
	}

	users, err := s.ListUsers(proj.ID, 100)
	if err != nil {
		t.Fatalf("users: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("want 2 users, got %d", len(users))
	}
	// alice is most recent? bob last_seen=900 > alice 600, so bob first.
	var alice *model.UserSummary
	for i := range users {
		if users[i].UserID == "alice" {
			alice = &users[i]
		}
	}
	if alice == nil || alice.TraceCount != 2 || alice.SpanCount != 2 || alice.TokensInput != 30 {
		t.Fatalf("alice agg wrong: %+v", alice)
	}

	tools, err := s.ListTools(proj.ID, 100)
	if err != nil {
		t.Fatalf("tools: %v", err)
	}
	if len(tools) != 1 || tools[0].ToolName != "search" || tools[0].CallCount != 2 || tools[0].ErrorCount != 1 {
		t.Fatalf("tools agg wrong: %+v", tools)
	}
	if tools[0].AvgDurationNs != 200 { // (100 + 300) / 2
		t.Fatalf("avg duration wrong: %d", tools[0].AvgDurationNs)
	}

	calls, err := s.ListToolCalls(proj.ID, "search", 50)
	if err != nil {
		t.Fatalf("tool calls: %v", err)
	}
	if len(calls) != 2 || calls[0].SpanID != "d" { // newest first
		t.Fatalf("tool calls wrong: %+v", calls)
	}
}
