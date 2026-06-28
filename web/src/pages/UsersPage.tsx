import { useParams } from 'react-router-dom'
import { api } from '../lib/api'
import { formatCost, formatInt } from '../lib/format'
import { parseJsonArray } from '../lib/parse'
import { useAsync } from '../hooks/useAsync'
import {
  EmptyState,
  ErrorBox,
  ModelBadges,
  RelativeTime,
  Spinner,
} from '../components/common'

export function UsersPage() {
  const { slug = '' } = useParams()
  const { data, loading, error } = useAsync(() => api.listUsers(slug), [slug])
  const users = data?.items ?? []

  return (
    <div className="page">
      <div className="page-header">
        <h1>Users</h1>
        <span className="dim">{slug}</span>
      </div>

      {loading && <Spinner label="Loading users…" />}
      {error && <ErrorBox error={error} />}

      {!loading && !error && users.length === 0 && (
        <EmptyState title="No users yet">
          <p>
            Users appear when traces carry a user id or email. Send traces with user attributes to
            populate this view.
          </p>
        </EmptyState>
      )}

      {!loading && !error && users.length > 0 && (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>User</th>
                <th className="num">Traces</th>
                <th className="num">Spans</th>
                <th className="num">Errors</th>
                <th className="num">Tokens</th>
                <th className="num">Cost</th>
                <th>Models</th>
                <th>Last seen</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => {
                const models = parseJsonArray(u.models)
                const key = u.userId || u.userEmail
                return (
                  <tr key={key}>
                    <td>
                      <div className="user-cell">
                        <span className="mono" title={u.userId}>
                          {u.userId || <span className="dim">—</span>}
                        </span>
                        {u.userEmail && (
                          <span className="user-email" title={u.userEmail}>
                            {u.userEmail}
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="num">{formatInt(u.traceCount)}</td>
                    <td className="num">{formatInt(u.spanCount)}</td>
                    <td className="num">
                      {u.errorCount > 0 ? (
                        <span className="error-count">{u.errorCount}</span>
                      ) : (
                        <span className="dim">0</span>
                      )}
                    </td>
                    <td
                      className="num"
                      title={`${formatInt(u.tokensInput)} in / ${formatInt(u.tokensOutput)} out`}
                    >
                      {formatInt(u.tokensInput)}
                      <span className="dim"> / </span>
                      {formatInt(u.tokensOutput)}
                    </td>
                    <td className="num">{formatCost(u.costTotalMicrocents)}</td>
                    <td>
                      <ModelBadges providers={[]} models={models} />
                    </td>
                    <td>
                      <RelativeTime ns={u.lastSeenNs} />
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
