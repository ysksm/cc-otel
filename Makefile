APP    := ccotel
BIN    := bin/$(APP)
WEB    := web
GO     := go
# Linux needs CGO_ENABLED=1 for purego's dlopen; macOS works either way.
# Windows/macOS cross-builds use CGO_ENABLED=0 (see `cross`).
CGO    ?= 1

.PHONY: help web build run dev test sample cross clean tidy

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

web: ## Build the React SPA into internal/webui/dist
	cd $(WEB) && npm install && npm run build

build: web ## Build the single binary (embeds the SPA)
	CGO_ENABLED=$(CGO) $(GO) build -o $(BIN) ./cmd/ccotel
	@echo "built $(BIN)"

run: ## Run the server (downloads libduckdb on first run)
	CGO_ENABLED=$(CGO) $(GO) run ./cmd/ccotel

dev: ## Run Go API (:8080) + Vite dev (:5173) together
	./scripts/dev.sh

test: ## Run the Go test suite
	CGO_ENABLED=$(CGO) $(GO) test ./...

sample: ## Send sample OTLP traces to a running server
	./scripts/send-sample-traces.sh

cross: web ## Cross-compile CGO-free binaries (windows/macos/linux) into dist/
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -o dist/$(APP)-windows-amd64.exe ./cmd/ccotel
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 $(GO) build -o dist/$(APP)-darwin-amd64 ./cmd/ccotel
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 $(GO) build -o dist/$(APP)-darwin-arm64 ./cmd/ccotel
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 $(GO) build -o dist/$(APP)-linux-amd64 ./cmd/ccotel
	@echo "cross-compiled into dist/ (each needs the matching libduckdb at runtime; it is auto-downloaded)"

tidy: ## go mod tidy
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -rf bin dist
	rm -rf internal/webui/dist/assets internal/webui/dist/index.html
