package api

import (
	"net/http"
	"strconv"

	"github.com/ysksm/cc-otel/internal/model"
	"github.com/ysksm/cc-otel/internal/store"
)

func (s *Server) project(w http.ResponseWriter, r *http.Request) (model.Project, bool) {
	slug := r.PathValue("slug")
	p, err := s.store.GetProjectBySlug(s.workspaceID, slug)
	if err == store.ErrNotFound {
		writeError(w, http.StatusNotFound, "project not found")
		return model.Project{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return model.Project{}, false
	}
	return p, true
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListProjects(s.workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []model.Project{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Slug == "" {
		writeError(w, http.StatusBadRequest, "slug is required")
		return
	}
	if body.Name == "" {
		body.Name = titleFromSlug(body.Slug)
	}
	p, err := s.store.CreateProject(s.workspaceID, body.Name, body.Slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleListTraces(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	limit := atoiDefault(r.URL.Query().Get("limit"), 50)
	before := atoi64Default(r.URL.Query().Get("before"), 0)
	traces, err := s.store.ListTraces(p.ID, limit, before)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if traces == nil {
		traces = []model.TraceSummary{}
	}
	var nextCursor string
	if len(traces) == limit && limit > 0 {
		nextCursor = strconv.FormatInt(traces[len(traces)-1].StartTimeNs, 10)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": traces, "nextCursor": nextCursor})
}

func (s *Server) handleGetTrace(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	traceID := r.PathValue("traceId")
	summary, err := s.store.GetTrace(p.ID, traceID)
	if err == store.ErrNotFound {
		writeError(w, http.StatusNotFound, "trace not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	spans, err := s.store.ListTraceSpans(p.ID, traceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if spans == nil {
		spans = []model.Span{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"trace": summary, "spans": spans})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	limit := atoiDefault(r.URL.Query().Get("limit"), 50)
	sessions, err := s.store.ListSessions(p.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sessions == nil {
		sessions = []model.SessionSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": sessions})
}

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.project(w, r); !ok {
		return
	}
	keys, err := s.store.ListAPIKeys(s.workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if keys == nil {
		keys = []model.APIKey{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": keys})
}

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.project(w, r); !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	_ = decodeJSON(r, &body)
	if body.Name == "" {
		body.Name = "API key"
	}
	key, err := s.store.CreateAPIKey(s.workspaceID, body.Name, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func atoi64Default(s string, def int64) int64 {
	if s == "" {
		return def
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return def
}
