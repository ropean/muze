# Go project tasks — similar role to npm scripts in Node (single entry: `make <target>`).
# Optional: install https://taskfile.dev or github.com/go-task/task for YAML-style tasks.

.PHONY: build test fmt lint vet clean

BINARY_NAME ?= music-provider-cn

build:
	go build -o $(BINARY_NAME) .

test:
	go test -race ./...

fmt:
	gofmt -s -w .
	go fmt ./...

vet:
	go vet ./...

# `lint`: standard library checks. For stricter rules, install golangci-lint and use `make lint-full`.
lint: vet

lint-full:
	golangci-lint run ./...

clean:
	rm -f $(BINARY_NAME)
