package store

import (
	"fmt"

	"github.com/ysksm/cc-otel/internal/model"
)

// Analytics returns time-bucketed metrics for a project since sinceNs, bucketed
// by bucketNs, plus range totals and the top models by cost. bucketNs is a
// server-computed integer (safe to inline).
func (s *Store) Analytics(projectID string, sinceNs, bucketNs int64) (model.Analytics, error) {
	if bucketNs <= 0 {
		bucketNs = int64(3_600_000_000_000) // 1h
	}
	out := model.Analytics{BucketNs: bucketNs, SinceNs: sinceNs}

	// Time series.
	seriesQ := fmt.Sprintf(`SELECT
		(start_time_ns // %d) * %d                AS bucket,
		count(DISTINCT trace_id)                  AS traces,
		count(*)                                  AS spans,
		count(*) FILTER (WHERE status_code = 2)   AS errors,
		sum(cost_total_microcents)                AS cost,
		sum(tokens_input)                         AS tin,
		sum(tokens_output)                        AS tout,
		CAST(avg(end_time_ns - start_time_ns) AS BIGINT) AS avg_dur
		FROM spans WHERE project_id = ? AND start_time_ns >= ?
		GROUP BY bucket ORDER BY bucket ASC`, bucketNs, bucketNs)
	rows, err := s.db.Query(seriesQ, projectID, sinceNs)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var p model.AnalyticsPoint
		if err := rows.Scan(&p.BucketStartNs, &p.TraceCount, &p.SpanCount, &p.ErrorCount,
			&p.CostTotalMicrocents, &p.TokensInput, &p.TokensOutput, &p.AvgDurationNs); err != nil {
			return out, err
		}
		out.Series = append(out.Series, p)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	// Totals over the range.
	if err := s.db.QueryRow(`SELECT
		count(DISTINCT trace_id), count(*), count(*) FILTER (WHERE status_code = 2),
		coalesce(sum(cost_total_microcents), 0), coalesce(sum(tokens_input), 0), coalesce(sum(tokens_output), 0)
		FROM spans WHERE project_id = ? AND start_time_ns >= ?`, projectID, sinceNs).
		Scan(&out.Totals.TraceCount, &out.Totals.SpanCount, &out.Totals.ErrorCount,
			&out.Totals.CostTotalMicrocents, &out.Totals.TokensInput, &out.Totals.TokensOutput); err != nil {
		return out, err
	}

	// Top models by cost.
	mrows, err := s.db.Query(`SELECT model, sum(cost_total_microcents) AS cost, count(DISTINCT trace_id) AS traces
		FROM spans WHERE project_id = ? AND start_time_ns >= ? AND model <> ''
		GROUP BY model ORDER BY cost DESC LIMIT 8`, projectID, sinceNs)
	if err != nil {
		return out, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var m model.TopModel
		if err := mrows.Scan(&m.Model, &m.CostTotalMicrocents, &m.TraceCount); err != nil {
			return out, err
		}
		out.TopModels = append(out.TopModels, m)
	}
	return out, mrows.Err()
}
