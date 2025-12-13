# Makefile for obs-mcp server

CONTAINER_CLI ?= docker
TOOLS_DIR := hack/tools

ROOT_DIR := $(shell pwd)
TOOLS_BIN_DIR := $(ROOT_DIR)/tmp/bin

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

$(TOOLS_BIN_DIR):
	mkdir -p $(TOOLS_BIN_DIR)

$(TOOLS_BIN_DIR)/golangci-lint: $(TOOLS_DIR)/go.mod | $(TOOLS_BIN_DIR)
	cd $(TOOLS_DIR) && go build -o $(TOOLS_BIN_DIR)/golangci-lint github.com/golangci/golangci-lint/v2/cmd/golangci-lint

.PHONY: lint
lint: $(TOOLS_BIN_DIR)/golangci-lint ## Run golangci-lint
	$(TOOLS_BIN_DIR)/golangci-lint run --timeout=10m ./...

.PHONY: lint-fix
lint-fix: $(TOOLS_BIN_DIR)/golangci-lint ## Run golangci-lint with fix
	$(TOOLS_BIN_DIR)/golangci-lint run --timeout=10m --fix ./...

.PHONY: setup
setup: check-tools ## Install dependencies for all components
	go mod download
	cd $(TOOLS_DIR) && go mod download
