// Package store is the DuckDB-backed persistence layer for cc-otel.
package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ysksm/cc-otel/internal/duckdblib"
	_ "github.com/ysksm/cc-otel/internal/pduckdb" // registers the "duckdb" driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store wraps a DuckDB connection and the schema/queries cc-otel needs.
type Store struct {
	db      *sql.DB
	dataDir string
	dbPath  string
}

// Open ensures libduckdb is available, opens the database file, applies the
// schema and returns a ready Store. dbPath may be ":memory:" or a file path.
func Open(dbPath string) (*Store, error) {
	if _, err := duckdblib.Ensure(); err != nil {
		return nil, fmt.Errorf("ensure libduckdb: %w", err)
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}
	// DuckDB is single-writer; serialize access through one connection so
	// concurrent ingestion and reads do not race the embedded engine.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	dir := "."
	if dbPath != ":memory:" && dbPath != "" {
		dir = filepath.Dir(dbPath)
	}
	s := &Store{db: db, dataDir: dir, dbPath: dbPath}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// DB exposes the underlying *sql.DB (used sparingly).
func (s *Store) DB() *sql.DB { return s.db }

// Close releases the database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		b, err := migrationsFS.ReadFile("migrations/" + n)
		if err != nil {
			return err
		}
		for _, stmt := range splitSQL(string(b)) {
			if _, err := s.db.Exec(stmt); err != nil {
				return fmt.Errorf("migration %s (%s): %w", n, firstLine(stmt), err)
			}
		}
	}
	return nil
}

// splitSQL strips line comments and splits a script into individual statements.
// The schema contains no semicolons inside string literals, so a simple split is
// safe and keeps us independent of multi-statement driver behaviour.
func splitSQL(script string) []string {
	var sb strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	var out []string
	for _, part := range strings.Split(sb.String(), ";") {
		if strings.TrimSpace(part) != "" {
			out = append(out, part)
		}
	}
	return out
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	if len(s) > 60 {
		return s[:60]
	}
	return s
}

// newID returns a random 16-byte hex identifier.
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// newToken returns a fresh API token and its sha256 hash.
func newToken() (token, hash string) {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	token = "ccotel_" + hex.EncodeToString(b)
	hash = hashToken(token)
	return
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
