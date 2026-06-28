import { useState } from 'react'
import { api, ApiError } from '../lib/api'
import { useAuth } from '../hooks/useAuth'

export function LoginPage() {
  const { refresh } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | undefined>()

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = email.trim()
    if (!trimmed || !password || submitting) return
    setSubmitting(true)
    setError(undefined)
    try {
      await api.login(trimmed, password)
      // Success: re-fetch /me so the gate re-renders into the app.
      refresh()
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError('invalid email or password')
      } else {
        setError(err instanceof Error ? err.message : String(err))
      }
      setSubmitting(false)
    }
  }

  return (
    <div className="login-screen">
      <form className="login-card" onSubmit={onSubmit}>
        <div className="login-brand">
          <span className="brand-mark">◆</span>
          <span className="brand-name">cc-otel</span>
        </div>
        <p className="login-subtitle dim">Sign in to continue</p>

        <label className="login-field">
          <span className="ann-label">Email</span>
          <input
            type="text"
            autoComplete="username"
            placeholder="you@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            disabled={submitting}
            autoFocus
          />
        </label>

        <label className="login-field">
          <span className="ann-label">Password</span>
          <input
            type="password"
            autoComplete="current-password"
            placeholder="••••••••"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={submitting}
          />
        </label>

        {error && <div className="login-error">{error}</div>}

        <button
          className="btn btn-primary login-submit"
          type="submit"
          disabled={submitting || !email.trim() || !password}
        >
          {submitting ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  )
}
