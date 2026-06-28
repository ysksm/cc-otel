import { Link, useParams } from 'react-router-dom'
import { api } from '../lib/api'
import { formatDuration, formatInt } from '../lib/format'
import { useAsync } from '../hooks/useAsync'
import { EmptyState, ErrorBox, RelativeTime, Spinner } from '../components/common'

export function ToolsPage() {
  const { slug = '' } = useParams()
  const { data, loading, error } = useAsync(() => api.listTools(slug), [slug])
  const tools = data?.items ?? []

  return (
    <div className="page">
      <div className="page-header">
        <h1>Tools</h1>
        <span className="dim">{slug}</span>
      </div>

      {loading && <Spinner label="Loading tools…" />}
      {error && <ErrorBox error={error} />}

      {!loading && !error && tools.length === 0 && (
        <EmptyState title="No tools yet">
          <p>
            Tools appear when traces contain <code>execute_tool</code> spans. Send traces that
            record tool calls to populate this view.
          </p>
        </EmptyState>
      )}

      {!loading && !error && tools.length > 0 && (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Tool</th>
                <th className="num">Calls</th>
                <th className="num">Errors</th>
                <th className="num">Avg duration</th>
                <th>Last seen</th>
              </tr>
            </thead>
            <tbody>
              {tools.map((t) => (
                <tr key={t.toolName}>
                  <td className="cell-name">
                    <Link
                      to={`/projects/${slug}/tools/${encodeURIComponent(t.toolName)}`}
                      className="link"
                      title={t.toolName}
                    >
                      {t.toolName || <span className="dim">(unnamed)</span>}
                    </Link>
                  </td>
                  <td className="num">{formatInt(t.callCount)}</td>
                  <td className="num">
                    {t.errorCount > 0 ? (
                      <span className="error-count">{t.errorCount}</span>
                    ) : (
                      <span className="dim">0</span>
                    )}
                  </td>
                  <td className="num">{formatDuration(t.avgDurationNs)}</td>
                  <td>
                    <RelativeTime ns={t.lastSeenNs} />
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
