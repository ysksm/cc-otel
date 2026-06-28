package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/ysksm/cc-otel/internal/otlp"
	"github.com/ysksm/cc-otel/internal/store"
)

const maxIngestBytes = 32 << 20 // 32 MiB

// handleIngestTraces implements OTLP/HTTP POST /v1/traces (protobuf or JSON).
func (s *Server) handleIngestTraces(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		writeOTLPError(w, http.StatusUnauthorized, "missing Authorization bearer token")
		return
	}
	wsID, keyID, err := s.store.ResolveAPIKey(token)
	if err != nil {
		writeOTLPError(w, http.StatusUnauthorized, "invalid api key")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxIngestBytes))
	if err != nil {
		writeOTLPError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	ct := strings.ToLower(r.Header.Get("Content-Type"))
	var rspans []otlp.ResourceSpans
	if strings.Contains(ct, "json") {
		rspans, err = otlp.DecodeJSON(body)
	} else {
		rspans, err = otlp.DecodeProtobuf(body)
	}
	if err != nil {
		writeOTLPError(w, http.StatusBadRequest, err.Error())
		return
	}

	defaultSlug := r.Header.Get("X-Latitude-Project")
	cache := map[string]string{}
	resolve := func(slug string) (string, bool) {
		if slug == "" {
			slug = defaultSlug
		}
		if slug == "" {
			slug = "default"
		}
		if id, ok := cache[slug]; ok {
			return id, true
		}
		p, err := s.store.GetProjectBySlug(wsID, slug)
		if err == store.ErrNotFound {
			// Auto-create projects on first trace for local convenience.
			p, err = s.store.CreateProject(wsID, titleFromSlug(slug), slug)
		}
		if err != nil {
			return "", false
		}
		cache[slug] = p.ID
		return p.ID, true
	}

	spans, rejected := otlp.ToSpans(rspans, otlp.IngestContext{
		WorkspaceID:    wsID,
		APIKeyID:       keyID,
		ResolveProject: resolve,
	})
	if len(spans) > 0 {
		if err := s.store.InsertSpans(spans); err != nil {
			writeOTLPError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	s.store.TouchAPIKey(keyID)

	if rejected > 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"partialSuccess": map[string]any{
				"rejectedSpans": rejected,
				"errorMessage":  "some spans had an unresolved project",
			},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

// writeOTLPError responds with the google.rpc.Status-shaped body OTLP expects.
func writeOTLPError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"code": code, "message": msg})
}

func titleFromSlug(slug string) string {
	if slug == "" {
		return "Default"
	}
	parts := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
