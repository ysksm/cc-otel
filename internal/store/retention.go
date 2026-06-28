package store

import (
	"os"

	"github.com/ysksm/cc-otel/internal/model"
)

// StorageStats returns counts, the span time range and the DuckDB file size.
func (s *Store) StorageStats() (model.StorageStats, error) {
	var st model.StorageStats
	if err := s.db.QueryRow(`SELECT count(*), count(DISTINCT trace_id),
		coalesce(min(start_time_ns), 0), coalesce(max(start_time_ns), 0) FROM spans`).
		Scan(&st.Spans, &st.Traces, &st.OldestNs, &st.NewestNs); err != nil {
		return st, err
	}
	_ = s.db.QueryRow(`SELECT count(*) FROM span_embeddings`).Scan(&st.Embeddings)
	_ = s.db.QueryRow(`SELECT count(*) FROM scores`).Scan(&st.Scores)
	_ = s.db.QueryRow(`SELECT count(*) FROM signals`).Scan(&st.Signals)
	if s.dbPath != "" && s.dbPath != ":memory:" {
		if fi, err := os.Stat(s.dbPath); err == nil {
			st.DBSizeBytes = fi.Size()
		}
	}
	return st, nil
}

// DeleteOlderThan removes spans started before cutoffNs and any embeddings,
// scores and signals orphaned as a result. Returns the number of spans deleted.
// The count is taken via SELECT (the purego driver does not report RowsAffected
// for DELETE).
func (s *Store) DeleteOlderThan(cutoffNs int64) (int64, error) {
	var n int64
	if err := s.db.QueryRow(`SELECT count(*) FROM spans WHERE start_time_ns < ?`, cutoffNs).Scan(&n); err != nil {
		return 0, err
	}
	if _, err := s.db.Exec(`DELETE FROM spans WHERE start_time_ns < ?`, cutoffNs); err != nil {
		return 0, err
	}
	if err := s.cleanupOrphans(); err != nil {
		return n, err
	}
	return n, nil
}

// PurgeTelemetry deletes all spans, embeddings, scores and signals (config such
// as projects, monitors, evaluations and accounts is kept). Returns spans deleted.
func (s *Store) PurgeTelemetry() (int64, error) {
	var spans int64
	_ = s.db.QueryRow(`SELECT count(*) FROM spans`).Scan(&spans)
	for _, t := range []string{"spans", "span_embeddings", "scores", "signals"} {
		if _, err := s.db.Exec(`DELETE FROM ` + t); err != nil {
			return 0, err
		}
	}
	return spans, nil
}

// cleanupOrphans removes per-trace rows whose trace no longer has spans.
func (s *Store) cleanupOrphans() error {
	stmts := []string{
		`DELETE FROM span_embeddings WHERE trace_id NOT IN (SELECT DISTINCT trace_id FROM spans)`,
		`DELETE FROM scores  WHERE trace_id <> '' AND trace_id NOT IN (SELECT DISTINCT trace_id FROM spans)`,
		`DELETE FROM signals WHERE trace_id <> '' AND trace_id NOT IN (SELECT DISTINCT trace_id FROM spans)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
