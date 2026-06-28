#!/usr/bin/env bash
# Build cc-otel: the React SPA (embedded) + the Go binary.
#
# Usage:
#   scripts/build.sh            Build the SPA + a binary for the current OS -> bin/ccotel
#   scripts/build.sh --release  Build the SPA + cross-compile all platforms -> dist/ (+ archives)
#
# Notes:
#   - The Go side is CGO-free, EXCEPT on Linux where purego needs CGO_ENABLED=1
#     to access dlopen. Cross-compiled release binaries are all CGO_ENABLED=0.
#   - The native libduckdb is downloaded/cached at runtime, not bundled here.
set -euo pipefail
cd "$(dirname "$0")/.."

APP=ccotel
RELEASE_TARGETS="windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64"

log() { printf '\033[36m==>\033[0m %s\n' "$*"; }

build_web() {
  log "Building web UI (Vite) -> internal/webui/dist"
  ( cd web && npm install && npm run build )
  # vite's emptyOutDir wipes the tracked placeholder; restore it so the
  # //go:embed dist directive still compiles from a clean checkout.
  printf 'placeholder - replaced by Vite build output\n' > internal/webui/dist/.gitkeep
}

build_local() {
  local os cgo
  os="$(go env GOOS)"
  cgo=0
  [ "$os" = "linux" ] && cgo=1
  log "Building binary for $os ($(go env GOARCH), CGO_ENABLED=$cgo) -> bin/$APP"
  mkdir -p bin
  CGO_ENABLED=$cgo go build -trimpath -o "bin/$APP" ./cmd/ccotel
  log "Done: bin/$APP"
}

build_release() {
  log "Cross-compiling release binaries (CGO_ENABLED=0) -> dist/"
  rm -rf dist && mkdir -p dist
  for t in $RELEASE_TARGETS; do
    local os="${t%/*}" arch="${t#*/}" out name
    out="dist/${APP}-${os}-${arch}"
    [ "$os" = "windows" ] && out="${out}.exe"
    log "  ${os}/${arch}"
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -o "$out" ./cmd/ccotel
    name="$(basename "$out")"
    if [ "$os" = "windows" ]; then
      ( cd dist && zip -q "${APP}-${os}-${arch}.zip" "$name" )
    else
      ( cd dist && tar czf "${APP}-${os}-${arch}.tar.gz" "$name" )
    fi
  done
  log "Release artifacts:"
  ls -la dist
}

build_web
case "${1:-}" in
  --release | -r) build_release ;;
  "") build_local ;;
  *)
    echo "unknown option: $1" >&2
    echo "usage: scripts/build.sh [--release]" >&2
    exit 2
    ;;
esac
