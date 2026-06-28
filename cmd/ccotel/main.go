// Command ccotel runs the cc-otel server: OTLP ingestion, REST API and (when
// built) the embedded web UI, all backed by a single embedded DuckDB file.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ysksm/cc-otel/internal/api"
	"github.com/ysksm/cc-otel/internal/auth"
	"github.com/ysksm/cc-otel/internal/store"
	"github.com/ysksm/cc-otel/internal/webui"
)

func main() {
	addr := env("CCOTEL_ADDR", ":8080")
	dbPath := env("CCOTEL_DB_PATH", "./data/ccotel.duckdb")
	devToken := os.Getenv("CCOTEL_DEV_API_KEY")

	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			log.Fatalf("create data dir: %v", err)
		}
	}

	log.Printf("cc-otel: opening DuckDB at %s (downloading libduckdb if needed)...", dbPath)
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ws, proj, token, err := st.Bootstrap(devToken)
	if err != nil {
		log.Fatalf("bootstrap: %v", err)
	}
	if token != "" {
		log.Printf("cc-otel: created API key for ingestion: %s", token)
		log.Printf("cc-otel: (also written to %s)", filepath.Join(filepath.Dir(dbPath), "dev-api-key.txt"))
	}

	// Opt-in auth: bootstrap an admin account when CCOTEL_ADMIN_PASSWORD is set.
	if pw := os.Getenv("CCOTEL_ADMIN_PASSWORD"); pw != "" {
		email := env("CCOTEL_ADMIN_EMAIL", "admin@local")
		if _, err := st.GetAccountByEmail(email); err == store.ErrNotFound {
			hash, herr := auth.HashPassword(pw)
			if herr != nil {
				log.Fatalf("hash admin password: %v", herr)
			}
			if _, cerr := st.CreateAccount(email, hash, "admin"); cerr != nil {
				log.Fatalf("create admin account: %v", cerr)
			}
			log.Printf("cc-otel: created admin account %s (auth enabled)", email)
		} else {
			log.Printf("cc-otel: auth enabled (admin account %s)", email)
		}
	}

	webFS, ok := webui.FS()
	if !ok {
		log.Printf("cc-otel: web UI not embedded; serving API + placeholder page")
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(st, ws.ID, webFS).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("cc-otel: listening on http://localhost%s  (workspace=%s project=%s)", addr, ws.Slug, proj.Slug)
		log.Printf("cc-otel: OTLP endpoint: http://localhost%s/v1/traces", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Printf("cc-otel: shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
