.PHONY: build build-orbit build-apsis test test-short test-verbose test-coverage test-local lint clean install install-local-next-task modernize

BIN_DIR ?= $(HOME)/.local/bin

# Build metadata
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)

# Build both binaries
build: build-orbit build-apsis

# Build orbit only
build-orbit:
	go build -ldflags="$(LDFLAGS)" -o orbit ./cmd/orbit

# Build apsis only
build-apsis:
	go build -ldflags="$(LDFLAGS)" -o apsis ./cmd/apsis

# Run all tests
test:
	go test ./...

# Run tests quickly (skip slow retry tests)
test-short:
	go test -short ./...

# Local dispatch, real rune integration, agent output and cancellation checks
test-local:
	go test ./cmd/orbit ./internal/orbit ./internal/agents ./internal/agents/codex ./internal/rune

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f orbit apsis coverage.out coverage.html

# Install both binaries to GOPATH/bin
install:
	go install ./cmd/orbit
	go install ./cmd/apsis

# Replace the agentic-coding prototype with Orbit's compatibility entry point.
install-local-next-task: build-orbit
	@mkdir -p "$(BIN_DIR)"
	@if [ -e "$(BIN_DIR)/local-next-task" ] || [ -L "$(BIN_DIR)/local-next-task" ]; then \
		case "$$(readlink "$(BIN_DIR)/local-next-task")" in \
			orbit|"$(BIN_DIR)/orbit"|*/agentic-coding/scripts/local_next_task.py) ;; \
			*) echo "Refusing to replace unrelated $(BIN_DIR)/local-next-task" >&2; exit 1 ;; \
		esac; \
	fi
	install -m 755 orbit "$(BIN_DIR)/orbit"
	ln -sfn orbit "$(BIN_DIR)/local-next-task"

# Run modernize to update code to modern Go idioms
modernize:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./...
