import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, type APIKey } from '../lib/api'
import { formatRelativeIso } from '../lib/format'
import { useAsync } from '../hooks/useAsync'
import { Badge, EmptyState, ErrorBox, Spinner } from '../components/common'

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
                    <td title={k.createdAt}>{formatRelativeIso(k.createdAt)}</td>
                    <td>
                      {k.lastUsedAt ? (
                        <span title={k.lastUsedAt}>{formatRelativeIso(k.lastUsedAt)}</span>
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

      <EvaluationsSection slug={slug} />

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

function EvaluationsSection({ slug }: { slug: string }) {
  const { data, loading, error, reload } = useAsync(() => api.listEvaluations(slug), [slug])
  const evaluations = data?.items ?? []

  const [name, setName] = useState('')
  const [prompt, setPrompt] = useState('')
  const [provider, setProvider] = useState('')
  const [model, setModel] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<Error | undefined>()

  const onCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmedName = name.trim()
    const trimmedPrompt = prompt.trim()
    if (!trimmedName || !trimmedPrompt || creating) return
    setCreating(true)
    setCreateError(undefined)
    try {
      await api.createEvaluation(slug, {
        name: trimmedName,
        prompt: trimmedPrompt,
        provider: provider.trim() || undefined,
        model: model.trim() || undefined,
        enabled,
      })
      setName('')
      setPrompt('')
      setProvider('')
      setModel('')
      setEnabled(true)
      reload()
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)))
    } finally {
      setCreating(false)
    }
  }

  return (
    <section className="settings-section">
      <h2>Evaluations</h2>
      <p className="dim section-desc">
        Evaluations run an LLM-as-judge over a trace and record a score. Configure the default judge
        endpoint and key via the <code>CCOTEL_LLM_PROVIDER</code>, <code>CCOTEL_LLM_BASE_URL</code>,{' '}
        <code>CCOTEL_LLM_API_KEY</code>, and <code>CCOTEL_LLM_MODEL</code> environment variables —
        e.g. point <code>CCOTEL_LLM_BASE_URL</code> at a local Ollama{' '}
        <code>http://localhost:11434/v1</code> to run fully locally. A per-evaluation provider/model
        overrides the env defaults.
      </p>

      <form className="create-eval-form" onSubmit={onCreate}>
        <div className="eval-form-row">
          <label className="eval-field">
            <span className="ann-label">Name</span>
            <input
              type="text"
              placeholder="Helpfulness"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </label>
          <label className="eval-field">
            <span className="ann-label">Provider</span>
            <input
              type="text"
              placeholder="openai"
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
            />
          </label>
          <label className="eval-field">
            <span className="ann-label">Model</span>
            <input
              type="text"
              placeholder="gpt-4o"
              value={model}
              onChange={(e) => setModel(e.target.value)}
            />
          </label>
        </div>
        <label className="eval-field">
          <span className="ann-label">Prompt / instructions</span>
          <textarea
            className="ann-textarea"
            placeholder="Judge whether the assistant's final answer is helpful and correct…"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            rows={4}
          />
        </label>
        <div className="eval-form-footer">
          <label className="ann-toggle">
            <input
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
            />
            <span>Enabled</span>
          </label>
          <button
            className="btn btn-primary"
            type="submit"
            disabled={creating || !name.trim() || !prompt.trim()}
          >
            {creating ? 'Creating…' : 'Create evaluation'}
          </button>
        </div>
      </form>
      {createError && <ErrorBox error={createError} />}

      {loading && <Spinner label="Loading evaluations…" />}
      {error && <ErrorBox error={error} />}

      {!loading && !error && evaluations.length === 0 && (
        <EmptyState title="No evaluations">
          <p>Create an LLM-as-judge evaluation above to score traces automatically.</p>
        </EmptyState>
      )}

      {!loading && !error && evaluations.length > 0 && (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Slug</th>
                <th>Provider / Model</th>
                <th>Enabled</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              {evaluations.map((ev) => (
                <tr key={ev.id}>
                  <td>{ev.name}</td>
                  <td className="mono">{ev.slug}</td>
                  <td>
                    <span className="badge-row">
                      {ev.provider && <Badge tone="provider">{ev.provider}</Badge>}
                      {ev.model && <Badge tone="model">{ev.model}</Badge>}
                      {!ev.provider && !ev.model && <span className="dim">default</span>}
                    </span>
                  </td>
                  <td>
                    {ev.enabled ? (
                      <Badge tone="default">enabled</Badge>
                    ) : (
                      <Badge tone="muted">disabled</Badge>
                    )}
                  </td>
                  <td title={ev.createdAt}>{formatRelativeIso(ev.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
