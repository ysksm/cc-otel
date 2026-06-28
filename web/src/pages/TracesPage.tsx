import { useCallback, useEffect, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { api, type TraceSummary } from '../lib/api'
import { formatCost, formatDuration, formatInt, formatTokens } from '../lib/format'
import { parseJsonArray } from '../lib/parse'
import {
  EmptyState,
  ErrorBox,
  ModelBadges,
  RelativeTime,
  ScoreBadge,
  Spinner,
} from '../components/common'
import { TraceDrawer, type DrawerTab } from '../components/TraceDrawer'

const DRAWER_TABS: readonly DrawerTab[] = ['spans', 'conversation', 'trace', 'scores']

function parseTab(value: string | null): DrawerTab {
  return value && (DRAWER_TABS as readonly string[]).includes(value)
    ? (value as DrawerTab)
    : 'spans'
}

interface Filters {
  q: string
  provider: string
  model: string
  errors: boolean
}

const EMPTY_FILTERS: Filters = { q: '', provider: '', model: '', errors: false }

function filtersActive(f: Filters): boolean {
  return f.q !== '' || f.provider !== '' || f.model !== '' || f.errors
}

/** Build the listTraces opts for the active filters (omitting empty values). */
function filterOpts(f: Filters): {
  q?: string
  provider?: string
  model?: string
  errors?: boolean
} {
  return {
    q: f.q.trim() || undefined,
    provider: f.provider.trim() || undefined,
    model: f.model.trim() || undefined,
    errors: f.errors || undefined,
  }
}

/** Build a download URL for exporting the (filtered) traces as CSV or JSON. */
function exportUrl(slug: string, f: Filters, format: 'csv' | 'json'): string {
  const params = new URLSearchParams({ format })
  if (f.q.trim()) params.set('q', f.q.trim())
  if (f.provider.trim()) params.set('provider', f.provider.trim())
  if (f.model.trim()) params.set('model', f.model.trim())
  if (f.errors) params.set('errors', 'true')
  return `/api/projects/${encodeURIComponent(slug)}/traces/export?${params.toString()}`
}

export function TracesPage() {
  const { slug = '' } = useParams()
  const [searchParams, setSearchParams] = useSearchParams()

  const [traces, setTraces] = useState<TraceSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState<Error | undefined>()
  const [done, setDone] = useState(false)

  // Filter bar inputs (controlled). `q` is debounced into `appliedQ` before querying.
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS)
  const [appliedQ, setAppliedQ] = useState('')

  const traceId = searchParams.get('traceId') ?? undefined
  const tab = parseTab(searchParams.get('tab'))
  const spanId = searchParams.get('spanId') ?? undefined

  // Debounce the search text (~300ms) so we don't query on every keystroke.
  useEffect(() => {
    const handle = setTimeout(() => setAppliedQ(filters.q), 300)
    return () => clearTimeout(handle)
  }, [filters.q])

  // The effective filters used for querying (search uses the debounced value).
  const active: Filters = { ...filters, q: appliedQ }

  // Initial load + reload whenever the project or any active filter changes.
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(undefined)
    setTraces([])
    setDone(false)
    api
      .listTraces(slug, { limit: 50, ...filterOpts(active) })
      .then((res) => {
        if (cancelled) return
        setTraces(res.items)
        setDone(res.items.length === 0 || !res.nextCursor)
        setLoading(false)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        setError(err instanceof Error ? err : new Error(String(err)))
        setLoading(false)
      })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slug, appliedQ, filters.provider, filters.model, filters.errors])

  const loadMore = useCallback(() => {
    if (loadingMore || done || traces.length === 0) return
    const last = traces[traces.length - 1]
    setLoadingMore(true)
    api
      .listTraces(slug, { limit: 50, before: last.startTimeNs, ...filterOpts(active) })
      .then((res) => {
        setTraces((prev) => [...prev, ...res.items])
        if (res.items.length === 0 || !res.nextCursor) setDone(true)
        setLoadingMore(false)
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err : new Error(String(err)))
        setLoadingMore(false)
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slug, traces, loadingMore, done, appliedQ, filters.provider, filters.model, filters.errors])

  const clearFilters = () => setFilters(EMPTY_FILTERS)

  const anyFilterActive = filtersActive(filters)

  const openTrace = (id: string) => {
    const next = new URLSearchParams(searchParams)
    next.set('traceId', id)
    if (!next.get('tab')) next.set('tab', 'spans')
    next.delete('spanId')
    setSearchParams(next)
  }

  const closeDrawer = () => {
    const next = new URLSearchParams(searchParams)
    next.delete('traceId')
    next.delete('tab')
    next.delete('spanId')
    setSearchParams(next)
  }

  const setTab = (t: DrawerTab) => {
    const next = new URLSearchParams(searchParams)
    next.set('tab', t)
    setSearchParams(next)
  }

  const setSpan = (id: string | undefined) => {
    const next = new URLSearchParams(searchParams)
    if (id) next.set('spanId', id)
    else next.delete('spanId')
    setSearchParams(next)
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1>Traces</h1>
        <span className="dim">{slug}</span>
      </div>

      <div className="filter-bar">
        <input
          type="text"
          className="filter-search"
          placeholder="Search name, id, or message content"
          value={filters.q}
          onChange={(e) => setFilters((f) => ({ ...f, q: e.target.value }))}
        />
        <input
          type="text"
          className="filter-input"
          placeholder="Provider, e.g. openai"
          value={filters.provider}
          onChange={(e) => setFilters((f) => ({ ...f, provider: e.target.value }))}
        />
        <input
          type="text"
          className="filter-input"
          placeholder="Model, e.g. gpt-4o"
          value={filters.model}
          onChange={(e) => setFilters((f) => ({ ...f, model: e.target.value }))}
        />
        <label className="filter-check">
          <input
            type="checkbox"
            checked={filters.errors}
            onChange={(e) => setFilters((f) => ({ ...f, errors: e.target.checked }))}
          />
          Errors only
        </label>
        {anyFilterActive && (
          <button type="button" className="filter-clear" onClick={clearFilters}>
            Clear
          </button>
        )}
        <div className="export-actions">
          <a className="btn btn-sm" href={exportUrl(slug, active, 'csv')} download>
            Export CSV
          </a>
          <a className="btn btn-sm" href={exportUrl(slug, active, 'json')} download>
            Export JSON
          </a>
        </div>
      </div>

      {loading && <Spinner label="Loading traces…" />}
      {error && <ErrorBox error={error} />}

      {!loading && !error && traces.length === 0 && anyFilterActive && (
        <EmptyState title="No matching traces">
          <p>No traces match the current filters.</p>
          <p>
            Try broadening your search or <span className="link" onClick={clearFilters}>clear the filters</span>.
          </p>
        </EmptyState>
      )}

      {!loading && !error && traces.length === 0 && !anyFilterActive && (
        <EmptyState title="No traces yet">
          <p>
            Send OpenTelemetry traces to <code>POST /v1/traces</code> with an API key, and they will
            appear here.
          </p>
          <p>
            Create a key in <strong>Settings</strong> to get started.
          </p>
        </EmptyState>
      )}

      {!loading && !error && traces.length > 0 && (
        <>
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Root span</th>
                  <th>Model</th>
                  <th>Score</th>
                  <th className="num">Spans</th>
                  <th className="num">Errors</th>
                  <th className="num">Tokens</th>
                  <th className="num">Cost</th>
                  <th className="num">Duration</th>
                </tr>
              </thead>
              <tbody>
                {traces.map((t) => {
                  const providers = parseJsonArray(t.providers)
                  const models = parseJsonArray(t.models)
                  const isActive = t.traceId === traceId
                  return (
                    <tr
                      key={t.traceId}
                      className={`clickable${isActive ? ' active' : ''}`}
                      onClick={() => openTrace(t.traceId)}
                    >
                      <td>
                        <RelativeTime ns={t.startTimeNs} />
                      </td>
                      <td className="cell-name" title={t.rootSpanName}>
                        {t.rootSpanName || <span className="dim">—</span>}
                      </td>
                      <td>
                        <ModelBadges providers={providers} models={models} />
                      </td>
                      <td>
                        <ScoreBadge
                          scoreCount={t.scoreCount}
                          passedCount={t.passedCount}
                          failedCount={t.failedCount}
                          avgScore={t.avgScore}
                        />
                      </td>
                      <td className="num">{formatInt(t.spanCount)}</td>
                      <td className="num">
                        {t.errorCount > 0 ? (
                          <span className="error-count">{t.errorCount}</span>
                        ) : (
                          <span className="dim">0</span>
                        )}
                      </td>
                      <td className="num" title={`${formatInt(t.tokensInput)} in / ${formatInt(t.tokensOutput)} out`}>
                        {formatTokens(t.tokensInput)}
                        <span className="dim"> / </span>
                        {formatTokens(t.tokensOutput)}
                      </td>
                      <td className="num">
                        {formatCost(t.costTotalMicrocents)}
                      </td>
                      <td className="num">{formatDuration(t.durationNs)}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          <div className="load-more">
            {!done ? (
              <button className="btn" onClick={loadMore} disabled={loadingMore}>
                {loadingMore ? 'Loading…' : 'Load more'}
              </button>
            ) : (
              <span className="dim">
                {traces.length} trace{traces.length === 1 ? '' : 's'} • end of list
              </span>
            )}
          </div>
        </>
      )}

      {traceId && (
        <TraceDrawer
          slug={slug}
          traceId={traceId}
          tab={tab}
          selectedSpanId={spanId}
          onClose={closeDrawer}
          onTabChange={setTab}
          onSelectSpan={setSpan}
        />
      )}
    </div>
  )
}
