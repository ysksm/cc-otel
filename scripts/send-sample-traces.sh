#!/usr/bin/env bash
# Sends a few sample OTLP/JSON traces (GenAI semantic conventions) to a running
# cc-otel instance so the UI has data to show.
#
# Usage: scripts/send-sample-traces.sh [endpoint] [api_key]
#   endpoint  defaults to http://localhost:8080
#   api_key   defaults to $CCOTEL_DEV_API_KEY or ./data/dev-api-key.txt
set -euo pipefail

ENDPOINT="${1:-${CCOTEL_ENDPOINT:-http://localhost:8080}}"
API_KEY="${2:-${CCOTEL_DEV_API_KEY:-}}"
if [[ -z "${API_KEY}" && -f ./data/dev-api-key.txt ]]; then
  API_KEY="$(tr -d '[:space:]' < ./data/dev-api-key.txt)"
fi
if [[ -z "${API_KEY}" ]]; then
  echo "No API key. Set CCOTEL_DEV_API_KEY, pass it as arg 2, or run the server to generate ./data/dev-api-key.txt" >&2
  exit 1
fi

# Base time: 5 minutes ago, in nanoseconds.
BASE_NS=$(( ( $(date +%s) - 300 ) * 1000000000 ))
t() { echo $(( BASE_NS + $1 )); }

read -r -d '' PAYLOAD <<JSON || true
{
  "resourceSpans": [{
    "resource": { "attributes": [
      { "key": "service.name", "value": { "stringValue": "demo-agent" } }
    ]},
    "scopeSpans": [{
      "scope": { "name": "openinference.instrumentation", "version": "1.0.0" },
      "spans": [
        {
          "traceId": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1",
          "spanId": "bbbbbbbbbbbbbbb1",
          "name": "chat gpt-4o",
          "kind": 3,
          "startTimeUnixNano": "$(t 0)",
          "endTimeUnixNano": "$(t 1200000000)",
          "status": { "code": 1 },
          "attributes": [
            { "key": "gen_ai.system", "value": { "stringValue": "openai" } },
            { "key": "gen_ai.request.model", "value": { "stringValue": "gpt-4o" } },
            { "key": "gen_ai.response.model", "value": { "stringValue": "gpt-4o-2024-08-06" } },
            { "key": "gen_ai.operation.name", "value": { "stringValue": "chat" } },
            { "key": "session.id", "value": { "stringValue": "session-demo-1" } },
            { "key": "user.id", "value": { "stringValue": "alice" } },
            { "key": "gen_ai.usage.input_tokens", "value": { "intValue": "1200" } },
            { "key": "gen_ai.usage.output_tokens", "value": { "intValue": "350" } },
            { "key": "gen_ai.input.messages", "value": { "stringValue": "[{\"role\":\"user\",\"content\":\"What is the weather in Tokyo?\"}]" } },
            { "key": "gen_ai.output.messages", "value": { "stringValue": "[{\"role\":\"assistant\",\"content\":\"Let me check.\"}]" } }
          ]
        },
        {
          "traceId": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1",
          "spanId": "bbbbbbbbbbbbbbb2",
          "parentSpanId": "bbbbbbbbbbbbbbb1",
          "name": "execute_tool get_weather",
          "kind": 1,
          "startTimeUnixNano": "$(t 300000000)",
          "endTimeUnixNano": "$(t 500000000)",
          "status": { "code": 1 },
          "attributes": [
            { "key": "gen_ai.operation.name", "value": { "stringValue": "execute_tool" } },
            { "key": "gen_ai.tool.name", "value": { "stringValue": "get_weather" } },
            { "key": "session.id", "value": { "stringValue": "session-demo-1" } },
            { "key": "gen_ai.tool.call.arguments", "value": { "stringValue": "{\"city\":\"Tokyo\"}" } },
            { "key": "gen_ai.tool.call.result", "value": { "stringValue": "{\"tempC\":18}" } }
          ]
        },
        {
          "traceId": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2",
          "spanId": "ccccccccccccccc1",
          "name": "chat claude",
          "kind": 3,
          "startTimeUnixNano": "$(t 60000000000)",
          "endTimeUnixNano": "$(t 64000000000)",
          "status": { "code": 2, "message": "rate limited" },
          "attributes": [
            { "key": "gen_ai.system", "value": { "stringValue": "anthropic" } },
            { "key": "gen_ai.request.model", "value": { "stringValue": "claude-sonnet-4-6" } },
            { "key": "session.id", "value": { "stringValue": "session-demo-2" } },
            { "key": "error.type", "value": { "stringValue": "rate_limit_error" } },
            { "key": "gen_ai.usage.input_tokens", "value": { "intValue": "800" } },
            { "key": "gen_ai.usage.output_tokens", "value": { "intValue": "0" } }
          ]
        }
      ]
    }]
  }]
}
JSON

echo "Sending sample traces to ${ENDPOINT}/v1/traces ..."
curl -sS -X POST "${ENDPOINT}/v1/traces" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "${PAYLOAD}"
echo
