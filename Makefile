.PHONY: help check fmt vet test test-race build

# Self-documenting: add ## comment after a target to include it in make help.
help: ## Show available targets
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-12s %s\n", $$1, $$2}'

check: fmt vet test-race ## Run all quality checks (fmt, vet, test with race detector)

fmt: ## Check formatting
	gofmt -l .

vet: ## Run go vet
	go vet ./...

test: ## Run tests
	go test ./...

test-race: ## Run tests with race detector (requires C compiler)
	CGO_ENABLED=1 go test -race ./...

build: ## Build the afk binary
	go build ./cmd/afk
