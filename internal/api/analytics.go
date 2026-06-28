package api

import (
	"net/http"
	"time"
)

// rangePreset maps a range key to (lookback, bucket) durations.
var rangePresets = map[string]struct{ lookback, bucket time.Duration }{
	"1h":  {time.Hour, 5 * time.Minute},
	"24h": {24 * time.Hour, time.Hour},
	"7d":  {7 * 24 * time.Hour, 6 * time.Hour},
	"30d": {30 * 24 * time.Hour, 24 * time.Hour},
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	rng := r.URL.Query().Get("range")
	preset, found := rangePresets[rng]
	if !found {
		rng = "24h"
		preset = rangePresets[rng]
	}
	sinceNs := time.Now().Add(-preset.lookback).UnixNano()
	bucketNs := preset.bucket.Nanoseconds()

	data, err := s.store.Analytics(p.ID, sinceNs, bucketNs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	data.Range = rng
	writeJSON(w, http.StatusOK, data)
}
