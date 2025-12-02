# Root Makefile for genie-plugin project
# Builds all components: obs-mcp, layout-manager, and dynamic-plugin

CONTAINER_CLI ?= docker

.PHONY: help
help: ## Show this help message
	@echo "Genie Plugin - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@command -v go >/dev/null 2>&1 || { echo "Error: go is required but not installed."; exit 1; }
	@command -v yarn >/dev/null 2>&1 || { echo "Error: yarn is required but not installed."; exit 1; }
	@command -v $(CONTAINER_CLI) >/dev/null 2>&1 || echo "Warning: $(CONTAINER_CLI) is not installed. Container builds will fail."
	@echo "✓ All required tools are installed"

# ============================================================================
# Component: obs-mcp
# ============================================================================

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

# ============================================================================
# Component: layout-manager
# ============================================================================

.PHONY: layout-manager-build
layout-manager-build: ## Build layout-manager binary
	@cd layout-manager && $(MAKE) build

.PHONY: layout-manager-test
layout-manager-test: ## Run layout-manager tests
	@cd layout-manager && $(MAKE) test

.PHONY: layout-manager-clean
layout-manager-clean: ## Clean layout-manager build artifacts
	@cd layout-manager && $(MAKE) clean

.PHONY: layout-manager-container
layout-manager-container: layout-manager-build ## Build layout-manager container image
	@cd layout-manager && CONTAINER_CLI=$(CONTAINER_CLI) $(MAKE) docker-build

# ============================================================================
# Component: dynamic-plugin
# ============================================================================

.PHONY: dynamic-plugin-build
dynamic-plugin-build: ## Build dynamic-plugin
	@cd dynamic-plugin && yarn install && yarn build

.PHONY: dynamic-plugin-test
dynamic-plugin-test: ## Run dynamic-plugin tests
	@cd dynamic-plugin && yarn lint

.PHONY: dynamic-plugin-clean
dynamic-plugin-clean: ## Clean dynamic-plugin build artifacts
	@cd dynamic-plugin && yarn clean

.PHONY: dynamic-plugin-container
dynamic-plugin-container: dynamic-plugin-build ## Build dynamic-plugin container image
	@cd dynamic-plugin && $(CONTAINER_CLI) build -f Dockerfile -t dynamic-plugin:latest .

# ============================================================================
# Aggregate targets
# ============================================================================

.PHONY: build
build: obs-mcp-build layout-manager-build dynamic-plugin-build ## Build all components

.PHONY: test
test: obs-mcp-test layout-manager-test dynamic-plugin-test ## Run tests for all components

.PHONY: clean
clean: obs-mcp-clean layout-manager-clean dynamic-plugin-clean ## Clean all build artifacts

.PHONY: container
container: obs-mcp-container layout-manager-container dynamic-plugin-container ## Build all container images

.PHONY: format
format: ## Format all code
	@cd obs-mcp && go fmt ./...
	@cd layout-manager && $(MAKE) fmt
	@cd dynamic-plugin && yarn lint --fix

.PHONY: setup
setup: check-tools ## Install dependencies for all components
	@cd obs-mcp && go mod download
	@cd layout-manager && $(MAKE) deps
	@cd dynamic-plugin && yarn install
