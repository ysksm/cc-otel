import type { Span } from '../lib/api'
import { formatDuration } from '../lib/format'
import { buildSpanTree, flattenTree } from '../lib/tree'

/**
 * Span waterfall. Each row's bar is positioned/sized relative to the trace
 * window [winStart, winEnd]. Indentation encodes tree depth.
 */
export function Waterfall({
  spans,
  winStart,
  winEnd,
  selectedId,
  onSelect,
}: {
  spans: Span[]
  winStart: number
  winEnd: number
  selectedId: string | undefined
  onSelect: (spanId: string) => void
}) {
  const tree = buildSpanTree(spans)
  const rows = flattenTree(tree)
  const total = Math.max(1, winEnd - winStart)

  return (
    <div className="waterfall">
      {rows.map(({ span, depth }) => {
        const offset = ((span.startTimeNs - winStart) / total) * 100
        const width = Math.max(0.5, ((span.endTimeNs - span.startTimeNs) / total) * 100)
        const clampedOffset = Math.min(99.5, Math.max(0, offset))
        const clampedWidth = Math.min(100 - clampedOffset, width)
        const isError = span.statusCode === 2
        const isTool = span.operation === 'execute_tool'
        const selected = span.spanId === selectedId

        return (
          <button
            key={span.spanId}
            className={`wf-row${selected ? ' selected' : ''}`}
            onClick={() => onSelect(span.spanId)}
          >
            <div className="wf-label" style={{ paddingLeft: `${depth * 16}px` }}>
              {isError && <span className="error-dot" title="error" />}
              {isTool && <span className="tool-icon" title="tool call">⚙</span>}
              <span className="wf-name" title={span.name}>
                {span.name || '(unnamed)'}
              </span>
              {span.model && <span className="wf-model" title={`${span.provider} / ${span.model}`}>{span.model}</span>}
            </div>
            <div className="wf-track">
              <div
                className={`wf-bar${isError ? ' bar-error' : ''}${isTool ? ' bar-tool' : ''}`}
                style={{ left: `${clampedOffset}%`, width: `${clampedWidth}%` }}
              />
            </div>
            <div className="wf-dur">{formatDuration(span.endTimeNs - span.startTimeNs)}</div>
          </button>
        )
      })}
    </div>
  )
}
