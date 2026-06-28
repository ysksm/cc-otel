package api

import (
	"net/http"

	"github.com/ysksm/cc-otel/internal/model"
	"github.com/ysksm/cc-otel/internal/store"
)

var validConditions = map[string]bool{
	store.CondTraceError: true,
	store.CondCostGt:     true,
	store.CondLatencyGt:  true,
	store.CondTokensGt:   true,
	store.CondModelUsed:  true,
}

func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListMonitors(p.ID, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []model.Monitor{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	var body struct {
		Name          string  `json:"name"`
		ConditionType string  `json:"conditionType"`
		Threshold     float64 `json:"threshold"`
		ValueStr      string  `json:"valueStr"`
		Enabled       *bool   `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !validConditions[body.ConditionType] {
		writeError(w, http.StatusBadRequest, "invalid conditionType")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	m, err := s.store.CreateMonitor(model.Monitor{
		WorkspaceID: s.workspaceID, ProjectID: p.ID, Name: body.Name,
		ConditionType: body.ConditionType, Threshold: body.Threshold, ValueStr: body.ValueStr, Enabled: enabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// handleEvaluateMonitors backfills enabled monitors over recent traces.
func (s *Server) handleEvaluateMonitors(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	created, err := s.store.EvaluateAllMonitors(p.ID, 10000)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"created": created})
}

func (s *Server) handleListSignals(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListSignals(p.ID, atoiDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []model.Signal{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
