import { Link, useParams } from 'react-router-dom'
import { api, type Span } from '../lib/api'
import { formatDuration, formatInt } from '../lib/format'
import { useAsync } from '../hooks/useAsync'
import {
  Badge,
  EmptyState,
  ErrorBox,
  MonoId,
  RelativeTime,
  Spinner,
  Stat,
} from '../components/common'

function statusLabel(code: number): { text: string; tone: 'default' | 'error' | 'muted' } {
  if (code === 2) return { text: 'error', tone: 'error' }
  if (code === 1) return { text: 'ok', tone: 'default' }
  return { text: 'unset', tone: 'muted' }
}

export function ToolDetailPage() {
  const { slug = '', toolName: rawToolName = '' } = useParams()
  const toolName = decodeURIComponent(rawToolName)
  const { data, loading, error } = useAsync(
    () => api.getTool(slug, toolName),
    [slug, toolName],
  )

  const tool = data?.tool
  const calls = data?.calls ?? []

  return (
    <div className="page">
      <div className="page-header">
        <h1>{toolName || 'Tool'}</h1>
        <Link to={`/projects/${slug}/tools`} className="link">
          ← Tools
        </Link>
      </div>

      {loading && <Spinner label="Loading tool…" />}
      {error && <ErrorBox error={error} />}

      {!loading && !error && tool && (
        <>
          <div className="stat-grid tool-summary">
            <Stat label="Calls" value={formatInt(tool.callCount)} />
            <Stat
              label="Errors"
              value={
                tool.errorCount > 0 ? (
                  <span className="error-count">{tool.errorCount}</span>
                ) : (
                  <span className="dim">0</span>
                )
              }
            />
            <Stat label="Avg duration" value={formatDuration(tool.avgDurationNs)} />
            <Stat label="Last seen" value={<RelativeTime ns={tool.lastSeenNs} />} />
          </div>

          <h2 className="subhead">Recent calls</h2>

          {calls.length === 0 ? (
            <EmptyState title="No recent calls">
              <p>No spans were found for this tool.</p>
            </EmptyState>
          ) : (
            <div className="table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Span</th>
                    <th>Status</th>
                    <th className="num">Duration</th>
                    <th>Model</th>
                    <th>Trace</th>
                  </tr>
                </thead>
                <tbody>
                  {calls.map((span) => (
                    <CallRow key={span.spanId} slug={slug} span={span} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  )
}

function CallRow({ slug, span }: { slug: string; span: Span }) {
  const status = statusLabel(span.statusCode)
  const to = `/projects/${slug}/traces?traceId=${encodeURIComponent(
    span.traceId,
  )}&tab=spans&spanId=${encodeURIComponent(span.spanId)}`
  const model = span.responseModel || span.model

  return (
    <tr className="clickable">
      <td className="cell-name">
        <Link to={to} className="row-link" title={span.name}>
          {span.name || <span className="dim">(unnamed span)</span>}
        </Link>
      </td>
      <td>
        <Badge tone={status.tone}>{status.text}</Badge>
      </td>
      <td className="num">{formatDuration(span.endTimeNs - span.startTimeNs)}</td>
      <td>
        {model || span.provider ? (
          <span className="badge-row">
            {span.provider && <Badge tone="provider">{span.provider}</Badge>}
            {model && <Badge tone="model">{model}</Badge>}
          </span>
        ) : (
          <span className="dim">—</span>
        )}
      </td>
      <td>
        <Link to={to} className="link">
          <MonoId value={span.traceId} head={10} />
        </Link>
      </td>
    </tr>
  )
}
