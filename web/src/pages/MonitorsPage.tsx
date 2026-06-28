import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, type Monitor, type MonitorConditionType } from '../lib/api'
import { formatRelativeIso } from '../lib/format'
import { useAsync } from '../hooks/useAsync'
import { Badge, EmptyState, ErrorBox, Spinner } from '../components/common'

const CONDITION_LABELS: Record<MonitorConditionType, string> = {
  trace_error: 'Trace has errors',
  cost_gt: 'Cost over threshold (USD)',
  latency_gt: 'Latency over threshold (ms)',
  tokens_gt: 'Tokens over threshold',
  model_used: 'Model used',
}

const CONDITION_ORDER: MonitorConditionType[] = [
  'trace_error',
  'cost_gt',
  'latency_gt',
  'tokens_gt',
  'model_used',
]

/** Unit hint shown next to the numeric threshold input. */
function thresholdUnit(type: MonitorConditionType): string {
  switch (type) {
    case 'cost_gt':
      return 'USD'
    case 'latency_gt':
      return 'ms'
    case 'tokens_gt':
      return 'tokens'
    default:
      return ''
  }
}

function usesThreshold(type: MonitorConditionType): boolean {
  return type === 'cost_gt' || type === 'latency_gt' || type === 'tokens_gt'
}

/** Human-readable summary of a monitor's condition. */
function conditionSummary(m: Monitor): string {
  switch (m.conditionType) {
    case 'trace_error':
      return 'Has error spans'
    case 'cost_gt':
      return `Cost > $${m.threshold}`
    case 'latency_gt':
      return `Latency > ${m.threshold}ms`
    case 'tokens_gt':
      return `Tokens > ${m.threshold}`
    case 'model_used':
      return `Model = ${m.valueStr}`
    default:
      return m.conditionType
  }
}

export function MonitorsPage() {
  const { slug = '' } = useParams()
  const { data, loading, error, reload } = useAsync(() => api.listMonitors(slug), [slug])
  const monitors = data?.items ?? []

  const [name, setName] = useState('')
  const [conditionType, setConditionType] = useState<MonitorConditionType>('trace_error')
  const [threshold, setThreshold] = useState('')
  const [valueStr, setValueStr] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<Error | undefined>()

  const [running, setRunning] = useState(false)
  const [runError, setRunError] = useState<Error | undefined>()
  const [runNote, setRunNote] = useState<string | undefined>()

  const needsThreshold = usesThreshold(conditionType)
  const needsModel = conditionType === 'model_used'
  const formValid =
    name.trim() !== '' &&
    (!needsThreshold || threshold.trim() !== '') &&
    (!needsModel || valueStr.trim() !== '')

  const onCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!formValid || creating) return
    setCreating(true)
    setCreateError(undefined)
    try {
      await api.createMonitor(slug, {
        name: name.trim(),
        conditionType,
        threshold: needsThreshold ? Number(threshold) : undefined,
        valueStr: needsModel ? valueStr.trim() : undefined,
        enabled,
      })
      setName('')
      setConditionType('trace_error')
      setThreshold('')
      setValueStr('')
      setEnabled(true)
      reload()
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)))
    } finally {
      setCreating(false)
    }
  }

  const onRun = async () => {
    if (running) return
    setRunning(true)
    setRunError(undefined)
    setRunNote(undefined)
    try {
      const res = await api.evaluateMonitors(slug)
      setRunNote(`Created ${res.created} signal${res.created === 1 ? '' : 's'}`)
    } catch (err) {
      setRunError(err instanceof Error ? err : new Error(String(err)))
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1>Monitors</h1>
        <span className="dim">{slug}</span>
      </div>

      <section className="settings-section">
        <h2>Create monitor</h2>
        <p className="dim section-desc">
          Monitors watch incoming traces and emit a signal when their condition matches. Use{' '}
          <strong>Run on existing traces</strong> to backfill signals over traces you already have.
        </p>

        <form className="create-eval-form" onSubmit={onCreate}>
          <div className="eval-form-row">
            <label className="eval-field">
              <span className="ann-label">Name</span>
              <input
                type="text"
                placeholder="Expensive traces"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
            <label className="eval-field">
              <span className="ann-label">Condition</span>
              <select
                value={conditionType}
                onChange={(e) => setConditionType(e.target.value as MonitorConditionType)}
              >
                {CONDITION_ORDER.map((t) => (
                  <option key={t} value={t}>
                    {CONDITION_LABELS[t]}
                  </option>
                ))}
              </select>
            </label>
            {needsThreshold && (
              <label className="eval-field">
                <span className="ann-label">Threshold ({thresholdUnit(conditionType)})</span>
                <input
                  type="number"
                  step="any"
                  min="0"
                  placeholder={conditionType === 'cost_gt' ? '0.01' : conditionType === 'latency_gt' ? '2000' : '5000'}
                  value={threshold}
                  onChange={(e) => setThreshold(e.target.value)}
                />
              </label>
            )}
            {needsModel && (
              <label className="eval-field">
                <span className="ann-label">Model</span>
                <input
                  type="text"
                  placeholder="gpt-4o"
                  value={valueStr}
                  onChange={(e) => setValueStr(e.target.value)}
                />
              </label>
            )}
          </div>
          <div className="eval-form-footer">
            <label className="ann-toggle">
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
              />
              <span>Enabled</span>
            </label>
            <button className="btn btn-primary" type="submit" disabled={creating || !formValid}>
              {creating ? 'Creating…' : 'Create monitor'}
            </button>
          </div>
        </form>
        {createError && <ErrorBox error={createError} />}
      </section>

      <section className="settings-section">
        <div className="monitors-list-head">
          <h2>Monitors</h2>
          <div className="monitors-run">
            {runNote && <span className="monitors-run-note">{runNote}</span>}
            <button className="btn" type="button" onClick={onRun} disabled={running}>
              {running ? 'Running…' : 'Run on existing traces'}
            </button>
          </div>
        </div>
        {runError && <ErrorBox error={runError} />}

        {loading && <Spinner label="Loading monitors…" />}
        {error && <ErrorBox error={error} />}

        {!loading && !error && monitors.length === 0 && (
          <EmptyState title="No monitors yet">
            <p>Create a monitor above to start emitting signals when traces match a condition.</p>
          </EmptyState>
        )}

        {!loading && !error && monitors.length > 0 && (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Condition</th>
                  <th>State</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                {monitors.map((m) => (
                  <tr key={m.id}>
                    <td>{m.name}</td>
                    <td>{conditionSummary(m)}</td>
                    <td>
                      {m.enabled ? (
                        <Badge tone="default">enabled</Badge>
                      ) : (
                        <Badge tone="muted">disabled</Badge>
                      )}
                    </td>
                    <td title={m.createdAt}>{formatRelativeIso(m.createdAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}
