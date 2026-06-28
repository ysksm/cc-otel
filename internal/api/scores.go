package api

import (
	"net/http"
	"strings"

	"github.com/ysksm/cc-otel/internal/eval"
	"github.com/ysksm/cc-otel/internal/llm"
	"github.com/ysksm/cc-otel/internal/model"
	"github.com/ysksm/cc-otel/internal/store"
)

func (s *Server) handleListTraceScores(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	scores, err := s.store.ListScoresForTrace(p.ID, r.PathValue("traceId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if scores == nil {
		scores = []model.Score{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": scores})
}

func (s *Server) handleCreateTraceScore(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	var body struct {
		Name      string   `json:"name"`
		SpanID    string   `json:"spanId"`
		Value     float64  `json:"value"`
		Passed    *bool    `json:"passed"`
		Reasoning string   `json:"reasoning"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	passed := body.Value >= 0.5
	if body.Passed != nil {
		passed = *body.Passed
	}
	if body.Name == "" {
		body.Name = "annotation"
	}
	sc := model.Score{
		WorkspaceID: s.workspaceID, ProjectID: p.ID, TraceID: r.PathValue("traceId"), SpanID: body.SpanID,
		Source: "annotation", SourceID: "UI", Name: body.Name,
		Value: clamp01(body.Value), Passed: passed, Reasoning: body.Reasoning,
	}
	created, err := s.store.CreateScore(sc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, created)
}

func (s *Server) handleListEvaluations(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListEvaluations(p.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []model.Evaluation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCreateEvaluation(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	var body struct {
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		Prompt   string `json:"prompt"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if body.Slug == "" {
		body.Slug = slugify(body.Name)
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	e := model.Evaluation{
		WorkspaceID: s.workspaceID, ProjectID: p.ID, Name: body.Name, Slug: body.Slug,
		Prompt: body.Prompt, Provider: body.Provider, Model: body.Model, Enabled: enabled,
	}
	created, err := s.store.CreateEvaluation(e)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, created)
}

func (s *Server) handleRunEvaluation(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	var body struct {
		TraceID string `json:"traceId"`
	}
	if err := decodeJSON(r, &body); err != nil || body.TraceID == "" {
		writeError(w, http.StatusBadRequest, "traceId is required")
		return
	}
	e, err := s.store.GetEvaluation(p.ID, r.PathValue("evalId"))
	if err == store.ErrNotFound {
		writeError(w, http.StatusNotFound, "evaluation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	spans, err := s.store.ListTraceSpans(p.ID, body.TraceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Per-evaluation provider/model override on top of the server's env config.
	cfg := s.llmCfg.Merge(llm.Config{Provider: e.Provider, Model: e.Model})
	client := llm.New(cfg)

	score := model.Score{
		WorkspaceID: s.workspaceID, ProjectID: p.ID, TraceID: body.TraceID,
		Source: "evaluation", SourceID: e.ID, Name: e.Name,
	}
	verdict, usage, runErr := eval.Run(r.Context(), client, e, spans)
	if runErr != nil {
		score.Errored = true
		score.Passed = false
		score.Value = 0
		score.Reasoning = runErr.Error()
	} else {
		score.Value = verdict.Value
		score.Passed = verdict.Passed
		score.Reasoning = verdict.Reasoning
		score.Tokens = usage.InputTokens + usage.OutputTokens
	}
	created, err := s.store.CreateScore(score)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, created)
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
