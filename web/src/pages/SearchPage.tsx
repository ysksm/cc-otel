import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, ApiError, type SearchHit } from '../lib/api'
import { formatInt } from '../lib/format'
import { useAsync } from '../hooks/useAsync'
import { EmptyState, ErrorBox, MonoId, Spinner } from '../components/common'

// Cap the incremental indexing loop so a perpetually un-indexed corpus can never
// spin forever; each call embeds up to 200 spans, so 100 iterations covers a lot.
const MAX_INDEX_ITERATIONS = 100

export function SearchPage() {
  const { slug = '' } = useParams()
  const { data, loading, error, reload } = useAsync(() => api.searchStatus(slug), [slug])

  return (
    <div className="page">
      <div className="page-header">
        <h1>Search</h1>
        <span className="dim">{slug}</span>
      </div>

      {loading && <Spinner label="Loading search…" />}
      {error && <ErrorBox error={error} />}

      {!loading && !error && data && !data.configured && (
        <EmptyState title="Semantic search is not configured">
          <p>
            Semantic search embeds your spans with an embedding model so you can find traces by
            meaning rather than exact text. To enable it, set an embedding model via environment
            variables:
          </p>
          <p>
            Set <code>CCOTEL_EMBED_MODEL</code> (and optionally <code>CCOTEL_EMBED_BASE_URL</code>,
            e.g. a local Ollama <code>http://localhost:11434/v1</code>, and{' '}
            <code>CCOTEL_EMBED_API_KEY</code>), then restart cc-otel.
          </p>
        </EmptyState>
      )}

      {!loading && !error && data && data.configured && (
        <SearchPanel
          slug={slug}
          indexed={data.indexed}
          total={data.total}
          model={data.model}
          onReloadStatus={reload}
        />
      )}
    </div>
  )
}

function SearchPanel({
  slug,
  indexed,
  total,
  model,
  onReloadStatus,
}: {
  slug: string
  indexed: number
  total: number
  model: string
  onReloadStatus: () => void
}) {
  const [indexing, setIndexing] = useState(false)
  const [indexProgress, setIndexProgress] = useState(0)
  const [indexError, setIndexError] = useState<Error | undefined>()

  const [query, setQuery] = useState('')
  const [searching, setSearching] = useState(false)
  const [searchError, setSearchError] = useState<Error | undefined>()
  const [hits, setHits] = useState<SearchHit[] | undefined>()

  const onBuildIndex = async () => {
    if (indexing) return
    setIndexing(true)
    setIndexProgress(0)
    setIndexError(undefined)
    try {
      let accumulated = 0
      for (let i = 0; i < MAX_INDEX_ITERATIONS; i++) {
        const res = await api.searchIndex(slug)
        accumulated += res.indexed
        setIndexProgress(accumulated)
        if (res.done || res.indexed === 0) break
      }
    } catch (err) {
      setIndexError(err instanceof Error ? err : new Error(String(err)))
    } finally {
      setIndexing(false)
      onReloadStatus()
    }
  }

  const onSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = query.trim()
    if (!trimmed || searching) return
    setSearching(true)
    setSearchError(undefined)
    try {
      const res = await api.searchSemantic(slug, trimmed)
      setHits(res.hits)
    } catch (err) {
      // A 400 means embeddings are not configured / nothing is indexed yet —
      // surface the backend's message rather than the verbose ApiError prefix.
      if (err instanceof ApiError && err.status === 400) {
        setSearchError(new Error(err.body || 'Semantic search is unavailable.'))
      } else {
        setSearchError(err instanceof Error ? err : new Error(String(err)))
      }
      setHits(undefined)
    } finally {
      setSearching(false)
    }
  }

  return (
    <>
      <section className="settings-section">
        <h2>Index</h2>
        <p className="dim section-desc">
          Spans are embedded incrementally. Build the index to embed any spans that have not been
          indexed yet, then search by meaning below.
        </p>
        <div className="index-status">
          <span>
            Indexed {formatInt(indexed)} / {formatInt(total)} spans (model:{' '}
            <code>{model || 'unknown'}</code>)
          </span>
          <button className="btn" type="button" onClick={onBuildIndex} disabled={indexing}>
            {indexing ? `Indexing… (${formatInt(indexProgress)})` : 'Build index'}
          </button>
        </div>
        {indexError && <ErrorBox error={indexError} />}
      </section>

      <section className="settings-section">
        <h2>Semantic search</h2>
        <form className="search-box" onSubmit={onSearch}>
          <input
            type="text"
            className="search-input"
            placeholder="Describe what you're looking for…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <button className="btn btn-primary" type="submit" disabled={searching || !query.trim()}>
            {searching ? 'Searching…' : 'Search'}
          </button>
        </form>

        {searching && <Spinner label="Searching…" />}
        {searchError && <ErrorBox error={searchError} />}

        {!searching && !searchError && hits && hits.length === 0 && (
          <EmptyState title="No results">
            <p>No spans matched that query. Try different wording or build the index first.</p>
          </EmptyState>
        )}

        {!searching && !searchError && hits && hits.length > 0 && (
          <div className="search-results">
            {hits.map((hit) => (
              <SearchResult key={`${hit.traceId}-${hit.spanId}`} slug={slug} hit={hit} />
            ))}
          </div>
        )}
      </section>
    </>
  )
}

function SearchResult({ slug, hit }: { slug: string; hit: SearchHit }) {
  const to = `/projects/${slug}/traces?traceId=${encodeURIComponent(hit.traceId)}&tab=spans&spanId=${encodeURIComponent(hit.spanId)}`
  const pct = Math.round(hit.score * 100)
  return (
    <Link to={to} className="search-result row-link">
      <div className="search-result-head">
        <span className="search-result-name">
          {hit.name || <span className="dim">(unnamed)</span>}
        </span>
        <span className="search-score" title={hit.score.toFixed(2)}>
          {pct}%
        </span>
      </div>
      {hit.snippet && <div className="search-snippet">{hit.snippet}</div>}
      <div className="search-result-meta">
        <MonoId value={hit.traceId} />
      </div>
    </Link>
  )
}
