# Vendored: go-pduckdb

This directory vendors **github.com/fpt/go-pduckdb** (v0.1.6, MIT License) — a
CGO-free DuckDB `database/sql` driver built on purego. See `LICENSE`.

## Local modifications
- `duckdb/library.go`: the shared-library loader was refactored to call a
  per-platform `openLibrary` function.
- `duckdb/loader_unix.go` (new): Unix `dlopen` path (darwin/linux/freebsd).
- `duckdb/loader_windows.go` (new): **Windows support added** via the Win32
  `LoadLibrary` API (`golang.org/x/sys/windows`). Upstream only shipped the Unix
  path, so the driver did not compile for Windows; this restores CGO-free
  Windows builds.
- Import paths were rewritten under `github.com/ysksm/cc-otel/internal/pduckdb`.
