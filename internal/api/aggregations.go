package api

import (
	"net/http"

	"github.com/ysksm/cc-otel/internal/model"
)

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListUsers(p.ID, atoiDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []model.UserSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListTools(p.ID, atoiDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []model.ToolSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleGetTool(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	toolName := r.PathValue("toolName")
	calls, err := s.store.ListToolCalls(p.ID, toolName, atoiDefault(r.URL.Query().Get("limit"), 50))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if calls == nil {
		calls = []model.Span{}
	}
	// Derive the summary row for this tool.
	var summary model.ToolSummary
	if tools, err := s.store.ListTools(p.ID, 500); err == nil {
		for _, t := range tools {
			if t.ToolName == toolName {
				summary = t
				break
			}
		}
	}
	if summary.ToolName == "" {
		summary.ToolName = toolName
	}
	writeJSON(w, http.StatusOK, map[string]any{"tool": summary, "calls": calls})
}
