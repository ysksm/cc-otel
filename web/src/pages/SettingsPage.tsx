import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, type APIKey } from '../lib/api'
import { formatRelative } from '../lib/format'
import { useAsync } from '../hooks/useAsync'
import { EmptyState, ErrorBox, Spinner } from '../components/common'

export function SettingsPage() {
  const { slug = '' } = useParams()
  const { data, loading, error, reload } = useAsync(() => api.listApiKeys(slug), [slug])
  const keys = data?.items ?? []

  const [name, setName] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<Error | undefined>()
  const [newToken, setNewToken] = useState<APIKey | undefined>()

  const onCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed || creating) return
    setCreating(true)
    setCreateError(undefined)
    try {
      const key = await api.createApiKey(slug, { name: trimmed })
      setNewToken(key)
      setName('')
      reload()
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)))
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1>Settings</h1>
        <span className="dim">{slug}</span>
      </div>

      <section className="settings-section">
        <h2>API keys</h2>
        <p className="dim section-desc">
          Use an API key to authenticate the OTLP exporter that sends traces to cc-otel.
        </p>

        <form className="create-key-form" onSubmit={onCreate}>
          <input
            type="text"
            placeholder="Key name (e.g. local-dev)"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <button className="btn btn-primary" type="submit" disabled={creating || !name.trim()}>
            {creating ? 'Creating…' : 'Create key'}
          </button>
        </form>
        {createError && <ErrorBox error={createError} />}

        {newToken?.token && <NewKeyBox apiKey={newToken} onDismiss={() => setNewToken(undefined)} />}

        {loading && <Spinner label="Loading keys…" />}
        {error && <ErrorBox error={error} />}

        {!loading && !error && keys.length === 0 && (
          <EmptyState title="No API keys">
            <p>Create your first key above to start ingesting traces.</p>
          </EmptyState>
        )}

        {!loading && !error && keys.length > 0 && (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Token</th>
                  <th>Created</th>
                  <th>Last used</th>
                </tr>
              </thead>
              <tbody>
                {keys.map((k) => (
                  <tr key={k.id}>
                    <td>{k.name}</td>
                    <td className="mono">{k.tokenPreview}</td>
                    <td title={k.createdAt}>{relativeFromIso(k.createdAt)}</td>
                    <td>
                      {k.lastUsedAt ? (
                        <span title={k.lastUsedAt}>{relativeFromIso(k.lastUsedAt)}</span>
                      ) : (
                        <span className="dim">never</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="settings-section">
        <h2>Send traces</h2>
        <p className="dim section-desc">
          Configure an OTLP/HTTP exporter to point at the cc-otel endpoint with your key as a Bearer
          token.
        </p>
        <ExporterSnippet />
      </section>
    </div>
  )
}

function NewKeyBox({ apiKey, onDismiss }: { apiKey: APIKey; onDismiss: () => void }) {
  const [copied, setCopied] = useState(false)
  const token = apiKey.token ?? ''

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(token)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* clipboard may be unavailable; user can select manually */
    }
  }

  return (
    <div className="new-key-box">
      <div className="new-key-head">
        <strong>Key “{apiKey.name}” created</strong>
        <button className="icon-btn" onClick={onDismiss} aria-label="Dismiss">
          ✕
        </button>
      </div>
      <p className="warn">
        Copy this token now — it will not be shown again.
      </p>
      <div className="token-copy">
        <code className="token-value">{token}</code>
        <button className="btn" onClick={copy}>
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
    </div>
  )
}

const ENDPOINT = 'http://localhost:8080/v1/traces'

function ExporterSnippet() {
  const snippet = `# OTLP/HTTP exporter environment
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=${ENDPOINT}
OTEL_EXPORTER_OTLP_TRACES_HEADERS=Authorization=Bearer <token>
OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http/protobuf`
  return (
    <div className="snippet">
      <pre className="code-block">{snippet}</pre>
      <p className="dim">
        Endpoint: <code>{ENDPOINT}</code> · Header: <code>Authorization: Bearer &lt;token&gt;</code>
      </p>
    </div>
  )
}

function relativeFromIso(iso: string): string {
  const ms = Date.parse(iso)
  if (Number.isNaN(ms)) return iso
  return formatRelative(ms * 1e6)
}
