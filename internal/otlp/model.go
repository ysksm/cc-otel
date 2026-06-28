// Package otlp decodes OpenTelemetry OTLP/HTTP trace payloads (protobuf and JSON)
// into a neutral representation and enriches spans with GenAI semantic
// conventions, mirroring latitude-llm's span model.
package otlp

import (
	"fmt"
	"strconv"
)

// Attrs is a flattened attribute map (string -> string|int64|float64|bool|[]any|map).
type Attrs map[string]any

// Span is a neutral OTLP span (decoder-independent).
type Span struct {
	TraceID       string
	SpanID        string
	ParentSpanID  string
	Name          string
	Kind          int
	StartNs       uint64
	EndNs         uint64
	StatusCode    int // 0=unset 1=ok 2=error
	StatusMessage string
	Attrs         Attrs
	Events        []Event
	Links         []Link
}

// Event is a span event.
type Event struct {
	Name   string
	TimeNs uint64
	Attrs  Attrs
}

// Link is a span link.
type Link struct {
	TraceID string
	SpanID  string
	Attrs   Attrs
}

// Scope groups spans emitted by one instrumentation scope.
type Scope struct {
	Name    string
	Version string
	Spans   []Span
}

// ResourceSpans groups scopes sharing a resource.
type ResourceSpans struct {
	Resource Attrs
	Scopes   []Scope
}

// Str returns the first present attribute coerced to string.
func (a Attrs) Str(keys ...string) string {
	for _, k := range keys {
		if v, ok := a[k]; ok && v != nil {
			switch t := v.(type) {
			case string:
				return t
			default:
				return fmt.Sprint(t)
			}
		}
	}
	return ""
}

// I64 returns the first present attribute as int64.
func (a Attrs) I64(keys ...string) (int64, bool) {
	for _, k := range keys {
		if v, ok := a[k]; ok && v != nil {
			switch t := v.(type) {
			case int64:
				return t, true
			case int:
				return int64(t), true
			case float64:
				return int64(t), true
			case string:
				if n, err := strconv.ParseInt(t, 10, 64); err == nil {
					return n, true
				}
				if f, err := strconv.ParseFloat(t, 64); err == nil {
					return int64(f), true
				}
			}
		}
	}
	return 0, false
}

// F64 returns the first present attribute as float64.
func (a Attrs) F64(keys ...string) (float64, bool) {
	for _, k := range keys {
		if v, ok := a[k]; ok && v != nil {
			switch t := v.(type) {
			case float64:
				return t, true
			case int64:
				return float64(t), true
			case int:
				return float64(t), true
			case string:
				if f, err := strconv.ParseFloat(t, 64); err == nil {
					return f, true
				}
			}
		}
	}
	return 0, false
}

// Bool returns the first present attribute as bool.
func (a Attrs) Bool(keys ...string) (bool, bool) {
	for _, k := range keys {
		if v, ok := a[k]; ok && v != nil {
			switch t := v.(type) {
			case bool:
				return t, true
			case string:
				if b, err := strconv.ParseBool(t); err == nil {
					return b, true
				}
			}
		}
	}
	return false, false
}

// Has reports whether any of the keys is present.
func (a Attrs) Has(keys ...string) bool {
	for _, k := range keys {
		if _, ok := a[k]; ok {
			return true
		}
	}
	return false
}

// Raw returns the first present raw value.
func (a Attrs) Raw(keys ...string) (any, bool) {
	for _, k := range keys {
		if v, ok := a[k]; ok {
			return v, true
		}
	}
	return nil, false
}
