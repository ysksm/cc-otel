package store

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/ysksm/cc-otel/internal/model"
)

// SpansNeedingEmbedding returns spans with message content that are not yet
// embedded (newest first), up to limit.
func (s *Store) SpansNeedingEmbedding(projectID string, limit int) ([]model.Span, error) {
	if limit <= 0 || limit > 2000 {
		limit = 200
	}
	q := "SELECT " + spanSelect + ` FROM spans
		WHERE project_id = ?
		AND (CAST(input_messages AS VARCHAR) <> '[]' OR CAST(output_messages AS VARCHAR) <> '[]')
		AND span_id NOT IN (SELECT span_id FROM span_embeddings WHERE project_id = ?)
		ORDER BY start_time_ns DESC LIMIT ?`
	rows, err := s.db.Query(q, projectID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Span
	for rows.Next() {
		sp, err := scanSpan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

// UpsertEmbedding stores (or replaces) the embedding for a span.
func (s *Store) UpsertEmbedding(projectID, traceID, spanID, embModel string, vec []float32, name, text string) error {
	b, err := json.Marshal(vec)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT OR REPLACE INTO span_embeddings
		(project_id, trace_id, span_id, model, dim, vec, name, text)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		projectID, traceID, spanID, embModel, len(vec), string(b), name, text)
	return err
}

// CountEmbeddings returns how many spans are embedded in a project.
func (s *Store) CountEmbeddings(projectID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT count(*) FROM span_embeddings WHERE project_id = ?`, projectID).Scan(&n)
	return n, err
}

// CountEmbeddableSpans returns how many spans have message content in a project.
func (s *Store) CountEmbeddableSpans(projectID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT count(*) FROM spans WHERE project_id = ?
		AND (CAST(input_messages AS VARCHAR) <> '[]' OR CAST(output_messages AS VARCHAR) <> '[]')`, projectID).Scan(&n)
	return n, err
}

// SearchSemantic ranks embedded spans by cosine similarity to query and returns
// the top results. Similarity is computed in Go over the project's vectors.
func (s *Store) SearchSemantic(projectID string, query []float32, limit int) ([]model.SearchHit, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.Query(`SELECT trace_id, span_id, name, text, CAST(vec AS VARCHAR)
		FROM span_embeddings WHERE project_id = ? LIMIT 50000`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []model.SearchHit
	for rows.Next() {
		var traceID, spanID, name, text, vecJSON string
		if err := rows.Scan(&traceID, &spanID, &name, &text, &vecJSON); err != nil {
			return nil, err
		}
		var vec []float32
		if json.Unmarshal([]byte(vecJSON), &vec) != nil {
			continue
		}
		score := cosine(query, vec)
		hits = append(hits, model.SearchHit{
			TraceID: traceID, SpanID: spanID, Name: name, Snippet: snippet(text, 200), Score: score,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

// cosine returns the cosine similarity of two equal-length vectors (0 if invalid).
func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func snippet(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
