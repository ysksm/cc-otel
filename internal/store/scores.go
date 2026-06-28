package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/ysksm/cc-otel/internal/model"
)

// ---- Evaluations -----------------------------------------------------------

// CreateEvaluation inserts an evaluator definition.
func (s *Store) CreateEvaluation(e model.Evaluation) (model.Evaluation, error) {
	if e.ID == "" {
		e.ID = newID()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(`INSERT INTO evaluations (id, workspace_id, project_id, name, slug, prompt, provider, model, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.WorkspaceID, e.ProjectID, e.Name, e.Slug, e.Prompt, e.Provider, e.Model, e.Enabled)
	if err != nil {
		return model.Evaluation{}, err
	}
	return e, nil
}

func scanEvaluation(sc interface{ Scan(...any) error }) (model.Evaluation, error) {
	var e model.Evaluation
	err := sc.Scan(&e.ID, &e.WorkspaceID, &e.ProjectID, &e.Name, &e.Slug, &e.Prompt, &e.Provider, &e.Model, &e.Enabled, &e.CreatedAt)
	return e, err
}

const evalCols = `id, workspace_id, project_id, name, slug, prompt, provider, model, enabled, created_at`

// ListEvaluations returns evaluators for a project.
func (s *Store) ListEvaluations(projectID string) ([]model.Evaluation, error) {
	rows, err := s.db.Query(`SELECT `+evalCols+` FROM evaluations WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Evaluation
	for rows.Next() {
		e, err := scanEvaluation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetEvaluation returns an evaluator by id within a project.
func (s *Store) GetEvaluation(projectID, id string) (model.Evaluation, error) {
	row := s.db.QueryRow(`SELECT `+evalCols+` FROM evaluations WHERE project_id = ? AND id = ?`, projectID, id)
	e, err := scanEvaluation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

// ---- Scores ----------------------------------------------------------------

// CreateScore inserts a score (annotation or evaluation result).
func (s *Store) CreateScore(sc model.Score) (model.Score, error) {
	if sc.ID == "" {
		sc.ID = newID()
	}
	if sc.CreatedAt.IsZero() {
		sc.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(`INSERT INTO scores
		(id, workspace_id, project_id, trace_id, span_id, session_id, source, source_id, name,
		 value, passed, errored, reasoning, duration_ns, tokens, cost_microcents)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sc.ID, sc.WorkspaceID, sc.ProjectID, sc.TraceID, sc.SpanID, sc.SessionID, sc.Source, sc.SourceID, sc.Name,
		sc.Value, sc.Passed, sc.Errored, sc.Reasoning, sc.DurationNs, sc.Tokens, sc.CostMicrocents)
	if err != nil {
		return model.Score{}, err
	}
	return sc, nil
}

const scoreCols = `id, trace_id, span_id, session_id, source, source_id, name, value, passed, errored,
	reasoning, duration_ns, tokens, cost_microcents, created_at`

func scanScore(sc interface{ Scan(...any) error }) (model.Score, error) {
	var x model.Score
	err := sc.Scan(&x.ID, &x.TraceID, &x.SpanID, &x.SessionID, &x.Source, &x.SourceID, &x.Name, &x.Value,
		&x.Passed, &x.Errored, &x.Reasoning, &x.DurationNs, &x.Tokens, &x.CostMicrocents, &x.CreatedAt)
	return x, err
}

// ListScoresForTrace returns all scores attached to a trace, newest first.
func (s *Store) ListScoresForTrace(projectID, traceID string) ([]model.Score, error) {
	rows, err := s.db.Query(`SELECT `+scoreCols+` FROM scores WHERE project_id = ? AND trace_id = ? ORDER BY created_at DESC`,
		projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Score
	for rows.Next() {
		x, err := scanScore(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
