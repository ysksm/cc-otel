#!/usr/bin/env bash
# Runs the Go API server (:8080) and the Vite dev server (:5173) together.
# Vite proxies /api and /v1 to the Go server. Ctrl-C stops both.
set -euo pipefail

cd "$(dirname "$0")/.."

CGO_ENABLED="${CGO_ENABLED:-1}" go run ./cmd/ccotel &
GO_PID=$!
trap 'kill "$GO_PID" 2>/dev/null || true' EXIT INT TERM

cd web
[ -d node_modules ] || npm install
npm run dev
