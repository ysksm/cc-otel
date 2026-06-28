package store

import "github.com/ysksm/cc-otel/internal/model"

// ListUsers aggregates spans by end-user (user.id) for a project.
func (s *Store) ListUsers(projectID string, limit int) ([]model.UserSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT
		user_id,
		arg_max(user_email, start_time_ns) AS user_email,
		count(DISTINCT trace_id)            AS trace_count,
		count(*)                            AS span_count,
		count(*) FILTER (WHERE status_code = 2) AS error_count,
		sum(tokens_input)                   AS tokens_input,
		sum(tokens_output)                  AS tokens_output,
		sum(cost_total_microcents)          AS cost_total_microcents,
		max(end_time_ns)                    AS last_seen_ns,
		coalesce(CAST(to_json(list(DISTINCT model) FILTER (WHERE model <> '')) AS VARCHAR), '[]') AS models
		FROM spans WHERE project_id = ? AND user_id <> ''
		GROUP BY user_id ORDER BY last_seen_ns DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.UserSummary
	for rows.Next() {
		var u model.UserSummary
		if err := rows.Scan(&u.UserID, &u.UserEmail, &u.TraceCount, &u.SpanCount, &u.ErrorCount,
			&u.TokensInput, &u.TokensOutput, &u.CostTotalMicrocents, &u.LastSeenNs, &u.Models); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ListTools aggregates tool-execution spans by tool name for a project.
func (s *Store) ListTools(projectID string, limit int) ([]model.ToolSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT
		tool_name,
		count(*)                                  AS call_count,
		count(*) FILTER (WHERE status_code = 2)   AS error_count,
		CAST(avg(end_time_ns - start_time_ns) AS BIGINT) AS avg_duration_ns,
		max(end_time_ns)                          AS last_seen_ns
		FROM spans WHERE project_id = ? AND tool_name <> ''
		GROUP BY tool_name ORDER BY call_count DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ToolSummary
	for rows.Next() {
		var t model.ToolSummary
		if err := rows.Scan(&t.ToolName, &t.CallCount, &t.ErrorCount, &t.AvgDurationNs, &t.LastSeenNs); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListToolCalls returns recent spans for a given tool name (newest first).
func (s *Store) ListToolCalls(projectID, toolName string, limit int) ([]model.Span, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query("SELECT "+spanSelect+
		" FROM spans WHERE project_id = ? AND tool_name = ? ORDER BY start_time_ns DESC LIMIT ?",
		projectID, toolName, limit)
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
