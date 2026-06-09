# Makefile for obs-mcp server

.DEFAULT_GOAL := run

CONTAINER_CLI ?= docker
IMAGE_NAME ?= ghcr.io/rhobs/obs-mcp
TAG ?= $(shell git rev-parse --short HEAD)
IMAGE_REF ?= $(IMAGE_NAME):$(TAG)
IMAGE ?= $(IMAGE_REF)
ifneq ($(findstring $(origin IMAGE),environment command line),)
$(warning IMAGE is deprecated, use IMAGE_REF instead)
endif
TOOLS_DIR := hack/tools
MCPCHECKER_VERSION ?= 0.0.18

ROOT_DIR := $(shell pwd)
TOOLS_BIN_DIR := $(ROOT_DIR)/tmp/bin
MCPCHECKER := $(TOOLS_BIN_DIR)/mcpchecker

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
	$(CONTAINER_CLI) build --build-arg GOLANG_BUILDER=$(GO_VERSION) --load -f Containerfile -t $(IMAGE_REF) .

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

.PHONY: update-go-deps
update-go-deps: ## Upgrade root Go module dependencies to latest and tidy
	@echo "==> Upgrading root module dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✓ Done. Run 'make test-unit' to verify."

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
TOOLSETS ?= metrics
EXTRA_FLAGS ?= --loki.use-route

.PHONY: run
run: build ## Run obs-mcp in HTTP mode (use LOG_LEVEL=debug to see backend call timings)
	@echo "Tip: Override backend URLs with PROMETHEUS_URL=https://... ALERTMANAGER_URL=https://... make run"
	@echo "Tip: Override toolsets with TOOLSETS=metrics,traces,otelcol make run"
	@echo "Note: AUTH_MODE=serviceaccount or header requires PROMETHEUS_URL and ALERTMANAGER_URL to be set"
	./obs-mcp --listen $(LISTEN_ADDR) --auth-mode $(AUTH_MODE) --insecure --log-level $(LOG_LEVEL) --toolsets $(TOOLSETS) $(EXTRA_FLAGS)
.PHONY: run-no-guardrails
run-no-guardrails: build ## Run obs-mcp in HTTP mode with guardrails disabled
	@echo "Tip: Override backend URLs with PROMETHEUS_URL=https://... ALERTMANAGER_URL=https://... make run-no-guardrails"
	@echo "Note: AUTH_MODE=serviceaccount or header requires PROMETHEUS_URL and ALERTMANAGER_URL to be set"
	./obs-mcp --listen $(LISTEN_ADDR) --auth-mode $(AUTH_MODE) --insecure --log-level $(LOG_LEVEL) --toolsets $(TOOLSETS) --guardrails none $(EXTRA_FLAGS)

.PHONY: run-prometheus
run-prometheus: build ## Run obs-mcp with Prometheus as the metrics backend
	@echo "Tip: Override backend URL with PROMETHEUS_URL=https://... make run-prometheus"
	./obs-mcp --listen $(LISTEN_ADDR) --auth-mode $(AUTH_MODE) --metrics-backend prometheus --insecure --log-level $(LOG_LEVEL) --toolsets $(TOOLSETS) $(EXTRA_FLAGS)


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
export KIND_CLUSTER_NAME
export CONTAINER_CLI
export IMAGE_REF

E2E_PROFILE ?= kind
.PHONY: test-e2e-setup
test-e2e-setup: ## Setup the cluster for E2E tests
	./hack/e2e/setup.sh $(if $(filter kind,$(E2E_PROFILE)),provision) prereqs extras --profile $(E2E_PROFILE)

.PHONY: test-e2e-deploy
test-e2e-deploy: container ## Build and deploy obs-mcp to the cluster
	./hack/e2e/setup.sh upload deploy --profile $(E2E_PROFILE)

.PHONY: test-e2e
test-e2e: ## Run E2E tests (requires cluster to be running)
	go test -mod=mod -v -tags=e2e -timeout=10m ./tests/e2e/...

.PHONY: test-e2e-teardown
test-e2e-teardown: ## Teardown E2E test cluster
	./hack/e2e/setup.sh down --profile $(E2E_PROFILE)

.PHONY: test-e2e-full
test-e2e-full: test-e2e-setup test-e2e-deploy test-e2e test-e2e-teardown ## Run full E2E test cycle (setup, test, teardown)

# OpenShift E2E Testing
# In CI, deploy-obs-mcp step calls test-e2e-openshift-deploy, then the step registry runs test-e2e && test-e2e-openshift.
# CI config:      https://github.com/openshift/release/blob/main/ci-operator/config/rhobs/obs-mcp/rhobs-obs-mcp-main.yaml
# Step registry:  https://github.com/openshift/release/blob/main/ci-operator/step-registry/rhobs/obs-mcp-e2e-tests/rhobs-obs-mcp-e2e-tests-commands.sh
.PHONY: test-e2e-openshift-deploy
test-e2e-openshift-deploy: ## Deploy obs-mcp to OpenShift (uses IMAGE env var from CI)
	# We use IMAGE until we update the CI job to pass IMAGE_REF instead.
	IMAGE_REF="$(IMAGE)" ./hack/e2e/setup.sh prereqs extras deploy --profile openshift

.PHONY: test-e2e-openshift
test-e2e-openshift: ## Run OpenShift route discovery E2E tests (requires oc login)
	go test -mod=mod -v -tags=e2e,openshift -timeout=5m ./tests/e2e/...

MCPCHECKER_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
MCPCHECKER_ARCH := $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

$(MCPCHECKER): | $(TOOLS_BIN_DIR)
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
EVAL_CONFIG ?= eval.yaml

.PHONY: run-mcpchecker-eval
run-mcpchecker-eval: $(MCPCHECKER) ## Run mcpchecker eval (TASK=name, CATEGORY=..., EVAL_CONFIG=eval-logs.yaml, RUNS=3)
ifdef TASK
	cd $(MCPCHECKER_EVAL_DIR) && $(MCPCHECKER) check $(EVAL_CONFIG) --run "$(TASK)" --runs $(RUNS) --verbose
else ifdef CATEGORY
	cd $(MCPCHECKER_EVAL_DIR) && $(MCPCHECKER) check $(EVAL_CONFIG) --label-selector "category=$(CATEGORY)" --runs $(RUNS) --parallel 4
else
	cd $(MCPCHECKER_EVAL_DIR) && $(MCPCHECKER) check $(EVAL_CONFIG) --runs $(RUNS) --parallel 4
endif
	$(MCPCHECKER) result summary $(MCPCHECKER_EVAL_DIR)/mcpchecker-obs-mcp-tools-out.json


# Loki evals and local smoke (OpenShift fixture via hack/e2e/setup.sh --stacks loki)
LOKI_EVAL_HACK_DIR := hack/loki_multitenancy_openshift
LOKI_MCP_PORT ?= 9100
LOKI_MCP_PID_FILE ?= .loki-mcp.pid
LOKI_MCP_HEALTH_TIMEOUT ?= 60
LOKI_MCP_HEALTH_INTERVAL ?= 2
LOKI_MCP_TOOLSETS ?= logs
LOKI_USE_ROUTE ?= true
LOKI_MCP_AUTH_MODE ?= kubeconfig
LOKI_LOCAL_PORT ?= 3100
LOKI_LOCAL_URL ?= http://127.0.0.1:$(LOKI_LOCAL_PORT)
LOKI_LOCAL_IMAGE ?= grafana/loki:3.4.2
LOKI_LOCAL_CONTAINER ?= obs-mcp-loki-local
LOKI_LOCAL_READY_TIMEOUT ?= 60
LOKI_LOCAL_LOG_JOB ?= obs-mcp-local

.PHONY: setup-loki-evals
setup-loki-evals: ## Deploy Loki operator test stack on OpenShift (no obs-mcp required)
	@test -n "$$(oc whoami 2>/dev/null)" || { echo "ERROR: oc login required"; exit 1; }
	./hack/e2e/setup.sh extras --profile openshift --stacks loki

.PHONY: verify-loki-evals
verify-loki-evals: ## Smoke-test Loki MCP tools (run-loki-mcp-server must be running on :9100)
	$(LOKI_EVAL_HACK_DIR)/03_verify.sh

.PHONY: run-loki-mcp-server
run-loki-mcp-server: build ## Start obs-mcp in background for Loki evals (logs toolset, port $(LOKI_MCP_PORT))
	@if [ -f $(LOKI_MCP_PID_FILE) ] && kill -0 $$(cat $(LOKI_MCP_PID_FILE)) 2>/dev/null; then \
		echo "obs-mcp already running (PID $$(cat $(LOKI_MCP_PID_FILE)))"; \
		exit 0; \
	fi
	@echo "Starting obs-mcp on 127.0.0.1:$(LOKI_MCP_PORT) with toolsets=$(LOKI_MCP_TOOLSETS)..."
	@LOKI_ARGS=""; \
	if [ -n "$(LOKI_URL)" ]; then LOKI_ARGS="--loki-url $(LOKI_URL)"; \
	elif [ "$(LOKI_USE_ROUTE)" = "true" ]; then LOKI_ARGS="--loki.use-route"; fi; \
	./obs-mcp --listen 127.0.0.1:$(LOKI_MCP_PORT) --auth-mode $(LOKI_MCP_AUTH_MODE) --insecure \
		--log-level $(LOG_LEVEL) --toolsets $(LOKI_MCP_TOOLSETS) $$LOKI_ARGS \
		>/tmp/loki-mcp-eval.log 2>&1 & echo $$! > $(LOKI_MCP_PID_FILE)
	@elapsed=0; \
	while [ $$elapsed -lt $(LOKI_MCP_HEALTH_TIMEOUT) ]; do \
		if curl -sf "http://127.0.0.1:$(LOKI_MCP_PORT)/health" >/dev/null 2>&1; then \
			echo "obs-mcp ready at http://127.0.0.1:$(LOKI_MCP_PORT)/mcp"; \
			exit 0; \
		fi; \
		sleep $(LOKI_MCP_HEALTH_INTERVAL); \
		elapsed=$$((elapsed + $(LOKI_MCP_HEALTH_INTERVAL))); \
	done; \
	echo "ERROR: obs-mcp failed to start (see /tmp/loki-mcp-eval.log)"; exit 1

.PHONY: stop-loki-mcp-server
stop-loki-mcp-server: ## Stop obs-mcp started by run-loki-mcp-server
	@if [ -f $(LOKI_MCP_PID_FILE) ]; then \
		PID=$$(cat $(LOKI_MCP_PID_FILE)); \
		echo "Stopping obs-mcp (PID: $$PID)"; \
		kill $$PID 2>/dev/null || true; \
		rm -f $(LOKI_MCP_PID_FILE); \
	else \
		echo "No $(LOKI_MCP_PID_FILE) found"; \
	fi

.PHONY: run-loki-evals
run-loki-evals: setup-loki-evals run-loki-mcp-server $(TOOLS_BIN_DIR)/mcpchecker ## Full Loki eval flow: stack + obs-mcp + mcpchecker (needs OPENAI_API_KEY)
	@set -e; \
	trap '$(MAKE) stop-loki-mcp-server' EXIT; \
	$(MAKE) verify-loki-evals; \
	$(MAKE) run-mcpchecker-eval CATEGORY=logs EVAL_CONFIG=eval-logs.yaml; \
	echo ""; \
	echo "Loki evals finished. Target pass rate: >= 80% tasks and assertions."

.PHONY: teardown-loki-evals
teardown-loki-evals: stop-loki-mcp-server ## Remove Loki eval test stack from OpenShift
	oc delete -f $(LOKI_EVAL_HACK_DIR)/03_lokistack.yaml --ignore-not-found
	oc delete -f $(LOKI_EVAL_HACK_DIR)/install/ --ignore-not-found
	oc delete clusterrole/obs-mcp-loki-gateway-read --ignore-not-found

.PHONY: setup-loki-local
setup-loki-local: ## Start Docker Loki and push a test log line (no cluster required)
	@command -v $(CONTAINER_CLI) >/dev/null 2>&1 || { echo "ERROR: $(CONTAINER_CLI) is required"; exit 1; }
	@if curl -sf "$(LOKI_LOCAL_URL)/ready" >/dev/null 2>&1; then \
		echo "Loki already reachable at $(LOKI_LOCAL_URL), pushing test log only"; \
	elif $(CONTAINER_CLI) inspect $(LOKI_LOCAL_CONTAINER) >/dev/null 2>&1; then \
		echo "Starting existing container $(LOKI_LOCAL_CONTAINER)..."; \
		$(CONTAINER_CLI) start $(LOKI_LOCAL_CONTAINER) >/dev/null; \
	else \
		$(CONTAINER_CLI) rm -f $(LOKI_LOCAL_CONTAINER) >/dev/null 2>&1 || true; \
		echo "Starting $(LOKI_LOCAL_IMAGE) as $(LOKI_LOCAL_CONTAINER) on $(LOKI_LOCAL_URL)..."; \
		if ! $(CONTAINER_CLI) run -d --name $(LOKI_LOCAL_CONTAINER) \
			-p $(LOKI_LOCAL_PORT):3100 $(LOKI_LOCAL_IMAGE) >/dev/null 2>&1; then \
			echo "ERROR: failed to start Loki on port $(LOKI_LOCAL_PORT) (in use?). Try: LOKI_LOCAL_PORT=3310 make setup-loki-local"; \
			exit 1; \
		fi; \
	fi
	@elapsed=0; \
	while [ $$elapsed -lt $(LOKI_LOCAL_READY_TIMEOUT) ]; do \
		if curl -sf "$(LOKI_LOCAL_URL)/ready" >/dev/null 2>&1; then \
			echo "Loki ready at $(LOKI_LOCAL_URL)"; \
			break; \
		fi; \
		sleep 2; \
		elapsed=$$((elapsed + 2)); \
	done; \
	if ! curl -sf "$(LOKI_LOCAL_URL)/ready" >/dev/null 2>&1; then \
		echo "ERROR: Loki did not become ready (see: $(CONTAINER_CLI) logs $(LOKI_LOCAL_CONTAINER))"; \
		exit 1; \
	fi
	LOKI_URL="$(LOKI_LOCAL_URL)" LOKI_LOG_JOB="$(LOKI_LOCAL_LOG_JOB)" \
		$(ROOT_DIR)/hack/loki_local/push_test_log.sh

.PHONY: stop-loki-local
stop-loki-local: ## Stop and remove the local Docker Loki container
	@$(CONTAINER_CLI) rm -f $(LOKI_LOCAL_CONTAINER) >/dev/null 2>&1 || true
	@echo "Stopped $(LOKI_LOCAL_CONTAINER) (if it was running)"

.PHONY: run-loki-mcp-local
run-loki-mcp-local: ## Start obs-mcp for local Docker Loki (header auth, logs toolset)
	@$(MAKE) run-loki-mcp-server LOKI_MCP_TOOLSETS=logs LOKI_MCP_AUTH_MODE=header \
		LOKI_URL=$(LOKI_LOCAL_URL) LOKI_USE_ROUTE=false

.PHONY: verify-loki-local
verify-loki-local: ## Smoke-test Loki MCP tools against Docker Loki (run-loki-mcp-local on :9100)
	LOKI_URL="$(LOKI_LOCAL_URL)" LOKI_LOG_JOB="$(LOKI_LOCAL_LOG_JOB)" \
		$(ROOT_DIR)/hack/loki_local/verify.sh

.PHONY: run-loki-local-smoke
run-loki-local-smoke: setup-loki-local run-loki-mcp-local ## Docker Loki + obs-mcp + verify (no OpenShift, no API key)
	@set -e; \
	trap '$(MAKE) stop-loki-mcp-server stop-loki-local' EXIT; \
	$(MAKE) verify-loki-local; \
	echo ""; \
	echo "Local Loki smoke OK. For agent evals use OpenShift: make run-loki-evals (needs OPENAI_API_KEY)."
