// Package model holds the core domain types shared across cc-otel.
package model

import "time"

// Workspace is the top-level tenant (one local workspace by default).
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
}

// Project groups telemetry within a workspace. Traces belong to a project.
type Project struct {
	ID           string     `json:"id"`
	WorkspaceID  string     `json:"workspaceId"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	FirstTraceAt *time.Time `json:"firstTraceAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// APIKey authenticates trace ingestion and scopes it to a workspace.
type APIKey struct {
	ID           string     `json:"id"`
	WorkspaceID  string     `json:"workspaceId"`
	Name         string     `json:"name"`
	TokenHash    string     `json:"-"`
	TokenPreview string     `json:"tokenPreview"`
	Token        string     `json:"token,omitempty"` // only returned on creation
	CreatedAt    time.Time  `json:"createdAt"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
}

// Span is a single OpenTelemetry span enriched with GenAI/LLM fields.
// JSON-shaped fields are kept as raw JSON strings (the values DuckDB stores).
type Span struct {
	WorkspaceID   string `json:"-"`
	ProjectID     string `json:"-"`
	APIKeyID      string `json:"-"`
	SessionID     string `json:"sessionId"`
	TraceID       string `json:"traceId"`
	SpanID        string `json:"spanId"`
	ParentSpanID  string `json:"parentSpanId"`
	StartTimeNs   int64  `json:"startTimeNs"`
	EndTimeNs     int64  `json:"endTimeNs"`
	Name          string `json:"name"`
	ServiceName   string `json:"serviceName"`
	Kind          int8   `json:"kind"`
	StatusCode    int16  `json:"statusCode"`
	StatusMessage string `json:"statusMessage"`
	ErrorType     string `json:"errorType"`
	ScopeName     string `json:"scopeName"`
	ScopeVersion  string `json:"scopeVersion"`

	Operation     string `json:"operation"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	ResponseModel string `json:"responseModel"`

	TokensInput       int64 `json:"tokensInput"`
	TokensOutput      int64 `json:"tokensOutput"`
	TokensCacheRead   int64 `json:"tokensCacheRead"`
	TokensCacheCreate int64 `json:"tokensCacheCreate"`
	TokensReasoning   int64 `json:"tokensReasoning"`

	CostInputMicrocents  int64 `json:"costInputMicrocents"`
	CostOutputMicrocents int64 `json:"costOutputMicrocents"`
	CostTotalMicrocents  int64 `json:"costTotalMicrocents"`
	CostIsEstimated      bool  `json:"costIsEstimated"`

	TimeToFirstTokenNs int64 `json:"timeToFirstTokenNs"`
	IsStreaming        bool  `json:"isStreaming"`

	ResponseID    string `json:"responseId"`
	FinishReasons string `json:"finishReasons"` // JSON array

	UserID    string `json:"userId"`
	UserEmail string `json:"userEmail"`

	ToolCallID string `json:"toolCallId"`
	ToolName   string `json:"toolName"`
	ToolInput  string `json:"toolInput"`
	ToolOutput string `json:"toolOutput"`
	ToolNames  string `json:"toolNames"` // JSON array

	InputMessages      string `json:"inputMessages"`      // JSON
	OutputMessages     string `json:"outputMessages"`     // JSON
	SystemInstructions string `json:"systemInstructions"` // JSON
	ToolDefinitions    string `json:"toolDefinitions"`    // JSON
	EventsJSON         string `json:"events"`             // JSON
	LinksJSON          string `json:"links"`              // JSON

	Tags       string `json:"tags"`       // JSON array
	Attributes string `json:"attributes"` // JSON object
	Resource   string `json:"resource"`   // JSON object
}

// Evaluation is an LLM-as-judge evaluator definition.
type Evaluation struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Prompt      string    `json:"prompt"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Score is an annotation or evaluation result attached to a trace/span.
type Score struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"-"`
	ProjectID       string    `json:"-"`
	TraceID         string    `json:"traceId"`
	SpanID          string    `json:"spanId"`
	SessionID       string    `json:"sessionId"`
	Source          string    `json:"source"`   // annotation | evaluation
	SourceID        string    `json:"sourceId"` // evaluation id, or "UI"
	Name            string    `json:"name"`
	Value           float64   `json:"value"`
	Passed          bool      `json:"passed"`
	Errored         bool      `json:"errored"`
	Reasoning       string    `json:"reasoning"`
	DurationNs      int64     `json:"durationNs"`
	Tokens          int64     `json:"tokens"`
	CostMicrocents  int64     `json:"costMicrocents"`
	CreatedAt       time.Time `json:"createdAt"`
}

// UserSummary aggregates a project's spans by end-user (user.id).
type UserSummary struct {
	UserID              string `json:"userId"`
	UserEmail           string `json:"userEmail"`
	TraceCount          int64  `json:"traceCount"`
	SpanCount           int64  `json:"spanCount"`
	ErrorCount          int64  `json:"errorCount"`
	TokensInput         int64  `json:"tokensInput"`
	TokensOutput        int64  `json:"tokensOutput"`
	CostTotalMicrocents int64  `json:"costTotalMicrocents"`
	LastSeenNs          int64  `json:"lastSeenNs"`
	Models              string `json:"models"` // JSON array
}

// ToolSummary aggregates tool-execution spans by tool name.
type ToolSummary struct {
	ToolName      string `json:"toolName"`
	CallCount     int64  `json:"callCount"`
	ErrorCount    int64  `json:"errorCount"`
	AvgDurationNs int64  `json:"avgDurationNs"`
	LastSeenNs    int64  `json:"lastSeenNs"`
}

// TraceSummary is an aggregated row from the traces view.
type TraceSummary struct {
	TraceID             string `json:"traceId"`
	SessionID           string `json:"sessionId"`
	SpanCount           int64  `json:"spanCount"`
	ErrorCount          int64  `json:"errorCount"`
	StartTimeNs         int64  `json:"startTimeNs"`
	EndTimeNs           int64  `json:"endTimeNs"`
	DurationNs          int64  `json:"durationNs"`
	TokensInput         int64  `json:"tokensInput"`
	TokensOutput        int64  `json:"tokensOutput"`
	TokensCacheRead     int64  `json:"tokensCacheRead"`
	TokensCacheCreate   int64  `json:"tokensCacheCreate"`
	TokensReasoning     int64  `json:"tokensReasoning"`
	CostTotalMicrocents int64  `json:"costTotalMicrocents"`
	Models              string `json:"models"`    // JSON array
	Providers           string `json:"providers"` // JSON array
	RootSpanName        string `json:"rootSpanName"`
	RootSpanID          string `json:"rootSpanId"`
}

// SessionSummary is an aggregated row from the sessions view.
type SessionSummary struct {
	SessionID           string `json:"sessionId"`
	TraceCount          int64  `json:"traceCount"`
	SpanCount           int64  `json:"spanCount"`
	ErrorCount          int64  `json:"errorCount"`
	StartTimeNs         int64  `json:"startTimeNs"`
	EndTimeNs           int64  `json:"endTimeNs"`
	DurationNs          int64  `json:"durationNs"`
	TokensInput         int64  `json:"tokensInput"`
	TokensOutput        int64  `json:"tokensOutput"`
	CostTotalMicrocents int64  `json:"costTotalMicrocents"`
	Models              string `json:"models"`
	Providers           string `json:"providers"`
	UserID              string `json:"userId"`
}
