package store

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ysksm/cc-otel/internal/model"
)

// Monitor condition types.
const (
	CondTraceError = "trace_error"
	CondCostGt     = "cost_gt"     // threshold in USD
	CondLatencyGt  = "latency_gt"  // threshold in ms
	CondTokensGt   = "tokens_gt"   // threshold in tokens
	CondModelUsed  = "model_used"  // value_str = model name
)

// CreateMonitor inserts a monitor definition.
func (s *Store) CreateMonitor(m model.Monitor) (model.Monitor, error) {
	if m.ID == "" {
		m.ID = newID()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(`INSERT INTO monitors (id, workspace_id, project_id, name, condition_type, threshold, value_str, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.WorkspaceID, m.ProjectID, m.Name, m.ConditionType, m.Threshold, m.ValueStr, m.Enabled)
	if err != nil {
		return model.Monitor{}, err
	}
	return m, nil
}

const monitorCols = `id, workspace_id, project_id, name, condition_type, threshold, value_str, enabled, created_at`

func scanMonitor(sc interface{ Scan(...any) error }) (model.Monitor, error) {
	var m model.Monitor
	err := sc.Scan(&m.ID, &m.WorkspaceID, &m.ProjectID, &m.Name, &m.ConditionType, &m.Threshold, &m.ValueStr, &m.Enabled, &m.CreatedAt)
	return m, err
}

// ListMonitors returns monitors for a project. If enabledOnly, only enabled ones.
func (s *Store) ListMonitors(projectID string, enabledOnly bool) ([]model.Monitor, error) {
	q := `SELECT ` + monitorCols + ` FROM monitors WHERE project_id = ?`
	if enabledOnly {
		q += ` AND enabled`
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Monitor
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListSignals returns signals for a project, newest first.
func (s *Store) ListSignals(projectID string, limit int) ([]model.Signal, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id, monitor_id, monitor_name, trace_id, session_id, description, value, created_at
		FROM signals WHERE project_id = ? ORDER BY created_at DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Signal
	for rows.Next() {
		var sig model.Signal
		if err := rows.Scan(&sig.ID, &sig.MonitorID, &sig.MonitorName, &sig.TraceID, &sig.SessionID,
			&sig.Description, &sig.Value, &sig.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sig)
	}
	return out, rows.Err()
}

// createSignalIfNew inserts a signal unless one already exists for this
// (monitor, trace). Returns true if a new signal was created. Ingestion is
// serialized (single DB connection), so the check-then-insert is race-free.
func (s *Store) createSignalIfNew(m model.Monitor, t model.TraceSummary, value float64, desc string) (bool, error) {
	var n int
	if err := s.db.QueryRow(`SELECT count(*) FROM signals WHERE monitor_id = ? AND trace_id = ?`,
		m.ID, t.TraceID).Scan(&n); err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	if _, err := s.db.Exec(`INSERT INTO signals
		(id, workspace_id, project_id, monitor_id, monitor_name, trace_id, session_id, description, value)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		newID(), m.WorkspaceID, m.ProjectID, m.ID, m.Name, t.TraceID, t.SessionID, desc, value); err != nil {
		return false, err
	}
	return true, nil
}

// matchMonitor evaluates a monitor against a trace summary.
func matchMonitor(m model.Monitor, t model.TraceSummary) (matched bool, value float64, desc string) {
	switch m.ConditionType {
	case CondTraceError:
		if t.ErrorCount > 0 {
			return true, float64(t.ErrorCount), fmt.Sprintf("%d error span(s) in trace", t.ErrorCount)
		}
	case CondCostGt:
		usd := float64(t.CostTotalMicrocents) / 1e8
		if usd > m.Threshold {
			return true, usd, fmt.Sprintf("cost $%.4f exceeds $%.4f", usd, m.Threshold)
		}
	case CondLatencyGt:
		ms := float64(t.DurationNs) / 1e6
		if ms > m.Threshold {
			return true, ms, fmt.Sprintf("duration %.0fms exceeds %.0fms", ms, m.Threshold)
		}
	case CondTokensGt:
		total := float64(t.TokensInput + t.TokensOutput + t.TokensCacheRead + t.TokensCacheCreate + t.TokensReasoning)
		if total > m.Threshold {
			return true, total, fmt.Sprintf("%.0f tokens exceed %.0f", total, m.Threshold)
		}
	case CondModelUsed:
		if m.ValueStr != "" && strings.Contains(t.Models, `"`+m.ValueStr+`"`) {
			return true, 1, fmt.Sprintf("model %q used", m.ValueStr)
		}
	}
	return false, 0, ""
}

// evalMonitors runs the monitors against the summaries and creates new signals.
// Returns the number of new signals created.
func (s *Store) evalMonitors(monitors []model.Monitor, summaries []model.TraceSummary) (int, error) {
	created := 0
	for _, t := range summaries {
		for _, m := range monitors {
			if ok, value, desc := matchMonitor(m, t); ok {
				isNew, err := s.createSignalIfNew(m, t, value, desc)
				if err != nil {
					return created, err
				}
				if isNew {
					created++
				}
			}
		}
	}
	return created, nil
}

// EvaluateMonitorsForTraces evaluates enabled monitors against the given traces
// (used right after ingestion). Best-effort: it loads each trace summary by id.
func (s *Store) EvaluateMonitorsForTraces(projectID string, traceIDs []string) (int, error) {
	if len(traceIDs) == 0 {
		return 0, nil
	}
	monitors, err := s.ListMonitors(projectID, true)
	if err != nil || len(monitors) == 0 {
		return 0, err
	}
	var summaries []model.TraceSummary
	for _, id := range traceIDs {
		t, err := s.GetTrace(projectID, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return 0, err
		}
		summaries = append(summaries, t)
	}
	return s.evalMonitors(monitors, summaries)
}

// EvaluateAllMonitors backfills enabled monitors over recent traces in a project.
func (s *Store) EvaluateAllMonitors(projectID string, limit int) (int, error) {
	monitors, err := s.ListMonitors(projectID, true)
	if err != nil || len(monitors) == 0 {
		return 0, err
	}
	summaries, err := s.ExportTraces(projectID, model.TraceFilter{}, limit)
	if err != nil {
		return 0, err
	}
	return s.evalMonitors(monitors, summaries)
}
