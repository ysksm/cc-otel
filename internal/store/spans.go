package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/ysksm/cc-otel/internal/model"
)

// spanInsertColumns is the column order used for inserts (ingested_at uses its
// DEFAULT). spanInsertArgs must produce values in exactly this order.
var spanInsertColumns = []string{
	"workspace_id", "project_id", "api_key_id", "session_id", "trace_id", "span_id", "parent_span_id",
	"start_time_ns", "end_time_ns", "name", "service_name", "kind", "status_code", "status_message",
	"error_type", "scope_name", "scope_version", "operation", "provider", "model", "response_model",
	"tokens_input", "tokens_output", "tokens_cache_read", "tokens_cache_create", "tokens_reasoning",
	"cost_input_microcents", "cost_output_microcents", "cost_total_microcents", "cost_is_estimated",
	"time_to_first_token_ns", "is_streaming", "response_id", "finish_reasons", "user_id", "user_email",
	"tool_call_id", "tool_name", "tool_input", "tool_output", "tool_names",
	"input_messages", "output_messages", "system_instructions", "tool_definitions", "events_json", "links_json",
	"tags", "attributes", "resource",
}

func jarr(s string) string {
	if strings.TrimSpace(s) == "" {
		return "[]"
	}
	return s
}

func jobj(s string) string {
	if strings.TrimSpace(s) == "" {
		return "{}"
	}
	return s
}

func spanInsertArgs(s model.Span) []any {
	return []any{
		s.WorkspaceID, s.ProjectID, s.APIKeyID, s.SessionID, s.TraceID, s.SpanID, s.ParentSpanID,
		s.StartTimeNs, s.EndTimeNs, s.Name, s.ServiceName, s.Kind, s.StatusCode, s.StatusMessage,
		s.ErrorType, s.ScopeName, s.ScopeVersion, s.Operation, s.Provider, s.Model, s.ResponseModel,
		s.TokensInput, s.TokensOutput, s.TokensCacheRead, s.TokensCacheCreate, s.TokensReasoning,
		s.CostInputMicrocents, s.CostOutputMicrocents, s.CostTotalMicrocents, s.CostIsEstimated,
		s.TimeToFirstTokenNs, s.IsStreaming, s.ResponseID, jarr(s.FinishReasons), s.UserID, s.UserEmail,
		s.ToolCallID, s.ToolName, s.ToolInput, s.ToolOutput, jarr(s.ToolNames),
		jarr(s.InputMessages), jarr(s.OutputMessages), jarr(s.SystemInstructions), jarr(s.ToolDefinitions),
		jarr(s.EventsJSON), jarr(s.LinksJSON), jarr(s.Tags), jobj(s.Attributes), jobj(s.Resource),
	}
}

// InsertSpans writes spans in a single transaction and marks first_trace_at on
// any involved projects that did not yet have a trace.
func (s *Store) InsertSpans(spans []model.Span) error {
	if len(spans) == 0 {
		return nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(spanInsertColumns)), ",")
	query := "INSERT INTO spans (" + strings.Join(spanInsertColumns, ", ") + ") VALUES (" + placeholders + ")"

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	projects := map[string]struct{}{}
	for _, sp := range spans {
		if _, err := stmt.Exec(spanInsertArgs(sp)...); err != nil {
			return err
		}
		if sp.ProjectID != "" {
			projects[sp.ProjectID] = struct{}{}
		}
	}
	for pid := range projects {
		if _, err := tx.Exec(`UPDATE projects SET first_trace_at = now() WHERE id = ? AND first_trace_at IS NULL`, pid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// traceSelect lists the columns of the traces view in TraceSummary order.
const traceSelect = `trace_id, session_id, span_count, error_count, min_start_time_ns, max_end_time_ns,
	duration_ns, tokens_input, tokens_output, tokens_cache_read, tokens_cache_create, tokens_reasoning,
	cost_total_microcents, CAST(models AS VARCHAR), CAST(providers AS VARCHAR), root_span_name, root_span_id`

func scanTrace(sc interface{ Scan(...any) error }) (model.TraceSummary, error) {
	var t model.TraceSummary
	err := sc.Scan(&t.TraceID, &t.SessionID, &t.SpanCount, &t.ErrorCount, &t.StartTimeNs, &t.EndTimeNs,
		&t.DurationNs, &t.TokensInput, &t.TokensOutput, &t.TokensCacheRead, &t.TokensCacheCreate, &t.TokensReasoning,
		&t.CostTotalMicrocents, &t.Models, &t.Providers, &t.RootSpanName, &t.RootSpanID)
	return t, err
}

// ListTraces returns trace summaries for a project, newest first. beforeNs is a
// keyset cursor on min_start_time_ns (0 = from newest).
func (s *Store) ListTraces(projectID string, limit int, beforeNs int64) ([]model.TraceSummary, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := []any{projectID}
	where := "project_id = ?"
	if beforeNs > 0 {
		where += " AND min_start_time_ns < ?"
		args = append(args, beforeNs)
	}
	args = append(args, limit)
	rows, err := s.db.Query("SELECT "+traceSelect+" FROM traces WHERE "+where+
		" ORDER BY min_start_time_ns DESC LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.TraceSummary
	for rows.Next() {
		t, err := scanTrace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTrace returns a single trace summary.
func (s *Store) GetTrace(projectID, traceID string) (model.TraceSummary, error) {
	row := s.db.QueryRow("SELECT "+traceSelect+" FROM traces WHERE project_id = ? AND trace_id = ?", projectID, traceID)
	t, err := scanTrace(row)
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

// spanSelect lists full span columns (JSON cast to text) in Span scan order.
const spanSelect = `session_id, trace_id, span_id, parent_span_id, start_time_ns, end_time_ns, name,
	service_name, kind, status_code, status_message, error_type, scope_name, scope_version,
	operation, provider, model, response_model, tokens_input, tokens_output, tokens_cache_read,
	tokens_cache_create, tokens_reasoning, cost_input_microcents, cost_output_microcents,
	cost_total_microcents, cost_is_estimated, time_to_first_token_ns, is_streaming, response_id,
	CAST(finish_reasons AS VARCHAR), user_id, user_email, tool_call_id, tool_name, tool_input, tool_output,
	CAST(tool_names AS VARCHAR), CAST(input_messages AS VARCHAR), CAST(output_messages AS VARCHAR),
	CAST(system_instructions AS VARCHAR), CAST(tool_definitions AS VARCHAR), CAST(events_json AS VARCHAR),
	CAST(links_json AS VARCHAR), CAST(tags AS VARCHAR), CAST(attributes AS VARCHAR), CAST(resource AS VARCHAR)`

func scanSpan(sc interface{ Scan(...any) error }) (model.Span, error) {
	var s model.Span
	err := sc.Scan(
		&s.SessionID, &s.TraceID, &s.SpanID, &s.ParentSpanID, &s.StartTimeNs, &s.EndTimeNs, &s.Name,
		&s.ServiceName, &s.Kind, &s.StatusCode, &s.StatusMessage, &s.ErrorType, &s.ScopeName, &s.ScopeVersion,
		&s.Operation, &s.Provider, &s.Model, &s.ResponseModel, &s.TokensInput, &s.TokensOutput, &s.TokensCacheRead,
		&s.TokensCacheCreate, &s.TokensReasoning, &s.CostInputMicrocents, &s.CostOutputMicrocents,
		&s.CostTotalMicrocents, &s.CostIsEstimated, &s.TimeToFirstTokenNs, &s.IsStreaming, &s.ResponseID,
		&s.FinishReasons, &s.UserID, &s.UserEmail, &s.ToolCallID, &s.ToolName, &s.ToolInput, &s.ToolOutput,
		&s.ToolNames, &s.InputMessages, &s.OutputMessages, &s.SystemInstructions, &s.ToolDefinitions,
		&s.EventsJSON, &s.LinksJSON, &s.Tags, &s.Attributes, &s.Resource,
	)
	return s, err
}

// ListTraceSpans returns all spans of a trace ordered by start time.
func (s *Store) ListTraceSpans(projectID, traceID string) ([]model.Span, error) {
	rows, err := s.db.Query("SELECT "+spanSelect+" FROM spans WHERE project_id = ? AND trace_id = ? ORDER BY start_time_ns ASC",
		projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Span
	for rows.Next() {
		sp, err := scanSpan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

// ListSessions returns session summaries for a project, newest first.
func (s *Store) ListSessions(projectID string, limit int) ([]model.SessionSummary, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT session_id, trace_count, span_count, error_count, min_start_time_ns,
		max_end_time_ns, duration_ns, tokens_input, tokens_output, cost_total_microcents,
		CAST(models AS VARCHAR), CAST(providers AS VARCHAR), user_id
		FROM sessions WHERE project_id = ? ORDER BY min_start_time_ns DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SessionSummary
	for rows.Next() {
		var ss model.SessionSummary
		if err := rows.Scan(&ss.SessionID, &ss.TraceCount, &ss.SpanCount, &ss.ErrorCount, &ss.StartTimeNs,
			&ss.EndTimeNs, &ss.DurationNs, &ss.TokensInput, &ss.TokensOutput, &ss.CostTotalMicrocents,
			&ss.Models, &ss.Providers, &ss.UserID); err != nil {
			return nil, err
		}
		out = append(out, ss)
	}
	return out, rows.Err()
}
