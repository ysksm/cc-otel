package api

import (
	"net/http"
	"time"
)

func (s *Server) handleStorageStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.StorageStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// handleStorageCleanup deletes telemetry older than the given number of days.
func (s *Server) handleStorageCleanup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Days int `json:"days"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Days <= 0 {
		writeError(w, http.StatusBadRequest, "days must be a positive integer")
		return
	}
	cutoff := time.Now().Add(-time.Duration(body.Days) * 24 * time.Hour).UnixNano()
	deleted, err := s.store.DeleteOlderThan(cutoff)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deletedSpans": deleted})
}

// handleStoragePurge deletes all telemetry (keeps config/accounts).
func (s *Server) handleStoragePurge(w http.ResponseWriter, r *http.Request) {
	deleted, err := s.store.PurgeTelemetry()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deletedSpans": deleted})
}
