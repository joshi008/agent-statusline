BIN := bin/agent-statusline
LDFLAGS := -s -w -X github.com/joshi008/agent-statusline/internal/cli.Version=dev -X github.com/joshi008/agent-statusline/internal/cli.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo none) -X github.com/joshi008/agent-statusline/internal/cli.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: build test lint bench
build:
	go build -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/agent-statusline
test:
	go test ./... -count=1
bench:
	go test ./internal/engine -bench . -benchmem -run '^$$'
lint:
	go vet ./...
