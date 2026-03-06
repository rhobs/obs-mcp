# Testing

This document describes how to run tests for obs-mcp.

## Linting

Run golangci-lint to check code quality:

```bash
make lint
```

To automatically fix issues:

```bash
make lint-fix
```

## Unit Tests

Run unit tests with:

```bash
make test-unit
```

## Manual Testing

Use the Makefile run targets to start the server locally for manual testing with curl or an MCP client.

### Start the server in HTTP mode

```bash
make run
```

This builds the binary and starts the server on `:9100` with `kubeconfig` auth, debug logging, and TLS verification disabled. Override defaults as needed:

```bash
LISTEN_ADDR=:8080 AUTH_MODE=header LOG_LEVEL=info make run
```

To point at a specific Prometheus/Alertmanager instance:

```bash
PROMETHEUS_URL=https://thanos.example.com ALERTMANAGER_URL=https://alertmanager.example.com make run
```

### Start the server with guardrails disabled

Useful when testing against Thanos versions before v0.40.0 (which don't expose `/api/v1/status/tsdb`):

```bash
make run-no-guardrails
```

### Structured log output

With `LOG_LEVEL=debug` (the default for make targets), every backend API call logs timing and result information:

```bash
level=debug msg="Backend call completed" backend=prometheus operation=list_metrics duration_ms=42 result_count=1523
level=warn msg="Guardrail rejected query" guardrail=disallow-blanket-regex query="up{job=~\".+\"}" error="..."
```

This is useful for spotting slow backend calls, guardrail rejections, and backend errors without any additional tooling.

## End-to-End (E2E) Tests

E2E tests validate obs-mcp against a real Kubernetes cluster with Prometheus.

### Prerequisites

- [Go](https://golang.org/dl/) 1.24+
- [Docker](https://docs.docker.com/get-docker/) or [Podman](https://podman.io/)
- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

### Running E2E Tests

**To use Podman instead of Docker, export the following:**

```bash
export CONTAINER_CLI=podman
```

#### Full Test Cycle

Run setup, deploy, test, and teardown in one command:

```bash
make test-e2e-full
```

#### Step-by-Step (Recommended for Development)

1. **Setup Kind cluster with kube-prometheus:**

```bash
make test-e2e-setup
```

This creates a Kind cluster and installs Prometheus Operator, Prometheus, and Alertmanager.

2. **Build and deploy obs-mcp:**

```bash
make test-e2e-deploy
```

3. **Run E2E tests:**

```bash
make test-e2e
```

4. **Teardown (when done):**

```bash
make test-e2e-teardown
```

## OpenShift E2E Tests

OpenShift-specific tests run against real OpenShift in-cluster monitoring. They use the `e2e,openshift` build tags and are kept separate from the Kind-based tests.

In CI, obs-mcp is built and deployed first (`make test-e2e-openshift-deploy`), then the tests run.

### Prerequisites

- Active `oc login` session with cluster-admin or monitoring access
- obs-mcp deployed to the cluster:
  ```bash
  IMAGE=<image> make test-e2e-openshift-deploy
  ```

### What is tested

| Test | Description |
|------|-------------|
| `TestRouteDiscovery_ThanosQuerier` | Discovers `thanos-querier` route in `openshift-monitoring` |
| `TestRouteDiscovery_PrometheusK8s` | Discovers `prometheus-k8s` route in `openshift-monitoring` |
| `TestRouteDiscovery_Alertmanager` | Discovers `alertmanager-main` route in `openshift-monitoring` |
| `TestRouteDiscovery_URLsAreReachable` | Verifies discovered URLs respond to HTTP requests (401 is acceptable) |
| `TestOpenShiftMetricsPresent` | Confirms `cluster_version` metric is reachable through obs-mcp (OpenShift-only metric) |

General tool correctness (instant query, range query, alerts, guardrails) is covered by the Kind-based `make test-e2e` suite and is not duplicated here.

### Running

```bash
make test-e2e-openshift
```

> [!NOTE]
> Route discovery tests (`TestRouteDiscovery_*`) call `pkg/k8s` directly using your local kubeconfig. `TestOpenShiftMetricsPresent` requires obs-mcp to be deployed and reachable.
