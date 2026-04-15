# Makefile for obs-mcp server

.DEFAULT_GOAL := run

CONTAINER_CLI ?= docker
IMAGE ?= ghcr.io/rhobs/obs-mcp
TAG ?= $(shell git rev-parse --short HEAD)
TOOLS_DIR := hack/tools
MCPCHECKER_VERSION ?= 0.0.15

ROOT_DIR := $(shell pwd)
TOOLS_BIN_DIR := $(ROOT_DIR)/tmp/bin

.PHONY: help
help: ## Show this help message
	@echo "obs-mcp - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@command -v go >/dev/null 2>&1 || { echo "Error: go is required but not installed."; exit 1; }
	@command -v $(CONTAINER_CLI) >/dev/null 2>&1 || echo "Warning: $(CONTAINER_CLI) is not installed. Container builds will fail."
	@echo "✓ All required tools are installed"

.PHONY: build
build: ## Build obs-mcp binary for local development
	go build -mod=mod -tags strictfipsruntime -o obs-mcp ./cmd/obs-mcp

.PHONY: build-linux
build-linux: ## Build obs-mcp binary for linux/amd64
	GOOS=linux GOARCH=amd64 go build -mod=mod -tags strictfipsruntime -o obs-mcp ./cmd/obs-mcp

.PHONY: test-unit
test-unit: ## Run obs-mcp unit tests
	go test -mod=mod -v -race ./...

.PHONY: clean
clean: ## Clean obs-mcp build artifacts
	go clean && rm -f obs-mcp

.PHONY: tag
tag: ## Create a release tag (usage: make tag VERSION=0.1.0)
ifndef VERSION
	$(error VERSION is required. Usage: make tag VERSION=0.1.0)
endif
	git tag -s "v$(VERSION)" -m "v$(VERSION)"
	@echo "Tag v$(VERSION) created."

GO_VERSION := $(shell awk '/^go /{print $$2}' go.mod | cut -d. -f1,2)

.PHONY: container
container: build-linux ## Build obs-mcp container image
	$(CONTAINER_CLI) build --build-arg GOLANG_BUILDER=$(GO_VERSION) --load -f Containerfile -t $(IMAGE):$(TAG) .

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

# Run targets - for local testing
LISTEN_ADDR ?= :9100
LOG_LEVEL ?= debug
AUTH_MODE ?= kubeconfig

.PHONY: run
run: build ## Run obs-mcp in HTTP mode (use LOG_LEVEL=debug to see backend call timings)
	@echo "Tip: Override backend URLs with PROMETHEUS_URL=https://... ALERTMANAGER_URL=https://... make run"
	@echo "Note: AUTH_MODE=serviceaccount or header requires PROMETHEUS_URL and ALERTMANAGER_URL to be set"
	./obs-mcp --listen $(LISTEN_ADDR) --auth-mode $(AUTH_MODE) --insecure --log-level $(LOG_LEVEL)

.PHONY: run-no-guardrails
run-no-guardrails: build ## Run obs-mcp in HTTP mode with guardrails disabled
	@echo "Tip: Override backend URLs with PROMETHEUS_URL=https://... ALERTMANAGER_URL=https://... make run-no-guardrails"
	@echo "Note: AUTH_MODE=serviceaccount or header requires PROMETHEUS_URL and ALERTMANAGER_URL to be set"
	./obs-mcp --listen $(LISTEN_ADDR) --auth-mode $(AUTH_MODE) --insecure --log-level $(LOG_LEVEL) --guardrails none

.PHONY: run-prometheus
run-prometheus: build ## Run obs-mcp with Prometheus as the metrics backend
	@echo "Tip: Override backend URL with PROMETHEUS_URL=https://... make run-prometheus"
	./obs-mcp --listen $(LISTEN_ADDR) --auth-mode $(AUTH_MODE) --metrics-backend prometheus --insecure --log-level $(LOG_LEVEL)


.PHONY: pf-alertmanager
pf-alertmanager: ## Port-forward alertmanager-main-0:9093 in background (prerequisite for pf targets)
	@oc port-forward -n openshift-monitoring pod/alertmanager-main-0 9093:9093 &
	@sleep 2

.PHONY: run-openshift-pf-prometheus
run-openshift-pf-prometheus: build pf-alertmanager ## Port-forward prometheus-k8s-0:9090 + alertmanager-main-0:9093 and start obs-mcp with header auth (requires oc login)
	@echo "Port-forwarding prometheus-k8s-0:9090..."
	@oc port-forward -n openshift-monitoring pod/prometheus-k8s-0 9090:9090 & \
		PF_PID=$$!; \
		sleep 2; \
		trap "kill $$PF_PID 2>/dev/null" EXIT; \
		PROMETHEUS_URL=http://localhost:9090 ALERTMANAGER_URL=http://localhost:9093 \
		./obs-mcp --listen $(LISTEN_ADDR) --auth-mode header --log-level $(LOG_LEVEL)

.PHONY: inspect
inspect: COMPOSE_HOST_GATEWAY = $(if $(filter podman,$(CONTAINER_CLI)),host.containers.internal,host.docker.internal)
inspect: ## Start obs-mcp + MCP Inspector via compose (port-forward Prometheus/Alertmanager first)
	CONTAINER_HOST_GATEWAY=$(COMPOSE_HOST_GATEWAY) $(CONTAINER_CLI) compose -f compose.yaml up --build

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
	go test -mod=mod -v -tags=e2e -timeout=10m ./tests/e2e/...

.PHONY: test-e2e-teardown
test-e2e-teardown: ## Teardown E2E test cluster
	chmod +x hack/e2e/teardown-cluster.sh
	CLUSTER_NAME=$(KIND_CLUSTER_NAME) ./hack/e2e/teardown-cluster.sh

MCPCHECKER_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
MCPCHECKER_ARCH := $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

$(TOOLS_BIN_DIR)/mcpchecker: | $(TOOLS_BIN_DIR)
	@echo "==> Installing mcpchecker v$(MCPCHECKER_VERSION) ($(MCPCHECKER_OS)/$(MCPCHECKER_ARCH))..."
	@curl -fsSL -o $(TOOLS_BIN_DIR)/mcpchecker.zip \
		https://github.com/mcpchecker/mcpchecker/releases/download/v$(MCPCHECKER_VERSION)/mcpchecker-$(MCPCHECKER_OS)-$(MCPCHECKER_ARCH).zip
	@unzip -o -q $(TOOLS_BIN_DIR)/mcpchecker.zip -d $(TOOLS_BIN_DIR)
	@rm -f $(TOOLS_BIN_DIR)/mcpchecker.zip
	@chmod +x $(TOOLS_BIN_DIR)/mcpchecker
	@echo "✓ mcpchecker v$(MCPCHECKER_VERSION) installed to $(TOOLS_BIN_DIR)/mcpchecker"

.PHONY: install-mcpchecker
install-mcpchecker: $(TOOLS_BIN_DIR)/mcpchecker ## Install mcpchecker CLI for running evals

MCPCHECKER_EVAL_DIR := evals/mcpchecker
RUNS ?= 1

.PHONY: run-mcpchecker-eval
run-mcpchecker-eval: $(TOOLS_BIN_DIR)/mcpchecker ## Run mcpchecker eval (TASK=name, CATEGORY=queries, RUNS=3 for consistency testing)
ifdef TASK
	cd $(MCPCHECKER_EVAL_DIR) && $(TOOLS_BIN_DIR)/mcpchecker check eval.yaml --run "$(TASK)" --runs $(RUNS) --verbose
else ifdef CATEGORY
	cd $(MCPCHECKER_EVAL_DIR) && $(TOOLS_BIN_DIR)/mcpchecker check eval.yaml --label-selector "category=$(CATEGORY)" --runs $(RUNS) --parallel 4
else
	cd $(MCPCHECKER_EVAL_DIR) && $(TOOLS_BIN_DIR)/mcpchecker check eval.yaml --runs $(RUNS) --parallel 4
endif

.PHONY: deploy-kube-state-metrics
deploy-kube-state-metrics: ## Deploy kube-state-metrics from kube-prometheus (for mcpchecker evals)
	@if [ ! -d "tmp/kube-prometheus" ]; then \
		echo "Error: tmp/kube-prometheus not found. Run 'make test-e2e-setup' first."; exit 1; \
	fi
	@echo "==> Installing kube-state-metrics..."
	@for f in tmp/kube-prometheus/manifests/kubeStateMetrics-*.yaml; do \
		kubectl apply -f "$$f"; \
	done
	kubectl -n monitoring rollout status deployment/kube-state-metrics --timeout=3m

.PHONY: deploy-node-exporter
deploy-node-exporter: ## Deploy node-exporter from kube-prometheus (for mcpchecker evals)
	@if [ ! -d "tmp/kube-prometheus" ]; then \
		echo "Error: tmp/kube-prometheus not found. Run 'make test-e2e-setup' first."; exit 1; \
	fi
	@echo "==> Installing node-exporter..."
	@for f in tmp/kube-prometheus/manifests/nodeExporter-*.yaml; do \
		kubectl apply -f "$$f"; \
	done
	kubectl -n monitoring rollout status daemonset/node-exporter --timeout=3m

.PHONY: deploy-kubelet-servicemonitors
deploy-kubelet-servicemonitors: ## Deploy kubelet/cAdvisor scrape configs from kube-prometheus (for container_* metrics)
	@if [ ! -d "tmp/kube-prometheus" ]; then \
		echo "Error: tmp/kube-prometheus not found. Run 'make test-e2e-setup' first."; exit 1; \
	fi
	@echo "==> Installing kubelet/cAdvisor ServiceMonitors..."
	@for f in tmp/kube-prometheus/manifests/kubernetesControlPlane-*.yaml; do \
		kubectl apply -f "$$f"; \
	done

.PHONY: deploy-more-kube-prom-targets
deploy-more-kube-prom-targets: deploy-kube-state-metrics deploy-node-exporter deploy-kubelet-servicemonitors ## Deploy additional kube-prometheus scrape targets (kube-state-metrics, node-exporter, kubelet)

.PHONY: test-e2e-full
test-e2e-full: test-e2e-setup test-e2e-deploy test-e2e test-e2e-teardown ## Run full E2E test cycle (setup, test, teardown)

# OpenShift E2E Testing
# In CI, deploy-obs-mcp step calls test-e2e-openshift-deploy, then the step registry runs test-e2e && test-e2e-openshift.
# CI config:      https://github.com/openshift/release/blob/main/ci-operator/config/rhobs/obs-mcp/rhobs-obs-mcp-main.yaml
# Step registry:  https://github.com/openshift/release/blob/main/ci-operator/step-registry/rhobs/obs-mcp-e2e-tests/rhobs-obs-mcp-e2e-tests-commands.sh
.PHONY: test-e2e-openshift-deploy
test-e2e-openshift-deploy: ## Deploy obs-mcp to OpenShift (uses IMAGE env var from CI)
	oc apply -f manifests/openshift_e2e/
	oc set image deployment/obs-mcp -n obs-mcp obs-mcp=$(IMAGE)
	oc -n obs-mcp rollout status deployment/obs-mcp --timeout=3m

.PHONY: test-e2e-openshift
test-e2e-openshift: ## Run OpenShift route discovery E2E tests (requires oc login)
	go test -mod=mod -v -tags=e2e,openshift -timeout=5m ./tests/e2e/...

