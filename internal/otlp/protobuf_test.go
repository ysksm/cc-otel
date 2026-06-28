package otlp

import (
	"testing"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

func kvStr(k, v string) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: k, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: v}}}
}
func kvInt(k string, v int64) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: k, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: v}}}
}

func TestDecodeProtobuf(t *testing.T) {
	td := &tracepb.TracesData{
		ResourceSpans: []*tracepb.ResourceSpans{{
			Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{kvStr("service.name", "svc")}},
			ScopeSpans: []*tracepb.ScopeSpans{{
				Scope: &commonpb.InstrumentationScope{Name: "openai", Version: "1"},
				Spans: []*tracepb.Span{{
					TraceId:           []byte{0xaa, 0xbb, 0xcc, 0xdd, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
					SpanId:            []byte{1, 2, 3, 4, 5, 6, 7, 8},
					Name:              "chat",
					Kind:              tracepb.Span_SPAN_KIND_CLIENT,
					StartTimeUnixNano: 1000,
					EndTimeUnixNano:   2000,
					Status:            &tracepb.Status{Code: tracepb.Status_STATUS_CODE_OK},
					Attributes: []*commonpb.KeyValue{
						kvStr("gen_ai.system", "openai"),
						kvStr("gen_ai.request.model", "gpt-4o"),
						kvStr("gen_ai.operation.name", "chat"),
						kvInt("gen_ai.usage.input_tokens", 100),
						kvInt("gen_ai.usage.output_tokens", 20),
					},
				}},
			}},
		}},
	}
	body, err := proto.Marshal(td)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rspans, err := DecodeProtobuf(body)
	if err != nil {
		t.Fatalf("decode protobuf: %v", err)
	}
	spans, rejected := ToSpans(rspans, IngestContext{
		WorkspaceID: "ws", APIKeyID: "k",
		ResolveProject: func(string) (string, bool) { return "p", true },
	})
	if rejected != 0 || len(spans) != 1 {
		t.Fatalf("want 1 span, got %d (rejected %d)", len(spans), rejected)
	}
	s := spans[0]
	if s.Provider != "openai" || s.Model != "gpt-4o" || s.Operation != "chat" {
		t.Fatalf("enrichment wrong: %+v", s)
	}
	if s.TraceID != "aabbccdd0102030405060708090a0b0c" || s.SpanID != "0102030405060708" {
		t.Fatalf("id hex wrong: trace=%s span=%s", s.TraceID, s.SpanID)
	}
	if s.TokensInput != 100 || s.TokensOutput != 20 {
		t.Fatalf("tokens wrong: %d/%d", s.TokensInput, s.TokensOutput)
	}
}
