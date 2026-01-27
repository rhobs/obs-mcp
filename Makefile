# Makefile for obs-mcp server

CONTAINER_CLI ?= docker
IMAGE ?= ghcr.io/rhobs/obs-mcp
TAG ?= $(shell git rev-parse --short HEAD)
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
	$(CONTAINER_CLI) build --load -f Containerfile -t $(IMAGE):$(TAG) .

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

.PHONY: generate-tools-doc
generate-tools-doc: ## Generate TOOLS.md from tool definitions
	go run ./cmd/generate-tools-doc/main.go

.PHONY: check-tools-doc
check-tools-doc: generate-tools-doc ## Check if TOOLS.md is up to date
	@git diff --exit-code TOOLS.md || { \
		echo ""; \
		echo "❌ TOOLS.md is out of sync with tool definitions!"; \
		echo ""; \
		echo "To fix, run: make generate-tools-doc"; \
		echo "Then commit the updated TOOLS.md"; \
		echo ""; \
		exit 1; \
	}

# E2E Testing
KIND_CLUSTER_NAME ?= obs-mcp-e2e

.PHONY: test-e2e-setup
test-e2e-setup: ## Setup Kind cluster with kube-prometheus for E2E tests
	chmod +x hack/e2e/setup-cluster.sh
	CLUSTER_NAME=$(KIND_CLUSTER_NAME) ./hack/e2e/setup-cluster.sh

.PHONY: test-e2e-images
test-e2e-images: container ## Build and load obs-mcp image into Kind cluster
ifeq ($(CONTAINER_CLI),podman)
	mkdir -p tmp
	$(CONTAINER_CLI) save --quiet -o tmp/obs-mcp.tar $(IMAGE):$(TAG)
	kind load image-archive --name $(KIND_CLUSTER_NAME) tmp/obs-mcp.tar
	rm -f tmp/obs-mcp.tar
else
	kind load docker-image --name $(KIND_CLUSTER_NAME) $(IMAGE):$(TAG)
endif

.PHONY: test-e2e-deploy
test-e2e-deploy: test-e2e-images ## Deploy obs-mcp to Kind cluster
	kubectl apply -f manifests/kubernetes/
	kubectl set image deployment/obs-mcp -n obs-mcp obs-mcp=$(IMAGE):$(TAG)
	kubectl apply -f hack/e2e/manifests/network_policy_to_access_prometheus.yaml
	kubectl -n obs-mcp rollout status deployment/obs-mcp --timeout=3m

.PHONY: test-e2e
test-e2e: ## Run E2E tests (requires cluster to be running)
	go test -v -tags=e2e -timeout=10m ./tests/e2e/...

.PHONY: test-e2e-teardown
test-e2e-teardown: ## Teardown E2E test cluster
	chmod +x hack/e2e/teardown-cluster.sh
	CLUSTER_NAME=$(KIND_CLUSTER_NAME) ./hack/e2e/teardown-cluster.sh

.PHONY: test-e2e-full
test-e2e-full: test-e2e-setup test-e2e-deploy test-e2e test-e2e-teardown ## Run full E2E test cycle (setup, test, teardown)
