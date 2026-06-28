package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ysksm/cc-otel/internal/model"
	"github.com/ysksm/cc-otel/internal/store"
)

// mockEmbedServer returns [1,0] for inputs mentioning "weather", else [0,1].
func mockEmbedServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/embeddings") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		type item struct {
			Embedding []float32 `json:"embedding"`
		}
		var data []item
		for _, in := range req.Input {
			if strings.Contains(strings.ToLower(in), "weather") {
				data = append(data, item{[]float32{1, 0}})
			} else {
				data = append(data, item{[]float32{0, 1}})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
}

func TestSemanticSearchEndToEnd(t *testing.T) {
	embed := mockEmbedServer(t)
	defer embed.Close()

	t.Setenv("CCOTEL_EMBED_PROVIDER", "openai")
	t.Setenv("CCOTEL_EMBED_MODEL", "mock")
	t.Setenv("CCOTEL_EMBED_BASE_URL", embed.URL)

	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	ws, proj, _, err := st.Bootstrap("")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := st.InsertSpans([]model.Span{
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t1", SpanID: "a", StartTimeNs: 1, EndTimeNs: 2,
			Name: "chat", InputMessages: `[{"role":"user","content":"what is the weather in Tokyo"}]`},
		{WorkspaceID: ws.ID, ProjectID: proj.ID, TraceID: "t2", SpanID: "b", StartTimeNs: 3, EndTimeNs: 4,
			Name: "chat", InputMessages: `[{"role":"user","content":"summarize the quarterly report"}]`},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	h := New(st, ws.ID, nil).Handler()

	// Index.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/projects/default/search/index", strings.NewReader("{}")))
	if rec.Code != http.StatusOK {
		t.Fatalf("index status %d: %s", rec.Code, rec.Body.String())
	}
	var idx struct{ Indexed int `json:"indexed"` }
	_ = json.Unmarshal(rec.Body.Bytes(), &idx)
	if idx.Indexed != 2 {
		t.Fatalf("want indexed 2, got %d (%s)", idx.Indexed, rec.Body.String())
	}

	// Semantic search for weather -> spanA first.
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodPost, "/api/projects/default/search/semantic",
		strings.NewReader(`{"query":"weather forecast"}`)))
	if rec2.Code != http.StatusOK {
		t.Fatalf("search status %d: %s", rec2.Code, rec2.Body.String())
	}
	var res struct {
		Hits []model.SearchHit `json:"hits"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(res.Hits) != 2 || res.Hits[0].SpanID != "a" {
		t.Fatalf("ranking wrong: %+v", res.Hits)
	}
}
