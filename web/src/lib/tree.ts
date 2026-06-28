import type { Span } from './api'
import { parseMessages } from './parse'

export interface SpanNode {
  span: Span
  depth: number
  children: SpanNode[]
}

/**
 * Build a span tree from a flat list of spans using parentSpanId.
 * Roots are spans whose parentSpanId is "" / missing, OR whose parent id is not
 * present among the span ids (orphans are promoted to roots). Children are
 * ordered by startTimeNs.
 */
export function buildSpanTree(spans: Span[]): SpanNode[] {
  const byId = new Map<string, Span>()
  for (const s of spans) byId.set(s.spanId, s)

  const childrenOf = new Map<string, Span[]>()
  const roots: Span[] = []

  for (const s of spans) {
    const parent = s.parentSpanId
    if (!parent || !byId.has(parent)) {
      roots.push(s)
    } else {
      const arr = childrenOf.get(parent) ?? []
      arr.push(s)
      childrenOf.set(parent, arr)
    }
  }

  const sortByStart = (a: Span, b: Span) => a.startTimeNs - b.startTimeNs

  const build = (span: Span, depth: number, seen: Set<string>): SpanNode => {
    seen.add(span.spanId)
    const kids = (childrenOf.get(span.spanId) ?? [])
      .slice()
      .sort(sortByStart)
      .filter((k) => !seen.has(k.spanId)) // guard against cycles
      .map((k) => build(k, depth + 1, seen))
    return { span, depth, children: kids }
  }

  const seen = new Set<string>()
  return roots
    .slice()
    .sort(sortByStart)
    .map((r) => build(r, 0, seen))
}

/** Flatten a span tree into a depth-first ordered list (for rendering rows). */
export function flattenTree(nodes: SpanNode[]): SpanNode[] {
  const out: SpanNode[] = []
  const walk = (n: SpanNode) => {
    out.push(n)
    for (const c of n.children) walk(c)
  }
  for (const n of nodes) walk(n)
  return out
}

/** Compute the [min start, max end] window over spans, in ns. */
export function traceWindow(spans: Span[]): { start: number; end: number } {
  if (spans.length === 0) return { start: 0, end: 0 }
  let start = Infinity
  let end = -Infinity
  for (const s of spans) {
    if (s.startTimeNs && s.startTimeNs < start) start = s.startTimeNs
    if (s.endTimeNs && s.endTimeNs > end) end = s.endTimeNs
  }
  if (!Number.isFinite(start)) start = 0
  if (!Number.isFinite(end)) end = start
  return { start, end }
}

/** Find the last span with non-empty outputMessages, or the last chat/LLM span. */
export function findConversationSpan(spans: Span[]): Span | undefined {
  const sorted = spans.slice().sort((a, b) => a.startTimeNs - b.startTimeNs)
  // Prefer the last span that actually produced output messages. Parse the JSON
  // so an empty array ("[]") is not mistaken for real content.
  for (let i = sorted.length - 1; i >= 0; i--) {
    if (parseMessages(sorted[i].outputMessages).length > 0) return sorted[i]
  }
  // Fall back to the last span that looks like an LLM/chat call.
  for (let i = sorted.length - 1; i >= 0; i--) {
    const s = sorted[i]
    if (parseMessages(s.inputMessages).length > 0 || s.operation === 'chat' || s.model) return s
  }
  return sorted[sorted.length - 1]
}
