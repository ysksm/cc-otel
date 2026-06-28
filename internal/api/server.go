// Package api exposes the OTLP ingestion endpoint and the REST API the web UI
// consumes, plus static serving of the embedded SPA.
package api

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/ysksm/cc-otel/internal/auth"
	"github.com/ysksm/cc-otel/internal/llm"
	"github.com/ysksm/cc-otel/internal/store"
)

// Server wires the store to HTTP handlers.
type Server struct {
	store       *store.Store
	workspaceID string
	webFS       fs.FS       // embedded SPA (may be nil if not built)
	llmCfg      llm.Config  // default LLM config for evaluations (from env)
	auth        auth.Config // opt-in dashboard auth (from env)
}

// New constructs a Server bound to the default local workspace.
func New(st *store.Store, workspaceID string, webFS fs.FS) *Server {
	return &Server{
		store:       st,
		workspaceID: workspaceID,
		webFS:       webFS,
		llmCfg:      llm.ConfigFromEnv(),
		auth:        auth.ConfigFromEnv(),
	}
}

// AuthEnabled reports whether dashboard auth is on.
func (s *Server) AuthEnabled() bool { return s.auth.Enabled }

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// OTLP ingestion
	mux.HandleFunc("POST /v1/traces", s.handleIngestTraces)

	// REST API
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	mux.HandleFunc("GET /api/projects/{slug}", s.handleGetProject)
	mux.HandleFunc("GET /api/projects/{slug}/traces", s.handleListTraces)
	mux.HandleFunc("GET /api/projects/{slug}/traces/export", s.handleExportTraces)
	mux.HandleFunc("GET /api/projects/{slug}/traces/{traceId}", s.handleGetTrace)
	mux.HandleFunc("GET /api/projects/{slug}/traces/{traceId}/export", s.handleExportTrace)
	mux.HandleFunc("GET /api/projects/{slug}/sessions", s.handleListSessions)
	mux.HandleFunc("GET /api/projects/{slug}/analytics", s.handleAnalytics)
	mux.HandleFunc("GET /api/projects/{slug}/users", s.handleListUsers)
	mux.HandleFunc("GET /api/projects/{slug}/tools", s.handleListTools)
	mux.HandleFunc("GET /api/projects/{slug}/tools/{toolName}", s.handleGetTool)
	mux.HandleFunc("GET /api/projects/{slug}/api-keys", s.handleListAPIKeys)
	mux.HandleFunc("POST /api/projects/{slug}/api-keys", s.handleCreateAPIKey)

	// Scores (annotations) + evaluations (LLM-as-judge)
	mux.HandleFunc("GET /api/projects/{slug}/traces/{traceId}/scores", s.handleListTraceScores)
	mux.HandleFunc("POST /api/projects/{slug}/traces/{traceId}/scores", s.handleCreateTraceScore)
	mux.HandleFunc("GET /api/projects/{slug}/monitors", s.handleListMonitors)
	mux.HandleFunc("POST /api/projects/{slug}/monitors", s.handleCreateMonitor)
	mux.HandleFunc("POST /api/projects/{slug}/monitors/evaluate", s.handleEvaluateMonitors)
	mux.HandleFunc("GET /api/projects/{slug}/signals", s.handleListSignals)
	mux.HandleFunc("GET /api/projects/{slug}/evaluations", s.handleListEvaluations)
	mux.HandleFunc("POST /api/projects/{slug}/evaluations", s.handleCreateEvaluation)
	mux.HandleFunc("POST /api/projects/{slug}/evaluations/{evalId}/run", s.handleRunEvaluation)

	// Auth (opt-in)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleMe)
	mux.HandleFunc("GET /api/accounts", s.handleListAccounts)
	mux.HandleFunc("POST /api/accounts", s.handleCreateAccount)

	// Static SPA (and client-side routing fallback)
	mux.HandleFunc("/", s.handleStatic)

	return withCORS(s.authGate(mux))
}

// withCORS allows the Vite dev server (and any local origin) to call the API.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Latitude-Project")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return strings.TrimSpace(h)
}
