package otlp

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

// DecodeProtobuf parses an OTLP/HTTP protobuf ExportTraceServiceRequest body.
// The body is wire-compatible with trace/v1.TracesData (both expose
// `repeated ResourceSpans resource_spans = 1`), which avoids the gRPC
// dependency the collector package would pull in.
func DecodeProtobuf(body []byte) ([]ResourceSpans, error) {
	var data tracepb.TracesData
	if err := proto.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("otlp protobuf decode: %w", err)
	}
	out := make([]ResourceSpans, 0, len(data.ResourceSpans))
	for _, rs := range data.ResourceSpans {
		nrs := ResourceSpans{Resource: Attrs{}}
		if rs.Resource != nil {
			nrs.Resource = pbAttrs(rs.Resource.Attributes)
		}
		for _, ss := range rs.ScopeSpans {
			nsc := Scope{}
			if ss.Scope != nil {
				nsc.Name = ss.Scope.Name
				nsc.Version = ss.Scope.Version
			}
			for _, sp := range ss.Spans {
				ns := Span{
					TraceID:      hex.EncodeToString(sp.TraceId),
					SpanID:       hex.EncodeToString(sp.SpanId),
					ParentSpanID: hex.EncodeToString(sp.ParentSpanId),
					Name:         sp.Name,
					Kind:         int(sp.Kind),
					StartNs:      sp.StartTimeUnixNano,
					EndNs:        sp.EndTimeUnixNano,
					Attrs:        pbAttrs(sp.Attributes),
				}
				if sp.Status != nil {
					ns.StatusCode = int(sp.Status.Code)
					ns.StatusMessage = sp.Status.Message
				}
				for _, e := range sp.Events {
					ns.Events = append(ns.Events, Event{Name: e.Name, TimeNs: e.TimeUnixNano, Attrs: pbAttrs(e.Attributes)})
				}
				for _, l := range sp.Links {
					ns.Links = append(ns.Links, Link{TraceID: hex.EncodeToString(l.TraceId), SpanID: hex.EncodeToString(l.SpanId), Attrs: pbAttrs(l.Attributes)})
				}
				nsc.Spans = append(nsc.Spans, ns)
			}
			nrs.Scopes = append(nrs.Scopes, nsc)
		}
		out = append(out, nrs)
	}
	return out, nil
}

func pbAttrs(kvs []*commonpb.KeyValue) Attrs {
	a := Attrs{}
	for _, kv := range kvs {
		if kv == nil {
			continue
		}
		a[kv.Key] = pbAnyValue(kv.Value)
	}
	return a
}

func pbAnyValue(v *commonpb.AnyValue) any {
	if v == nil {
		return nil
	}
	switch x := v.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return x.StringValue
	case *commonpb.AnyValue_IntValue:
		return x.IntValue
	case *commonpb.AnyValue_DoubleValue:
		return x.DoubleValue
	case *commonpb.AnyValue_BoolValue:
		return x.BoolValue
	case *commonpb.AnyValue_BytesValue:
		return string(x.BytesValue)
	case *commonpb.AnyValue_ArrayValue:
		arr := make([]any, 0, len(x.ArrayValue.Values))
		for _, e := range x.ArrayValue.Values {
			arr = append(arr, pbAnyValue(e))
		}
		return arr
	case *commonpb.AnyValue_KvlistValue:
		m := map[string]any{}
		for _, kv := range x.KvlistValue.Values {
			m[kv.Key] = pbAnyValue(kv.Value)
		}
		return m
	}
	return nil
}

// ---- OTLP/JSON --------------------------------------------------------------

type jsonExport struct {
	ResourceSpans []jsonResourceSpans `json:"resourceSpans"`
}
type jsonResourceSpans struct {
	Resource   jsonResource     `json:"resource"`
	ScopeSpans []jsonScopeSpans `json:"scopeSpans"`
}
type jsonResource struct {
	Attributes []jsonKV `json:"attributes"`
}
type jsonScopeSpans struct {
	Scope jsonScope  `json:"scope"`
	Spans []jsonSpan `json:"spans"`
}
type jsonScope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
type jsonSpan struct {
	TraceID           string          `json:"traceId"`
	SpanID            string          `json:"spanId"`
	ParentSpanID      string          `json:"parentSpanId"`
	Name              string          `json:"name"`
	Kind              json.RawMessage `json:"kind"`
	StartTimeUnixNano json.RawMessage `json:"startTimeUnixNano"`
	EndTimeUnixNano   json.RawMessage `json:"endTimeUnixNano"`
	Attributes        []jsonKV        `json:"attributes"`
	Status            jsonStatus      `json:"status"`
	Events            []jsonEvent     `json:"events"`
	Links             []jsonLink      `json:"links"`
}
type jsonStatus struct {
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
}
type jsonEvent struct {
	Name         string          `json:"name"`
	TimeUnixNano json.RawMessage `json:"timeUnixNano"`
	Attributes   []jsonKV        `json:"attributes"`
}
type jsonLink struct {
	TraceID    string   `json:"traceId"`
	SpanID     string   `json:"spanId"`
	Attributes []jsonKV `json:"attributes"`
}
type jsonKV struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

// DecodeJSON parses an OTLP/JSON ExportTraceServiceRequest body. Per the OTLP
// spec, trace_id/span_id are hex strings (not base64), and integers may be
// encoded as JSON strings; both are handled here.
func DecodeJSON(body []byte) ([]ResourceSpans, error) {
	var exp jsonExport
	if err := json.Unmarshal(body, &exp); err != nil {
		return nil, fmt.Errorf("otlp json decode: %w", err)
	}
	out := make([]ResourceSpans, 0, len(exp.ResourceSpans))
	for _, rs := range exp.ResourceSpans {
		nrs := ResourceSpans{Resource: jsonAttrs(rs.Resource.Attributes)}
		for _, ss := range rs.ScopeSpans {
			nsc := Scope{Name: ss.Scope.Name, Version: ss.Scope.Version}
			for _, sp := range ss.Spans {
				ns := Span{
					TraceID:       normalizeID(sp.TraceID),
					SpanID:        normalizeID(sp.SpanID),
					ParentSpanID:  normalizeID(sp.ParentSpanID),
					Name:          sp.Name,
					Kind:          int(parseJSONInt(sp.Kind)),
					StartNs:       parseJSONUint(sp.StartTimeUnixNano),
					EndNs:         parseJSONUint(sp.EndTimeUnixNano),
					StatusCode:    int(parseJSONInt(sp.Status.Code)),
					StatusMessage: sp.Status.Message,
					Attrs:         jsonAttrs(sp.Attributes),
				}
				for _, e := range sp.Events {
					ns.Events = append(ns.Events, Event{Name: e.Name, TimeNs: parseJSONUint(e.TimeUnixNano), Attrs: jsonAttrs(e.Attributes)})
				}
				for _, l := range sp.Links {
					ns.Links = append(ns.Links, Link{TraceID: normalizeID(l.TraceID), SpanID: normalizeID(l.SpanID), Attrs: jsonAttrs(l.Attributes)})
				}
				nsc.Spans = append(nsc.Spans, ns)
			}
			nrs.Scopes = append(nrs.Scopes, nsc)
		}
		out = append(out, nrs)
	}
	return out, nil
}

func jsonAttrs(kvs []jsonKV) Attrs {
	a := Attrs{}
	for _, kv := range kvs {
		a[kv.Key] = parseJSONAnyValue(kv.Value)
	}
	return a
}

func parseJSONAnyValue(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	if v, ok := m["stringValue"]; ok {
		var s string
		_ = json.Unmarshal(v, &s)
		return s
	}
	if v, ok := m["boolValue"]; ok {
		var b bool
		_ = json.Unmarshal(v, &b)
		return b
	}
	if v, ok := m["doubleValue"]; ok {
		var f float64
		_ = json.Unmarshal(v, &f)
		return f
	}
	if v, ok := m["intValue"]; ok {
		return parseJSONInt(v)
	}
	if v, ok := m["arrayValue"]; ok {
		var av struct {
			Values []json.RawMessage `json:"values"`
		}
		_ = json.Unmarshal(v, &av)
		arr := make([]any, 0, len(av.Values))
		for _, e := range av.Values {
			arr = append(arr, parseJSONAnyValue(e))
		}
		return arr
	}
	if v, ok := m["kvlistValue"]; ok {
		var kvl struct {
			Values []jsonKV `json:"values"`
		}
		_ = json.Unmarshal(v, &kvl)
		mm := map[string]any{}
		for _, e := range kvl.Values {
			mm[e.Key] = parseJSONAnyValue(e.Value)
		}
		return mm
	}
	return nil
}

// parseJSONInt parses an int that may be a JSON number or a quoted string.
func parseJSONInt(raw json.RawMessage) int64 {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return 0
	}
	s = strings.Trim(s, `"`)
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(f)
	}
	return 0
}

func parseJSONUint(raw json.RawMessage) uint64 {
	v := parseJSONInt(raw)
	if v < 0 {
		return 0
	}
	return uint64(v)
}

// normalizeID lowercases and strips dashes from a hex id.
func normalizeID(id string) string {
	return strings.ReplaceAll(strings.ToLower(id), "-", "")
}
