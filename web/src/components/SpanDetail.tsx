import type { Span } from '../lib/api'
import { formatAbsolute, formatCost, formatDuration, formatInt } from '../lib/format'
import { parseJson, parseMessages, prettyJson, stringifyUnknown } from '../lib/parse'
import { Badge, Stat } from './common'
import { JsonViewer } from './JsonViewer'
import { MessageThread } from './Messages'

function statusLabel(code: number): { text: string; tone: 'default' | 'error' | 'muted' } {
  if (code === 2) return { text: 'error', tone: 'error' }
  if (code === 1) return { text: 'ok', tone: 'default' }
  return { text: 'unset', tone: 'muted' }
}

export function SpanDetail({ span }: { span: Span }) {
  const status = statusLabel(span.statusCode)
  const inputMsgs = parseMessages(span.inputMessages)
  const outputMsgs = parseMessages(span.outputMessages)
  const sysInstr = parseSystem(span.systemInstructions)
  const hasMessages = inputMsgs.length > 0 || outputMsgs.length > 0 || !!sysInstr
  const hasTool = !!span.toolName || !!span.toolInput || !!span.toolOutput

  return (
    <div className="span-detail">
      <div className="span-detail-header">
        <h3>{span.name || '(unnamed span)'}</h3>
        <Badge tone={status.tone}>{status.text}</Badge>
        {span.operation === 'execute_tool' && <Badge tone="tool">tool</Badge>}
        {span.isStreaming && <Badge tone="muted">streaming</Badge>}
      </div>

      <section className="detail-section">
        <h4>Identifiers</h4>
        <div className="kv">
          <KV k="Trace" v={span.traceId} mono />
          <KV k="Span" v={span.spanId} mono />
          <KV k="Parent" v={span.parentSpanId || '—'} mono />
          {span.sessionId && <KV k="Session" v={span.sessionId} mono />}
          {span.responseId && <KV k="Response" v={span.responseId} mono />}
        </div>
      </section>

      <section className="detail-section">
        <h4>Timing</h4>
        <div className="stat-grid">
          <Stat label="Start" value={<span title={formatAbsolute(span.startTimeNs)}>{formatAbsolute(span.startTimeNs)}</span>} />
          <Stat label="Duration" value={formatDuration(span.endTimeNs - span.startTimeNs)} />
          {span.timeToFirstTokenNs > 0 && (
            <Stat label="TTFT" value={formatDuration(span.timeToFirstTokenNs)} />
          )}
          <Stat label="Streaming" value={span.isStreaming ? 'yes' : 'no'} />
        </div>
      </section>

      <section className="detail-section">
        <h4>Operation</h4>
        <div className="stat-grid">
          {span.operation && <Stat label="Operation" value={span.operation} />}
          {span.provider && <Stat label="Provider" value={<Badge tone="provider">{span.provider}</Badge>} />}
          {span.model && <Stat label="Model" value={<Badge tone="model">{span.model}</Badge>} />}
          {span.responseModel && span.responseModel !== span.model && (
            <Stat label="Response model" value={<Badge tone="model">{span.responseModel}</Badge>} />
          )}
          {span.serviceName && <Stat label="Service" value={span.serviceName} />}
        </div>
        {span.statusCode === 2 && (span.statusMessage || span.errorType) && (
          <div className="error-inline">
            {span.errorType && <strong>{span.errorType}: </strong>}
            {span.statusMessage}
          </div>
        )}
      </section>

      {(span.tokensInput || span.tokensOutput || span.tokensCacheRead || span.tokensCacheCreate || span.tokensReasoning) > 0 && (
        <section className="detail-section">
          <h4>Tokens</h4>
          <div className="stat-grid">
            <Stat label="Input" value={formatInt(span.tokensInput)} />
            <Stat label="Output" value={formatInt(span.tokensOutput)} />
            <Stat label="Cache read" value={formatInt(span.tokensCacheRead)} />
            <Stat label="Cache create" value={formatInt(span.tokensCacheCreate)} />
            <Stat label="Reasoning" value={formatInt(span.tokensReasoning)} />
          </div>
        </section>
      )}

      {(span.costTotalMicrocents || span.costInputMicrocents || span.costOutputMicrocents) > 0 && (
        <section className="detail-section">
          <h4>
            Cost {span.costIsEstimated && <Badge tone="muted">est.</Badge>}
          </h4>
          <div className="stat-grid">
            <Stat label="Input" value={formatCost(span.costInputMicrocents)} />
            <Stat label="Output" value={formatCost(span.costOutputMicrocents)} />
            <Stat label="Total" value={formatCost(span.costTotalMicrocents)} />
          </div>
        </section>
      )}

      {hasMessages && (
        <section className="detail-section">
          <h4>Messages</h4>
          <MessageThread system={sysInstr} messages={[...inputMsgs, ...outputMsgs]} />
        </section>
      )}

      {hasTool && (
        <section className="detail-section">
          <h4>Tool call</h4>
          {span.toolName && <KV k="Name" v={span.toolName} />}
          {span.toolCallId && <KV k="Call id" v={span.toolCallId} mono />}
          {span.toolInput && (
            <div className="tool-block">
              <div className="tool-block-label">Input</div>
              <pre className="json-block">{prettyJson(span.toolInput) || span.toolInput}</pre>
            </div>
          )}
          {span.toolOutput && (
            <div className="tool-block">
              <div className="tool-block-label">Output</div>
              <pre className="json-block">{prettyJson(span.toolOutput) || span.toolOutput}</pre>
            </div>
          )}
        </section>
      )}

      <section className="detail-section">
        <h4>Raw</h4>
        <JsonViewer label="attributes" raw={span.attributes} />
        <JsonViewer label="resource" raw={span.resource} />
        <JsonViewer label="events" raw={span.events} />
        <JsonViewer label="tags" raw={span.tags} />
      </section>
    </div>
  )
}

function parseSystem(s: string | undefined | null): string | undefined {
  if (!s) return undefined
  const v = parseJson<unknown>(s)
  if (v === undefined) return s
  if (typeof v === 'string') return v
  if (Array.isArray(v)) {
    return v
      .map((x) => (typeof x === 'string' ? x : stringifyUnknown(x)))
      .join('\n')
  }
  return stringifyUnknown(v)
}

function KV({ k, v, mono }: { k: string; v: string; mono?: boolean }) {
  return (
    <div className="kv-row">
      <span className="kv-key">{k}</span>
      <span className={`kv-val${mono ? ' mono' : ''}`} title={v}>
        {v}
      </span>
    </div>
  )
}
