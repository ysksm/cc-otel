package api

import (
	"net/http"

	"github.com/ysksm/cc-otel/internal/eval"
	"github.com/ysksm/cc-otel/internal/llm"
	"github.com/ysksm/cc-otel/internal/model"
)

func (s *Server) embedConfigured() bool { return s.embedCfg.Model != "" }

func (s *Server) handleSearchStatus(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	indexed, _ := s.store.CountEmbeddings(p.ID)
	total, _ := s.store.CountEmbeddableSpans(p.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": s.embedConfigured(),
		"model":      s.embedCfg.Model,
		"indexed":    indexed,
		"total":      total,
	})
}

// handleSearchIndex embeds a batch of un-indexed spans (incremental; call again
// until indexed == total). Returns how many were indexed this call.
func (s *Server) handleSearchIndex(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	if !s.embedConfigured() {
		writeError(w, http.StatusBadRequest, "embeddings are not configured (set CCOTEL_EMBED_MODEL)")
		return
	}
	spans, err := s.store.SpansNeedingEmbedding(p.ID, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(spans) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"indexed": 0, "done": true})
		return
	}
	client := llm.New(s.embedCfg)
	indexed := 0
	const batch = 64
	for start := 0; start < len(spans); start += batch {
		end := start + batch
		if end > len(spans) {
			end = len(spans)
		}
		chunk := spans[start:end]
		texts := make([]string, len(chunk))
		for i, sp := range chunk {
			texts[i] = eval.SpanText(sp)
		}
		vecs, err := client.Embed(r.Context(), texts)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		for i, sp := range chunk {
			if i >= len(vecs) {
				break
			}
			if err := s.store.UpsertEmbedding(p.ID, sp.TraceID, sp.SpanID, s.embedCfg.Model, vecs[i], sp.Name, texts[i]); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			indexed++
		}
	}
	remaining, _ := s.store.CountEmbeddableSpans(p.ID)
	have, _ := s.store.CountEmbeddings(p.ID)
	writeJSON(w, http.StatusOK, map[string]any{"indexed": indexed, "done": have >= remaining})
}

func (s *Server) handleSearchSemantic(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	if !s.embedConfigured() {
		writeError(w, http.StatusBadRequest, "embeddings are not configured (set CCOTEL_EMBED_MODEL)")
		return
	}
	var body struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	client := llm.New(s.embedCfg)
	vecs, err := client.Embed(r.Context(), []string{body.Query})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if len(vecs) == 0 {
		writeError(w, http.StatusBadGateway, "no embedding returned for query")
		return
	}
	hits, err := s.store.SearchSemantic(p.ID, vecs[0], body.Limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if hits == nil {
		hits = []model.SearchHit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}
