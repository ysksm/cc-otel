import { useState } from 'react'
import { useParams } from 'react-router-dom'
import type { AnalyticsRange } from '../lib/api'
import { api } from '../lib/api'
import { formatCost, formatInt, nsToDate } from '../lib/format'
import { useAsync } from '../hooks/useAsync'
import { BarChart } from '../components/BarChart'
import { EmptyState, ErrorBox, Spinner } from '../components/common'

const RANGES: { value: AnalyticsRange; label: string }[] = [
  { value: '1h', label: '1h' },
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' },
  { value: '30d', label: '30d' },
]

const ONE_DAY_NS = 24 * 60 * 60 * 1e9

/** Pick a compact x-axis label format based on the bucket width. */
function bucketLabel(ns: number, bucketNs: number): string {
  if (!ns) return ''
  const d = nsToDate(ns)
  if (bucketNs >= ONE_DAY_NS) {
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  }
  return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
}

const microcentsToUsd = (mc: number) => mc / 100_000_000

export function DashboardPage() {
  const { slug = '' } = useParams()
  const [range, setRange] = useState<AnalyticsRange>('24h')
  const { data, loading, error } = useAsync(() => api.getAnalytics(slug, range), [slug, range])

  const totals = data?.totals
  const series = data?.series ?? []
  const topModels = data?.topModels ?? []
  const hasData =
    !!totals &&
    (totals.traceCount > 0 ||
      totals.spanCount > 0 ||
      totals.costTotalMicrocents > 0 ||
      series.some((b) => b.traceCount > 0 || b.spanCount > 0 || b.costTotalMicrocents > 0))

  const errorRate =
    totals && totals.traceCount > 0 ? (totals.errorCount / totals.traceCount) * 100 : 0

  const startLabel = data && series.length > 0 ? bucketLabel(series[0].bucketStartNs, data.bucketNs) : ''
  const endLabel =
    data && series.length > 0
      ? bucketLabel(series[series.length - 1].bucketStartNs, data.bucketNs)
      : ''

  const maxCost = topModels.reduce((m, t) => (t.costTotalMicrocents > m ? t.costTotalMicrocents : m), 0)

  return (
    <div className="page">
      <div className="page-header">
        <h1>Dashboard</h1>
        <span className="dim">{slug}</span>
        <div className="range-selector" role="group" aria-label="Time range">
          {RANGES.map((r) => (
            <button
              key={r.value}
              type="button"
              className={`range-btn${range === r.value ? ' active' : ''}`}
              onClick={() => setRange(r.value)}
              aria-pressed={range === r.value}
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>

      {loading && <Spinner label="Loading analytics…" />}
      {error && <ErrorBox error={error} />}

      {!loading && !error && !hasData && (
        <EmptyState title="No activity in this range">
          <p>
            There is no trace data for the selected time range. Send OpenTelemetry traces to the
            ingest endpoint, or pick a wider range.
          </p>
          <pre className="code-block">{`POST http://localhost:8080/v1/traces
Authorization: Bearer <your-api-key>`}</pre>
        </EmptyState>
      )}

      {!loading && !error && hasData && totals && data && (
        <>
          <div className="stat-grid dash-stats">
            <div className="stat">
              <div className="stat-label">Traces</div>
              <div className="stat-value">{formatInt(totals.traceCount)}</div>
            </div>
            <div className="stat">
              <div className="stat-label">Errors</div>
              <div className={`stat-value${totals.errorCount > 0 ? ' stat-bad' : ''}`}>
                {formatInt(totals.errorCount)}
                {totals.traceCount > 0 && (
                  <span className="stat-sub"> {errorRate.toFixed(1)}%</span>
                )}
              </div>
            </div>
            <div className="stat">
              <div className="stat-label">Cost</div>
              <div className="stat-value">{formatCost(totals.costTotalMicrocents)}</div>
            </div>
            <div className="stat">
              <div className="stat-label">Tokens (in / out)</div>
              <div className="stat-value">
                {formatInt(totals.tokensInput)}
                <span className="dim"> / </span>
                {formatInt(totals.tokensOutput)}
              </div>
            </div>
          </div>

          <div className="chart-grid">
            <div className="chart-card">
              <div className="chart-title">Traces per bucket</div>
              <BarChart
                values={series.map((b) => b.traceCount)}
                startLabel={startLabel}
                endLabel={endLabel}
                formatMax={(v) => formatInt(v)}
                label="Traces per bucket"
              />
            </div>

            <div className="chart-card">
              <div className="chart-title">Cost per bucket</div>
              <BarChart
                values={series.map((b) => microcentsToUsd(b.costTotalMicrocents))}
                startLabel={startLabel}
                endLabel={endLabel}
                color="var(--green)"
                area
                formatMax={() => formatCost(maxBucketCost(series))}
                label="Cost per bucket"
              />
            </div>

            <div className="chart-card">
              <div className="chart-title">Errors per bucket</div>
              <BarChart
                values={series.map((b) => b.errorCount)}
                startLabel={startLabel}
                endLabel={endLabel}
                color="var(--red)"
                formatMax={(v) => formatInt(v)}
                label="Errors per bucket"
              />
            </div>
          </div>

          <div className="dash-section">
            <h2 className="subhead">Top models</h2>
            {topModels.length === 0 ? (
              <p className="dim">No model cost recorded in this range.</p>
            ) : (
              <div className="model-list">
                {topModels.map((m) => (
                  <div key={m.model} className="model-row">
                    <div className="model-head">
                      <span className="model-name mono" title={m.model}>
                        {m.model || <span className="dim">(unknown)</span>}
                      </span>
                      <span className="model-meta">
                        <span className="model-cost">{formatCost(m.costTotalMicrocents)}</span>
                        <span className="dim"> · {formatInt(m.traceCount)} traces</span>
                      </span>
                    </div>
                    <div className="model-bar-track">
                      <div
                        className="model-bar-fill"
                        style={{
                          width:
                            maxCost > 0
                              ? `${Math.max((m.costTotalMicrocents / maxCost) * 100, 1)}%`
                              : '0%',
                        }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}

function maxBucketCost(series: { costTotalMicrocents: number }[]): number {
  return series.reduce((m, b) => (b.costTotalMicrocents > m ? b.costTotalMicrocents : m), 0)
}
