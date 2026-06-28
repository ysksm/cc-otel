import { useEffect, useState } from 'react'
import { api, type Evaluation, type Score, type Span, type TraceSummary } from '../lib/api'
import {
  formatAbsolute,
  formatCost,
  formatDuration,
  formatInt,
  formatRelativeIso,
  formatScore,
} from '../lib/format'
import { parseJsonArray, parseMessages, parseJson, stringifyUnknown } from '../lib/parse'
import { findConversationSpan, traceWindow } from '../lib/tree'
import { useAsync } from '../hooks/useAsync'
import { Badge, EmptyState, ErrorBox, ModelBadges, Spinner, Stat, MonoId } from './common'
import { MessageThread } from './Messages'
import { SpanDetail } from './SpanDetail'
import { Waterfall } from './Waterfall'

export type DrawerTab = 'spans' | 'conversation' | 'trace' | 'scores'

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
          <TabBtn active={tab === 'scores'} onClick={() => onTabChange('scores')}>
            Scores
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
          {tab === 'scores' && <ScoresTab slug={slug} traceId={traceId} />}
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

function ScoresTab({ slug, traceId }: { slug: string; traceId: string }) {
  const { data, loading, error, reload } = useAsync(
    () => api.listTraceScores(slug, traceId),
    [slug, traceId],
  )
  const evals = useAsync(() => api.listEvaluations(slug), [slug])
  const scores = data?.items ?? []

  return (
    <div className="scores-tab">
      <section className="detail-section">
        <h4>Scores</h4>
        {loading && <Spinner label="Loading scores…" />}
        {error && <ErrorBox error={error} />}
        {!loading && !error && scores.length === 0 && (
          <div className="dim pad">No scores yet for this trace.</div>
        )}
        {!loading && !error && scores.length > 0 && (
          <div className="score-list">
            {scores.map((s) => (
              <ScoreCard key={s.id} score={s} />
            ))}
          </div>
        )}
      </section>

      <AnnotationForm slug={slug} traceId={traceId} onCreated={reload} />

      <RunEvaluation
        evaluations={evals.data?.items ?? []}
        loading={evals.loading}
        error={evals.error}
        slug={slug}
        traceId={traceId}
        onRan={reload}
      />
    </div>
  )
}

function scorePill(score: Score): { text: string; tone: 'pass' | 'fail' | 'error' } {
  if (score.errored) return { text: 'ERROR', tone: 'error' }
  if (score.passed) return { text: 'PASS', tone: 'pass' }
  return { text: 'FAIL', tone: 'fail' }
}

function ScoreCard({ score }: { score: Score }) {
  const pill = scorePill(score)
  return (
    <div className="score-card">
      <div className="score-card-head">
        <span className="score-value" title={`value ${score.value}`}>
          {formatScore(score.value)}
        </span>
        <span className={`score-pill score-pill-${pill.tone}`}>{pill.text}</span>
        <span className="score-name">{score.name || '(unnamed)'}</span>
        <Badge tone="muted">{score.source}</Badge>
        <span className="score-time dim" title={score.createdAt}>
          {formatRelativeIso(score.createdAt)}
        </span>
      </div>
      {score.reasoning && <div className="score-reasoning">{score.reasoning}</div>}
      {score.tokens > 0 && (
        <div className="score-meta dim">
          {formatInt(score.tokens)} tokens
          {score.costMicrocents > 0 && <> · {formatCost(score.costMicrocents)}</>}
        </div>
      )}
    </div>
  )
}

function AnnotationForm({
  slug,
  traceId,
  onCreated,
}: {
  slug: string
  traceId: string
  onCreated: () => void
}) {
  const [name, setName] = useState('annotation')
  const [value, setValue] = useState(1)
  const [passed, setPassed] = useState(true)
  const [reasoning, setReasoning] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<Error | undefined>()

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed || submitting) return
    setSubmitting(true)
    setSubmitError(undefined)
    try {
      await api.createAnnotation(slug, traceId, {
        name: trimmed,
        value,
        passed,
        reasoning: reasoning.trim() || undefined,
      })
      setName('annotation')
      setValue(1)
      setPassed(true)
      setReasoning('')
      onCreated()
    } catch (err) {
      setSubmitError(err instanceof Error ? err : new Error(String(err)))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className="detail-section">
      <h4>Add annotation</h4>
      <form className="annotation-form" onSubmit={onSubmit}>
        <div className="ann-row">
          <label className="ann-field">
            <span className="ann-label">Name</span>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="annotation"
            />
          </label>
          <label className="ann-field">
            <span className="ann-label">Value {value.toFixed(2)}</span>
            <input
              className="ann-slider"
              type="range"
              min={0}
              max={1}
              step={0.05}
              value={value}
              onChange={(e) => setValue(Number(e.target.value))}
            />
          </label>
        </div>
        <div className="ann-row">
          <label className="ann-toggle">
            <input
              type="checkbox"
              checked={passed}
              onChange={(e) => setPassed(e.target.checked)}
            />
            <span>{passed ? 'Pass' : 'Fail'}</span>
          </label>
        </div>
        <label className="ann-field">
          <span className="ann-label">Note (optional)</span>
          <textarea
            className="ann-textarea"
            value={reasoning}
            onChange={(e) => setReasoning(e.target.value)}
            placeholder="Why this score?"
            rows={3}
          />
        </label>
        {submitError && <ErrorBox error={submitError} />}
        <div className="ann-actions">
          <button
            className="btn btn-primary"
            type="submit"
            disabled={submitting || !name.trim()}
          >
            {submitting ? 'Submitting…' : 'Submit annotation'}
          </button>
        </div>
      </form>
    </section>
  )
}

function RunEvaluation({
  evaluations,
  loading,
  error,
  slug,
  traceId,
  onRan,
}: {
  evaluations: Evaluation[]
  loading: boolean
  error: Error | undefined
  slug: string
  traceId: string
  onRan: () => void
}) {
  const [runningId, setRunningId] = useState<string | undefined>()
  const [runError, setRunError] = useState<Error | undefined>()
  const enabled = evaluations.filter((e) => e.enabled)

  const run = async (ev: Evaluation) => {
    if (runningId) return
    setRunningId(ev.id)
    setRunError(undefined)
    try {
      await api.runEvaluation(slug, ev.id, traceId)
      onRan()
    } catch (err) {
      setRunError(err instanceof Error ? err : new Error(String(err)))
    } finally {
      setRunningId(undefined)
    }
  }

  return (
    <section className="detail-section">
      <h4>Run evaluation</h4>
      {loading && <Spinner label="Loading evaluations…" />}
      {error && <ErrorBox error={error} />}
      {runError && <ErrorBox error={runError} />}
      {!loading && !error && enabled.length === 0 && (
        <EmptyState title="No evaluations configured">
          <p>
            Create an LLM-as-judge evaluation in <strong>Settings</strong> to score this trace
            automatically.
          </p>
        </EmptyState>
      )}
      {!loading && !error && enabled.length > 0 && (
        <div className="eval-run-list">
          {enabled.map((ev) => (
            <div className="eval-run-row" key={ev.id}>
              <div className="eval-run-info">
                <span className="eval-run-name">{ev.name}</span>
                {ev.model && <Badge tone="model">{ev.model}</Badge>}
                {ev.provider && <Badge tone="provider">{ev.provider}</Badge>}
              </div>
              <button
                className="btn"
                onClick={() => run(ev)}
                disabled={runningId != null}
              >
                {runningId === ev.id ? 'Running…' : 'Run'}
              </button>
            </div>
          ))}
        </div>
      )}
    </section>
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
