# Go project tasks — similar role to npm scripts in Node (single entry: `make <target>`).
# Optional: install https://taskfile.dev or github.com/go-task/task for YAML-style tasks.

.PHONY: help build test test-cli test-http fmt lint vet clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

BINARY_NAME ?= muze
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X github.com/ropean/muze/internal/selfupdate.Version=$(VERSION)

build: ## Build the binary
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) .

test: ## Run all tests
	go test -race ./...

test-cli: ## Run CLI tests (cmd, api, downloader, models)
	go test -race ./cmd/... ./internal/api/... ./internal/downloader/... ./internal/models/...

test-http: ## Run HTTP server tests
	go test -race ./internal/server/...

fmt: ## Format source code
	gofmt -s -w .
	go fmt ./...

vet: ## Static analysis for common bugs (format mismatches, lock copies, etc.)
	go vet ./...

lint: vet ## Alias for vet

lint-full: ## Strict lint with golangci-lint (style, complexity, unused code, etc.)
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...

clean: ## Remove build artifacts
	rm -f $(BINARY_NAME)
