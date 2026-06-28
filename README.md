# cc-otel

A **local-first, single-binary LLM/agent observability tool**, inspired by
[latitude-llm](https://github.com/latitude-dev/latitude-llm) but rebuilt to run
**without Docker, Postgres, ClickHouse, Redis or Temporal**.

- **Backend:** Go (CGO-free)
- **Database:** a single embedded **DuckDB** file
- **Ingestion:** OpenTelemetry **OTLP/HTTP** (`POST /v1/traces`) with GenAI
  semantic-convention enrichment
- **Frontend:** React SPA (Vite), embedded into the binary
- **Targets:** **Windows + macOS** (+ Linux), built with `CGO_ENABLED=0` and
  cross-compiled from a single machine

## Why purego?

DuckDB is a native C++ library, so a Go program normally binds to it via CGO.
To keep `go-jira`-style **pure-Go cross-compilation** (no C toolchain, trivial
`GOOS`/`GOARCH` builds) while still using DuckDB, cc-otel binds to `libduckdb`
through [purego](https://github.com/ebitengine/purego). The vendored driver in
`internal/pduckdb` is based on `fpt/go-pduckdb` (MIT) with an **added Windows
loader** (`LoadLibrary`) so it builds CGO-free on Windows, macOS and Linux.

The native `libduckdb` itself is located on the system or downloaded once and
cached at runtime (see `internal/duckdblib`).

> Note: on **Linux**, purego needs `CGO_ENABLED=1` only to access `dlopen`; on
> the **Windows/macOS** targets it is fully CGO-free.

## Status

Milestone 1: OTLP ingestion → DuckDB → browse traces / spans / sessions.

Implemented:
- Cross-platform purego DuckDB binding + runtime `libduckdb` locator/downloader
- DuckDB schema (`spans` table + `traces`/`sessions` aggregate views), the
  workspace / project / api-key model with default seeding
- OTLP/HTTP ingestion (`POST /v1/traces`, protobuf + JSON) with GenAI
  semantic-convention enrichment (provider/model/operation, additive token
  normalization, cost estimation, session/user/tool/messages) and a REST API
- React SPA: projects, traces & sessions lists, trace detail with span waterfall
- **Evaluations & scores**: manual annotations on traces, plus LLM-as-judge
  evaluations (Scores tab + Evaluations settings)

## Evaluations (LLM-as-judge)

Define evaluators in **Settings → Evaluations** (a name + judge prompt + model),
then run them from a trace's **Scores** tab. Results are stored as scores. You can
also add manual annotations.

The judge LLM is configured via env vars (a per-evaluation provider/model
overrides these). Point it at a local Ollama to keep everything local:

```sh
# Local, fully offline (Ollama):
export CCOTEL_LLM_PROVIDER=openai           # OpenAI-compatible API
export CCOTEL_LLM_BASE_URL=http://localhost:11434/v1
export CCOTEL_LLM_MODEL=llama3.1

# or a hosted provider:
export CCOTEL_LLM_PROVIDER=anthropic
export CCOTEL_LLM_API_KEY=...
export CCOTEL_LLM_MODEL=claude-sonnet-4-6
```

## Quick start

Requirements: **Go 1.24+** and **Node 20+** (for the web UI).

```sh
# 1) Build the SPA and the single binary (embeds the UI)
make build

# 2) Run it (auto-downloads the matching libduckdb on first run)
make run
# -> http://localhost:8080  (prints an ingestion API key on first boot)

# 3) Send sample traces, then open the UI
make sample          # or: ./scripts/send-sample-traces.sh
```

Point any OpenTelemetry exporter at `http://localhost:8080/v1/traces` with
`Authorization: Bearer <api-key>` (the key is printed on first boot and saved to
`./data/dev-api-key.txt`).

### Development (hot reload)

```sh
make dev   # Go API on :8080 + Vite dev server on :5173 (proxies /api, /v1)
```

### Cross-compile (CGO-free) for Windows + macOS

```sh
make cross  # writes dist/ccotel-{windows-amd64.exe,darwin-amd64,darwin-arm64,linux-amd64}
```

Each binary is pure-Go; the matching native `libduckdb` is downloaded and cached
on first run (or provide your own via `DUCKDB_LIBRARY_PATH`).

### Tests

```sh
make test   # on Linux this runs with CGO_ENABLED=1 (purego needs dlopen)
```

## License

MIT. Vendored `internal/pduckdb` retains its upstream MIT license; see
`internal/pduckdb/LICENSE` and `internal/pduckdb/NOTICE.md`.
