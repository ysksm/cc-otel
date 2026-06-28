import { useEffect } from 'react'
import { api, type Span, type TraceSummary } from '../lib/api'
import { formatAbsolute, formatCost, formatDuration, formatInt } from '../lib/format'
import { parseJsonArray, parseMessages, parseJson, stringifyUnknown } from '../lib/parse'
import { findConversationSpan, traceWindow } from '../lib/tree'
import { useAsync } from '../hooks/useAsync'
import { Badge, ErrorBox, ModelBadges, Spinner, Stat, MonoId } from './common'
import { MessageThread } from './Messages'
import { SpanDetail } from './SpanDetail'
import { Waterfall } from './Waterfall'

export type DrawerTab = 'spans' | 'conversation' | 'trace'

export function TraceDrawer({
  slug,
  traceId,
  tab,
  selectedSpanId,
  onClose,
  onTabChange,
  onSelectSpan,
}: {
  slug: string
  traceId: string
  tab: DrawerTab
  selectedSpanId: string | undefined
  onClose: () => void
  onTabChange: (tab: DrawerTab) => void
  onSelectSpan: (spanId: string | undefined) => void
}) {
  const { data, loading, error } = useAsync(() => api.getTrace(slug, traceId), [slug, traceId])

  // Close on Escape.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <>
      <div className="drawer-backdrop" onClick={onClose} />
      <div className="drawer" role="dialog" aria-label="Trace detail">
        <div className="drawer-header">
          <div className="drawer-title">
            <span className="drawer-title-label">Trace</span>
            <MonoId value={traceId} head={16} />
          </div>
          <button className="icon-btn" onClick={onClose} aria-label="Close">
            ✕
          </button>
        </div>

        <div className="drawer-tabs">
          <TabBtn active={tab === 'spans'} onClick={() => onTabChange('spans')}>
            Spans
          </TabBtn>
          <TabBtn active={tab === 'conversation'} onClick={() => onTabChange('conversation')}>
            Conversation
          </TabBtn>
          <TabBtn active={tab === 'trace'} onClick={() => onTabChange('trace')}>
            Trace
          </TabBtn>
        </div>

        <div className="drawer-body">
          {loading && <Spinner label="Loading trace…" />}
          {error && <ErrorBox error={error} />}
          {data && tab === 'spans' && (
            <SpansTab
              spans={data.spans}
              trace={data.trace}
              selectedSpanId={selectedSpanId}
              onSelectSpan={onSelectSpan}
            />
          )}
          {data && tab === 'conversation' && <ConversationTab spans={data.spans} />}
          {data && tab === 'trace' && <TraceTab trace={data.trace} spans={data.spans} />}
        </div>
      </div>
    </>
  )
}

function TabBtn({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button className={`tab-btn${active ? ' active' : ''}`} onClick={onClick}>
      {children}
    </button>
  )
}

function SpansTab({
  spans,
  trace,
  selectedSpanId,
  onSelectSpan,
}: {
  spans: Span[]
  trace: TraceSummary
  selectedSpanId: string | undefined
  onSelectSpan: (spanId: string | undefined) => void
}) {
  if (spans.length === 0) {
    return <div className="dim pad">No spans recorded for this trace.</div>
  }
  const computed = traceWindow(spans)
  const winStart = trace.startTimeNs || computed.start
  const winEnd = trace.endTimeNs || computed.end
  const selected = spans.find((s) => s.spanId === selectedSpanId)

  return (
    <div className="spans-tab">
      <Waterfall
        spans={spans}
        winStart={winStart}
        winEnd={winEnd}
        selectedId={selectedSpanId}
        onSelect={(id) => onSelectSpan(id === selectedSpanId ? undefined : id)}
      />
      {selected ? (
        <div className="span-detail-wrap">
          <SpanDetail span={selected} />
        </div>
      ) : (
        <div className="dim pad">Select a span to see its details.</div>
      )}
    </div>
  )
}

function ConversationTab({ spans }: { spans: Span[] }) {
  const span = findConversationSpan(spans)
  if (!span) {
    return <div className="dim pad">No conversation found in this trace.</div>
  }
  const system = parseSystem(span.systemInstructions)
  const input = parseMessages(span.inputMessages)
  const output = parseMessages(span.outputMessages)

  if (!system && input.length === 0 && output.length === 0) {
    return <div className="dim pad">No conversation messages on the latest LLM span.</div>
  }

  return (
    <div className="conversation-tab">
      <MessageThread system={system} messages={[...input, ...output]} />
      <div className="conv-footer">
        {span.model && <Badge tone="model">{span.model}</Badge>}
        <span className="dim">
          {formatInt(span.tokensInput)} in / {formatInt(span.tokensOutput)} out
        </span>
        {span.costTotalMicrocents > 0 && (
          <span className="dim">
            {formatCost(span.costTotalMicrocents)} {span.costIsEstimated && '(est.)'}
          </span>
        )}
      </div>
    </div>
  )
}

function TraceTab({ trace, spans }: { trace: TraceSummary; spans: Span[] }) {
  const providers = parseJsonArray(trace.providers)
  const models = parseJsonArray(trace.models)
  const computed = traceWindow(spans)
  const start = trace.startTimeNs || computed.start
  const duration = trace.durationNs || computed.end - computed.start

  // Aggregate tokens across spans as a fallback / detail.
  const totalTokens =
    trace.tokensInput +
    trace.tokensOutput +
    trace.tokensCacheRead +
    trace.tokensCacheCreate +
    trace.tokensReasoning

  return (
    <div className="trace-tab">
      <div className="stat-grid">
        <Stat label="Start" value={<span title={formatAbsolute(start)}>{formatAbsolute(start)}</span>} />
        <Stat label="Duration" value={formatDuration(duration)} />
        <Stat label="Spans" value={formatInt(trace.spanCount || spans.length)} />
        <Stat
          label="Errors"
          value={
            trace.errorCount > 0 ? (
              <span className="error-count">{trace.errorCount}</span>
            ) : (
              '0'
            )
          }
        />
      </div>

      <div className="detail-section">
        <h4>Providers / Models</h4>
        <ModelBadges providers={providers} models={models} />
      </div>

      <div className="detail-section">
        <h4>Tokens</h4>
        <div className="stat-grid">
          <Stat label="Input" value={formatInt(trace.tokensInput)} />
          <Stat label="Output" value={formatInt(trace.tokensOutput)} />
          <Stat label="Cache read" value={formatInt(trace.tokensCacheRead)} />
          <Stat label="Cache create" value={formatInt(trace.tokensCacheCreate)} />
          <Stat label="Reasoning" value={formatInt(trace.tokensReasoning)} />
          <Stat label="Total" value={formatInt(totalTokens)} />
        </div>
      </div>

      <div className="detail-section">
        <h4>Cost</h4>
        <Stat label="Total" value={formatCost(trace.costTotalMicrocents)} />
      </div>

      <div className="detail-section">
        <h4>Identifiers</h4>
        <div className="kv">
          <div className="kv-row">
            <span className="kv-key">Trace</span>
            <span className="kv-val mono" title={trace.traceId}>
              {trace.traceId}
            </span>
          </div>
          {trace.sessionId && (
            <div className="kv-row">
              <span className="kv-key">Session</span>
              <span className="kv-val mono" title={trace.sessionId}>
                {trace.sessionId}
              </span>
            </div>
          )}
          {trace.rootSpanName && (
            <div className="kv-row">
              <span className="kv-key">Root span</span>
              <span className="kv-val">{trace.rootSpanName}</span>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function parseSystem(s: string | undefined | null): string | undefined {
  if (!s) return undefined
  const v = parseJson<unknown>(s)
  if (v === undefined) return s
  if (typeof v === 'string') return v
  if (Array.isArray(v)) {
    return v.map((x) => (typeof x === 'string' ? x : stringifyUnknown(x))).join('\n')
  }
  return stringifyUnknown(v)
}
