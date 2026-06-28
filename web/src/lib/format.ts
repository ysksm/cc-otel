// Units / formatting helpers.
//
// - times: startTimeNs etc. are nanoseconds since epoch. Convert to Date via ns/1e6.
// - durationNs: nanoseconds; format ns -> us -> ms -> s nicely.
// - cost: microcents, 1 USD = 100,000,000 microcents.
// - tokens: plain integers, optionally compact.

export function nsToDate(ns: number): Date {
  return new Date(ns / 1e6)
}

export function nsToMs(ns: number): number {
  return ns / 1e6
}

/** Format an absolute timestamp (nanoseconds since epoch) as a locale string. */
export function formatAbsolute(ns: number): string {
  if (!ns) return '—'
  return nsToDate(ns).toLocaleString()
}

/** Format a relative time like "3m ago" from nanoseconds-since-epoch. */
export function formatRelative(ns: number, now: number = Date.now()): string {
  if (!ns) return '—'
  const ms = ns / 1e6
  let diff = now - ms
  const future = diff < 0
  diff = Math.abs(diff)

  const sec = diff / 1000
  const min = sec / 60
  const hr = min / 60
  const day = hr / 24

  let val: string
  if (sec < 5) return 'just now'
  if (sec < 60) val = `${Math.floor(sec)}s`
  else if (min < 60) val = `${Math.floor(min)}m`
  else if (hr < 24) val = `${Math.floor(hr)}h`
  else if (day < 30) val = `${Math.floor(day)}d`
  else if (day < 365) val = `${Math.floor(day / 30)}mo`
  else val = `${Math.floor(day / 365)}y`

  return future ? `in ${val}` : `${val} ago`
}

/** Format a duration given in nanoseconds. */
export function formatDuration(ns: number): string {
  if (ns == null || Number.isNaN(ns)) return '—'
  if (ns <= 0) return '0ms'
  const us = ns / 1e3
  const ms = ns / 1e6
  const s = ns / 1e9

  if (ns < 1e3) return `${Math.round(ns)}ns`
  if (us < 1000) return `${us < 10 ? us.toFixed(1) : Math.round(us)}us`
  if (ms < 1000) return `${ms < 10 ? ms.toFixed(1) : Math.round(ms)}ms`
  if (s < 60) return `${s.toFixed(2)}s`
  const min = Math.floor(s / 60)
  const rem = Math.round(s % 60)
  return `${min}m ${rem}s`
}

const MICROCENTS_PER_USD = 100_000_000

/** Format microcents as USD. */
export function formatCost(microcents: number): string {
  if (microcents == null || Number.isNaN(microcents)) return '—'
  const usd = microcents / MICROCENTS_PER_USD
  if (usd === 0) return '$0'
  if (usd < 0.01) return `$${usd.toFixed(4)}`
  if (usd < 1) return `$${usd.toFixed(4)}`
  return `$${usd.toFixed(2)}`
}

/** Format a token count, compact for large values (1.2k). */
export function formatTokens(n: number): string {
  if (n == null || Number.isNaN(n)) return '0'
  if (n < 1000) return String(n)
  if (n < 1_000_000) return `${(n / 1000).toFixed(n % 1000 === 0 ? 0 : 1)}k`
  return `${(n / 1_000_000).toFixed(1)}M`
}

/** Plain integer with thousands separators. */
export function formatInt(n: number): string {
  if (n == null || Number.isNaN(n)) return '0'
  return n.toLocaleString()
}

/** Truncate a long id, keeping the head; full value is meant to go in a title attr. */
export function truncateId(id: string, head = 8): string {
  if (!id) return ''
  if (id.length <= head + 2) return id
  return `${id.slice(0, head)}…`
}
