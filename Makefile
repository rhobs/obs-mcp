# Makefile for obs-mcp server

CONTAINER_CLI ?= docker

.PHONY: help
help: ## Show this help message
	@echo "obs-mcp - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@command -v go >/dev/null 2>&1 || { echo "Error: go is required but not installed."; exit 1; }
	@command -v $(CONTAINER_CLI) >/dev/null 2>&1 || echo "Warning: $(CONTAINER_CLI) is not installed. Container builds will fail."
	@echo "✓ All required tools are installed"

.PHONY: build
build: ## Build obs-mcp binary
	go build -tags strictfipsruntime -o obs-mcp ./cmd/obs-mcp

.PHONY: test-unit
test-unit: ## Run obs-mcp unit tests
	go test -v -race ./...

.PHONY: clean
clean: ## Clean obs-mcp build artifacts
	go clean && rm -f obs-mcp

.PHONY: container
container: build ## Build obs-mcp container image
	$(CONTAINER_CLI) build -f Containerfile -t obs-mcp:latest .

.PHONY: format
format: ## Format all code
	go fmt ./...

.PHONY: setup
setup: check-tools ## Install dependencies for all components
	go mod download
