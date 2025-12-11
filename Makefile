# Makefile for obs-mcp server

CONTAINER_CLI ?= docker

.PHONY: help
help: ## Show this help message
	@echo "Genie Plugin - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@command -v go >/dev/null 2>&1 || { echo "Error: go is required but not installed."; exit 1; }
	@command -v $(CONTAINER_CLI) >/dev/null 2>&1 || echo "Warning: $(CONTAINER_CLI) is not installed. Container builds will fail."
	@echo "✓ All required tools are installed"

.PHONY: obs-mcp-build
obs-mcp-build: ## Build obs-mcp binary
	@cd obs-mcp && go build -tags strictfipsruntime -o obs-mcp ./cmd/obs-mcp

.PHONY: obs-mcp-test
obs-mcp-test: ## Run obs-mcp tests
	@cd obs-mcp && go test -v -race ./...

.PHONY: obs-mcp-clean
obs-mcp-clean: ## Clean obs-mcp build artifacts
	@cd obs-mcp && go clean && rm -f obs-mcp/obs-mcp

.PHONY: obs-mcp-container
obs-mcp-container: obs-mcp-build ## Build obs-mcp container image
	@cd obs-mcp && $(CONTAINER_CLI) build -f Containerfile -t obs-mcp:latest .

.PHONY: format
format: ## Format all code
	@cd obs-mcp && go fmt ./...

.PHONY: setup
setup: check-tools ## Install dependencies for all components
	@cd obs-mcp && go mod download
