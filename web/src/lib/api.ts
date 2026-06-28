// Typed REST client for the cc-otel Go backend.
//
// NOTE: several Span/Trace fields are JSON encoded as STRINGS by the backend and
// must be JSON.parse()'d on the client. See parse helpers below.

export interface Project {
  id: string
  workspaceId: string
  name: string
  slug: string
  firstTraceAt?: string
  createdAt: string
}

export interface TraceSummary {
  traceId: string
  sessionId: string
  spanCount: number
  errorCount: number
  startTimeNs: number
  endTimeNs: number
  durationNs: number
  tokensInput: number
  tokensOutput: number
  tokensCacheRead: number
  tokensCacheCreate: number
  tokensReasoning: number
  costTotalMicrocents: number
  models: string // JSON array string e.g. "[\"gpt-4o\"]"
  providers: string // JSON array string
  rootSpanName: string
  rootSpanId: string
  scoreCount: number
  passedCount: number
  failedCount: number
  avgScore?: number // omitted/undefined when no scores
}

export interface Span {
  sessionId: string
  traceId: string
  spanId: string
  parentSpanId: string
  startTimeNs: number
  endTimeNs: number
  name: string
  serviceName: string
  kind: number
  statusCode: number // 0 unset, 1 ok, 2 error
  statusMessage: string
  errorType: string
  scopeName: string
  scopeVersion: string
  operation: string
  provider: string
  model: string
  responseModel: string
  tokensInput: number
  tokensOutput: number
  tokensCacheRead: number
  tokensCacheCreate: number
  tokensReasoning: number
  costInputMicrocents: number
  costOutputMicrocents: number
  costTotalMicrocents: number
  costIsEstimated: boolean
  timeToFirstTokenNs: number
  isStreaming: boolean
  responseId: string
  userId: string
  userEmail: string
  toolCallId: string
  toolName: string
  toolInput: string
  toolOutput: string
  // The trailing string fields are JSON encoded as strings; JSON.parse them.
  finishReasons: string
  toolNames: string
  inputMessages: string
  outputMessages: string
  systemInstructions: string
  toolDefinitions: string
  events: string
  links: string
  tags: string
  attributes: string
  resource: string
}

export interface SessionSummary {
  sessionId: string
  traceCount: number
  spanCount: number
  errorCount: number
  startTimeNs: number
  endTimeNs: number
  durationNs: number
  tokensInput: number
  tokensOutput: number
  costTotalMicrocents: number
  models: string
  providers: string
  userId: string
}

export interface UserSummary {
  userId: string
  userEmail: string
  traceCount: number
  spanCount: number
  errorCount: number
  tokensInput: number
  tokensOutput: number
  costTotalMicrocents: number
  lastSeenNs: number
  models: string // JSON array string e.g. "[\"gpt-4o\"]"
}

export interface ToolSummary {
  toolName: string
  callCount: number
  errorCount: number
  avgDurationNs: number
  lastSeenNs: number
}

export type AnalyticsRange = '1h' | '24h' | '7d' | '30d'

export interface AnalyticsTotals {
  traceCount: number
  spanCount: number
  errorCount: number
  costTotalMicrocents: number
  tokensInput: number
  tokensOutput: number
}

export interface AnalyticsBucket {
  bucketStartNs: number
  traceCount: number
  spanCount: number
  errorCount: number
  costTotalMicrocents: number
  tokensInput: number
  tokensOutput: number
  avgDurationNs: number
}

export interface AnalyticsModel {
  model: string
  costTotalMicrocents: number
  traceCount: number
}

export interface Analytics {
  range: string
  bucketNs: number
  sinceNs: number
  totals: AnalyticsTotals
  series: AnalyticsBucket[]
  topModels: AnalyticsModel[]
}

export interface APIKey {
  id: string
  workspaceId: string
  name: string
  tokenPreview: string
  createdAt: string
  lastUsedAt?: string
  token?: string // plaintext, present ONCE on creation
}

export interface Score {
  id: string
  traceId: string
  spanId: string
  sessionId: string
  source: 'annotation' | 'evaluation'
  sourceId: string
  name: string
  value: number // 0..1
  passed: boolean
  errored: boolean
  reasoning: string
  durationNs: number
  tokens: number
  costMicrocents: number
  createdAt: string
}

export interface Evaluation {
  id: string
  workspaceId: string
  projectId: string
  name: string
  slug: string
  prompt: string
  provider: string
  model: string
  enabled: boolean
  createdAt: string
}

export type MonitorConditionType =
  | 'trace_error'
  | 'cost_gt'
  | 'latency_gt'
  | 'tokens_gt'
  | 'model_used'

export interface Monitor {
  id: string
  workspaceId: string
  projectId: string
  name: string
  conditionType: MonitorConditionType
  threshold: number
  valueStr: string
  enabled: boolean
  createdAt: string
}

export interface Signal {
  id: string
  monitorId: string
  monitorName: string
  traceId: string
  sessionId: string
  description: string
  value: number
  createdAt: string
}

export interface Account {
  id: string
  email: string
  role: 'admin' | 'member'
  createdAt: string
}

export interface Me {
  enabled: boolean
  authenticated: boolean
  account?: Account
}

export interface TracesResponse {
  items: TraceSummary[]
  nextCursor: string
}

export interface TraceDetailResponse {
  trace: TraceSummary
  spans: Span[]
}

export class ApiError extends Error {
  status: number
  body: string
  constructor(status: number, body: string) {
    super(`API error ${status}: ${body}`)
    this.status = status
    this.body = body
  }
}

// Global 401 handler. The AuthProvider registers a callback here so a session
// that expires mid-use re-triggers an auth refresh, dropping the user back to
// the login screen. The /api/auth/* endpoints are exempt so login failures
// (expected 401s) don't fire it.
let onUnauthorized: (() => void) | undefined
export function setOnUnauthorized(handler: (() => void) | undefined): void {
  onUnauthorized = handler
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    let body = ''
    try {
      body = await res.text()
    } catch {
      /* ignore */
    }
    if (res.status === 401 && !path.startsWith('/api/auth/')) {
      onUnauthorized?.()
    }
    throw new ApiError(res.status, body)
  }
  if (res.status === 204) {
    return undefined as T
  }
  return (await res.json()) as T
}

export const api = {
  listProjects(): Promise<{ items: Project[] }> {
    return request('/api/projects')
  },
  createProject(body: { name: string; slug: string }): Promise<Project> {
    return request('/api/projects', { method: 'POST', body: JSON.stringify(body) })
  },
  getProject(slug: string): Promise<Project> {
    return request(`/api/projects/${encodeURIComponent(slug)}`)
  },
  listTraces(
    slug: string,
    opts?: {
      limit?: number
      before?: number
      q?: string
      provider?: string
      model?: string
      errors?: boolean
      minDuration?: number
    },
  ): Promise<TracesResponse> {
    const params = new URLSearchParams()
    params.set('limit', String(opts?.limit ?? 50))
    if (opts?.before != null) params.set('before', String(opts.before))
    if (opts?.q) params.set('q', opts.q)
    if (opts?.provider) params.set('provider', opts.provider)
    if (opts?.model) params.set('model', opts.model)
    if (opts?.errors) params.set('errors', 'true')
    if (opts?.minDuration != null) params.set('minDuration', String(opts.minDuration))
    return request(`/api/projects/${encodeURIComponent(slug)}/traces?${params.toString()}`)
  },
  getTrace(slug: string, traceId: string): Promise<TraceDetailResponse> {
    return request(
      `/api/projects/${encodeURIComponent(slug)}/traces/${encodeURIComponent(traceId)}`,
    )
  },
  listSessions(slug: string, opts?: { limit?: number }): Promise<{ items: SessionSummary[] }> {
    const params = new URLSearchParams()
    params.set('limit', String(opts?.limit ?? 50))
    return request(`/api/projects/${encodeURIComponent(slug)}/sessions?${params.toString()}`)
  },
  listUsers(slug: string): Promise<{ items: UserSummary[] }> {
    return request(`/api/projects/${encodeURIComponent(slug)}/users`)
  },
  listTools(slug: string): Promise<{ items: ToolSummary[] }> {
    return request(`/api/projects/${encodeURIComponent(slug)}/tools`)
  },
  getAnalytics(slug: string, range: AnalyticsRange = '24h'): Promise<Analytics> {
    const params = new URLSearchParams()
    params.set('range', range)
    return request(`/api/projects/${encodeURIComponent(slug)}/analytics?${params.toString()}`)
  },
  getTool(slug: string, toolName: string): Promise<{ tool: ToolSummary; calls: Span[] }> {
    return request(
      `/api/projects/${encodeURIComponent(slug)}/tools/${encodeURIComponent(toolName)}`,
    )
  },
  listApiKeys(slug: string): Promise<{ items: APIKey[] }> {
    return request(`/api/projects/${encodeURIComponent(slug)}/api-keys`)
  },
  createApiKey(slug: string, body: { name: string }): Promise<APIKey> {
    return request(`/api/projects/${encodeURIComponent(slug)}/api-keys`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  listTraceScores(slug: string, traceId: string): Promise<{ items: Score[] }> {
    return request(
      `/api/projects/${encodeURIComponent(slug)}/traces/${encodeURIComponent(traceId)}/scores`,
    )
  },
  createAnnotation(
    slug: string,
    traceId: string,
    body: { name: string; spanId?: string; value: number; passed?: boolean; reasoning?: string },
  ): Promise<Score> {
    return request(
      `/api/projects/${encodeURIComponent(slug)}/traces/${encodeURIComponent(traceId)}/scores`,
      { method: 'POST', body: JSON.stringify(body) },
    )
  },
  listEvaluations(slug: string): Promise<{ items: Evaluation[] }> {
    return request(`/api/projects/${encodeURIComponent(slug)}/evaluations`)
  },
  createEvaluation(
    slug: string,
    body: {
      name: string
      slug?: string
      prompt: string
      provider?: string
      model?: string
      enabled?: boolean
    },
  ): Promise<Evaluation> {
    return request(`/api/projects/${encodeURIComponent(slug)}/evaluations`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  runEvaluation(slug: string, evalId: string, traceId: string): Promise<Score> {
    return request(
      `/api/projects/${encodeURIComponent(slug)}/evaluations/${encodeURIComponent(evalId)}/run`,
      { method: 'POST', body: JSON.stringify({ traceId }) },
    )
  },
  listMonitors(slug: string): Promise<{ items: Monitor[] }> {
    return request(`/api/projects/${encodeURIComponent(slug)}/monitors`)
  },
  createMonitor(
    slug: string,
    body: {
      name: string
      conditionType: MonitorConditionType
      threshold?: number
      valueStr?: string
      enabled?: boolean
    },
  ): Promise<Monitor> {
    return request(`/api/projects/${encodeURIComponent(slug)}/monitors`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  evaluateMonitors(slug: string): Promise<{ created: number }> {
    return request(`/api/projects/${encodeURIComponent(slug)}/monitors/evaluate`, {
      method: 'POST',
      body: JSON.stringify({}),
    })
  },
  listSignals(slug: string): Promise<{ items: Signal[] }> {
    return request(`/api/projects/${encodeURIComponent(slug)}/signals`)
  },
  getMe(): Promise<Me> {
    return request('/api/auth/me')
  },
  login(email: string, password: string): Promise<{ account: Account }> {
    return request('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
  },
  logout(): Promise<{ ok: true }> {
    return request('/api/auth/logout', { method: 'POST', body: JSON.stringify({}) })
  },
  listAccounts(): Promise<{ items: Account[] }> {
    return request('/api/accounts')
  },
  createAccount(body: {
    email: string
    password: string
    role: 'admin' | 'member'
  }): Promise<Account> {
    return request('/api/accounts', { method: 'POST', body: JSON.stringify(body) })
  },
}
