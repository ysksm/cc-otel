package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ysksm/cc-otel/internal/model"
	"github.com/ysksm/cc-otel/internal/store"
)

// handleExportTraces streams the project's traces (matching the same filters as
// the list endpoint) as CSV (default) or JSON for download.
func (s *Server) handleExportTraces(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	filter := model.TraceFilter{
		Search:      q.Get("q"),
		Provider:    q.Get("provider"),
		Model:       q.Get("model"),
		ErrorsOnly:  q.Get("errors") == "true" || q.Get("errorsOnly") == "true",
		MinDuration: atoi64Default(q.Get("minDuration"), 0),
	}
	limit := atoiDefault(q.Get("limit"), 10000)
	traces, err := s.store.ExportTraces(p.ID, filter, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stamp := time.Now().UTC().Format("20060102-150405")
	if q.Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ccotel-traces-%s-%s.json"`, p.Slug, stamp))
		if traces == nil {
			traces = []model.TraceSummary{}
		}
		_ = json.NewEncoder(w).Encode(traces)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ccotel-traces-%s-%s.csv"`, p.Slug, stamp))
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{
		"trace_id", "session_id", "start_time", "duration_ms", "span_count", "error_count",
		"tokens_input", "tokens_output", "cost_usd", "models", "providers", "root_span_name",
		"score_count", "avg_score",
	})
	for _, t := range traces {
		avg := ""
		if t.AvgScore != nil {
			avg = strconv.FormatFloat(*t.AvgScore, 'f', 4, 64)
		}
		_ = cw.Write([]string{
			t.TraceID,
			t.SessionID,
			time.Unix(0, t.StartTimeNs).UTC().Format(time.RFC3339Nano),
			strconv.FormatFloat(float64(t.DurationNs)/1e6, 'f', 3, 64),
			strconv.FormatInt(t.SpanCount, 10),
			strconv.FormatInt(t.ErrorCount, 10),
			strconv.FormatInt(t.TokensInput, 10),
			strconv.FormatInt(t.TokensOutput, 10),
			strconv.FormatFloat(float64(t.CostTotalMicrocents)/1e8, 'f', 6, 64),
			t.Models,
			t.Providers,
			t.RootSpanName,
			strconv.FormatInt(t.ScoreCount, 10),
			avg,
		})
	}
}

// handleExportTrace downloads a single trace (summary + spans) as JSON.
func (s *Server) handleExportTrace(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ccotel-trace-%s.json"`, traceID))
	_ = json.NewEncoder(w).Encode(map[string]any{"trace": summary, "spans": spans})
}
