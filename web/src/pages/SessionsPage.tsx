import { useParams } from 'react-router-dom'
import { api } from '../lib/api'
import { formatCost, formatDuration, formatInt, formatTokens } from '../lib/format'
import { parseJsonArray } from '../lib/parse'
import { useAsync } from '../hooks/useAsync'
import {
  EmptyState,
  ErrorBox,
  ModelBadges,
  MonoId,
  RelativeTime,
  Spinner,
} from '../components/common'

export function SessionsPage() {
  const { slug = '' } = useParams()
  const { data, loading, error } = useAsync(() => api.listSessions(slug, { limit: 50 }), [slug])
  const sessions = data?.items ?? []

  return (
    <div className="page">
      <div className="page-header">
        <h1>Sessions</h1>
        <span className="dim">{slug}</span>
      </div>

      {loading && <Spinner label="Loading sessions…" />}
      {error && <ErrorBox error={error} />}

      {!loading && !error && sessions.length === 0 && (
        <EmptyState title="No sessions yet">
          <p>Sessions group related traces by session id. Send traces with a session id to populate this view.</p>
        </EmptyState>
      )}

      {!loading && !error && sessions.length > 0 && (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Session</th>
                <th>Started</th>
                <th className="num">Traces</th>
                <th className="num">Spans</th>
                <th className="num">Errors</th>
                <th>Model</th>
                <th className="num">Tokens</th>
                <th className="num">Cost</th>
                <th className="num">Duration</th>
                <th>User</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map((s) => {
                const providers = parseJsonArray(s.providers)
                const models = parseJsonArray(s.models)
                return (
                  <tr key={s.sessionId}>
                    <td>
                      <MonoId value={s.sessionId} head={12} />
                    </td>
                    <td>
                      <RelativeTime ns={s.startTimeNs} />
                    </td>
                    <td className="num">{formatInt(s.traceCount)}</td>
                    <td className="num">{formatInt(s.spanCount)}</td>
                    <td className="num">
                      {s.errorCount > 0 ? (
                        <span className="error-count">{s.errorCount}</span>
                      ) : (
                        <span className="dim">0</span>
                      )}
                    </td>
                    <td>
                      <ModelBadges providers={providers} models={models} />
                    </td>
                    <td
                      className="num"
                      title={`${formatInt(s.tokensInput)} in / ${formatInt(s.tokensOutput)} out`}
                    >
                      {formatTokens(s.tokensInput)}
                      <span className="dim"> / </span>
                      {formatTokens(s.tokensOutput)}
                    </td>
                    <td className="num">{formatCost(s.costTotalMicrocents)}</td>
                    <td className="num">{formatDuration(s.durationNs)}</td>
                    <td className="cell-name" title={s.userId}>
                      {s.userId || <span className="dim">—</span>}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
