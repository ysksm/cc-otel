import type { ReactNode } from 'react'
import { formatAbsolute, formatRelative, truncateId } from '../lib/format'

export function Spinner({ label }: { label?: string }) {
  return (
    <div className="spinner">
      <div className="spinner-dot" />
      <span>{label ?? 'Loading…'}</span>
    </div>
  )
}

export function ErrorBox({ error }: { error: Error }) {
  return (
    <div className="error-box">
      <strong>Something went wrong</strong>
      <div className="error-msg">{error.message}</div>
    </div>
  )
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="empty-state">
      <div className="empty-title">{title}</div>
      {children && <div className="empty-body">{children}</div>}
    </div>
  )
}

/** Monospace, truncated id with a full-value tooltip. */
export function MonoId({ value, head = 8 }: { value: string; head?: number }) {
  if (!value) return <span className="mono dim">—</span>
  return (
    <span className="mono" title={value}>
      {truncateId(value, head)}
    </span>
  )
}

export function RelativeTime({ ns }: { ns: number }) {
  if (!ns) return <span className="dim">—</span>
  return <span title={formatAbsolute(ns)}>{formatRelative(ns)}</span>
}

export function Badge({
  children,
  tone = 'default',
  title,
}: {
  children: ReactNode
  tone?: 'default' | 'provider' | 'model' | 'error' | 'tool' | 'muted'
  title?: string
}) {
  return (
    <span className={`badge badge-${tone}`} title={title}>
      {children}
    </span>
  )
}

/** Render a list of provider/model strings as badges. */
export function ModelBadges({
  providers,
  models,
}: {
  providers: string[]
  models: string[]
}) {
  if (providers.length === 0 && models.length === 0) {
    return <span className="dim">—</span>
  }
  return (
    <span className="badge-row">
      {providers.map((p) => (
        <Badge key={`p-${p}`} tone="provider">
          {p}
        </Badge>
      ))}
      {models.map((m) => (
        <Badge key={`m-${m}`} tone="model">
          {m}
        </Badge>
      ))}
    </span>
  )
}

export function ErrorDot({ count }: { count: number }) {
  if (!count) return <span className="dim">0</span>
  return (
    <span className="error-count" title={`${count} error${count === 1 ? '' : 's'}`}>
      {count}
    </span>
  )
}

export function Stat({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="stat">
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
    </div>
  )
}
