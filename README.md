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

Milestone 1 (in progress): OTLP ingestion → DuckDB → browse traces/spans/sessions.

Implemented so far:
- Cross-platform purego DuckDB binding + runtime `libduckdb` locator/downloader
- DuckDB schema (`spans` table + `traces`/`sessions` aggregate views) and the
  workspace/project/api-key model with default seeding

## Development

Requirements: Go 1.24+. (Node 20+ for the web UI, once added.)

```sh
go test ./...        # on Linux, runs with CGO_ENABLED=1 for dlopen
```

## License

MIT. Vendored `internal/pduckdb` retains its upstream MIT license; see
`internal/pduckdb/LICENSE` and `internal/pduckdb/NOTICE.md`.
