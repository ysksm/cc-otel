import { Link, useParams } from 'react-router-dom'
import { api } from '../lib/api'
import { formatInt, formatRelativeIso } from '../lib/format'
import { useAsync } from '../hooks/useAsync'
import { Badge, EmptyState, ErrorBox, Spinner } from '../components/common'

export function SignalsPage() {
  const { slug = '' } = useParams()
  const { data, loading, error } = useAsync(() => api.listSignals(slug), [slug])
  const signals = data?.items ?? []

  return (
    <div className="page">
      <div className="page-header">
        <h1>Signals</h1>
        <span className="dim">{slug}</span>
      </div>

      {loading && <Spinner label="Loading signals…" />}
      {error && <ErrorBox error={error} />}

      {!loading && !error && signals.length === 0 && (
        <EmptyState title="No signals yet">
          <p>
            Signals appear when an enabled monitor matches a trace. Define monitors in{' '}
            <Link to={`/projects/${slug}/monitors`} className="link">
              Monitors
            </Link>
            .
          </p>
        </EmptyState>
      )}

      {!loading && !error && signals.length > 0 && (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Monitor</th>
                <th>Description</th>
                <th className="num">Value</th>
                <th>Trace</th>
              </tr>
            </thead>
            <tbody>
              {signals.map((s) => (
                <tr key={s.id}>
                  <td title={s.createdAt}>{formatRelativeIso(s.createdAt)}</td>
                  <td>
                    <Badge tone="default">{s.monitorName}</Badge>
                  </td>
                  <td>{s.description || <span className="dim">—</span>}</td>
                  <td className="num">{formatInt(s.value)}</td>
                  <td>
                    <Link
                      to={`/projects/${slug}/traces?traceId=${encodeURIComponent(s.traceId)}&tab=trace`}
                      className="mono link"
                      title={s.traceId}
                    >
                      {s.traceId}
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
