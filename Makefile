.PHONY: help check fmt vet test build

# Self-documenting: add ## comment after a target to include it in make help.
help: ## Show available targets
	@grep -E '^[a-z]+:.*##' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

check: fmt vet test ## Run all quality checks (fmt, vet, test)

fmt: ## Check formatting
	gofmt -l .

vet: ## Run go vet
	go vet ./...

test: ## Run tests
	go test ./...

build: ## Build the afk binary
	go build ./cmd/afk
