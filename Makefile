.PHONY: help test test-unit test-integration test-cover lint build install clean fmt vet tidy examples

.DEFAULT_GOAL := help

help: ## Display this help message
	@echo "rentfree - Multi-layer caching library"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

install: ## Download dependencies
	@echo "==> Installing dependencies..."
	go mod download
	go mod verify

tidy: ## Tidy go.mod and go.sum
	@echo "==> Tidying dependencies..."
	go mod tidy

fmt: ## Format code with gofmt
	@echo "==> Formatting code..."
	gofmt -s -w .

vet: ## Run go vet
	@echo "==> Running go vet..."
	go vet ./...

lint: ## Run golangci-lint
	@echo "==> Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

test: ## Run all tests
	@echo "==> Running tests..."
	go test -v -race -cover ./...

test-unit: ## Run unit tests only (skip integration tests)
	@echo "==> Running unit tests..."
	go test -v -race -short ./...

test-integration: ## Run integration tests (requires Docker)
	@echo "==> Starting Redis container..."
	@docker run -d --name rentfree-redis-test -p 6379:6379 redis:alpine || true
	@sleep 2
	@echo "==> Running integration tests..."
	@go test -v -race -run "Redis|Integration" ./internal/cache/... || (docker stop rentfree-redis-test && docker rm rentfree-redis-test && exit 1)
	@echo "==> Stopping Redis container..."
	@docker stop rentfree-redis-test
	@docker rm rentfree-redis-test

test-cover: ## Run tests with coverage report
	@echo "==> Running tests with coverage..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo ""
	@echo "==> Coverage summary:"
	go tool cover -func=coverage.txt | tail -1

coverage-html: test-cover ## Generate HTML coverage report
	@echo "==> Generating HTML coverage report..."
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report saved to coverage.html"

build: ## Build all packages
	@echo "==> Building packages..."
	go build ./...

examples: ## Run basic example
	@echo "==> Running basic example..."
	cd examples/basic && go run main.go

bench: ## Run benchmarks
	@echo "==> Running benchmarks..."
	go test -bench=. -benchmem -benchtime=1s ./...

bench-all: ## Run all benchmarks with more iterations
	@echo "==> Running comprehensive benchmarks..."
	go test -bench=. -benchmem -benchtime=3s -count=3 ./...

clean: ## Clean build artifacts and test files
	@echo "==> Cleaning..."
	rm -f coverage.txt coverage.html
	go clean -cache -testcache
	@docker stop rentfree-redis-test 2>/dev/null || true
	@docker rm rentfree-redis-test 2>/dev/null || true

ci: install lint test ## Run CI checks locally

check: fmt vet lint test ## Run all checks

.PHONY: release-check
release-check: ## Check if ready for release
	@echo "==> Pre-release checks..."
	@echo "Running linter..."
	@$(MAKE) lint
	@echo "Running all tests..."
	@$(MAKE) test
	@echo "Verifying dependencies..."
	@go mod verify
	@echo ""
	@echo "✓ All checks passed! Ready for release."
