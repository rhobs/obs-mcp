# Deployment Guide

This guide covers authentication modes and deploying obs-mcp on Kubernetes/OpenShift clusters.

## Authentication Modes

The `--auth-mode` flag controls how obs-mcp obtains bearer tokens for **Prometheus/Thanos** and, when the **traces** toolset is enabled, for **Tempo** gateways:

| Mode             | Token Source                                                                   | Use Case                                              |
|------------------|--------------------------------------------------------------------------------|-------------------------------------------------------|
| `kubeconfig`     | Bearer token from `~/.kube/config`                                             | Local development, accessing cluster via routes       |
| `serviceaccount` | Pod's mounted token at `/var/run/secrets/kubernetes.io/serviceaccount/token`   | In-cluster deployment on OpenShift/Kubernetes         |
| `header`         | Forwarded from incoming MCP request's `Authorization` header                   | Pass-through auth or when Prometheus doesn't require auth |

### `kubeconfig` mode

- Extracts the bearer token from your local kubeconfig
- **Auto-discovers** Prometheus/Thanos routes in OpenShift (only mode with auto-discovery)
- Requires token-based auth (`oc whoami -t` must return a token)
- Best for: **Local development** when logged into a cluster

### `serviceaccount` mode

- Reads the service account token mounted inside the pod
- Requires explicit `PROMETHEUS_URL` (no auto-discovery)
- The ServiceAccount must have RBAC permissions to query the metrics endpoint
- Best for: **In-cluster deployment** on OpenShift with RBAC-protected Thanos/Prometheus

### `header` mode

- Forwards the `Authorization` header from incoming MCP client requests to Prometheus
- If no header is provided, connects without authentication
- Requires explicit `PROMETHEUS_URL` (no auto-discovery)
- Best for: **Pass-through auth** scenarios or **Prometheus without authentication** (e.g., port-forwarded, local kube-prometheus)

## Deploying on a Cluster

Example manifests are provided in the `manifests/` directory, organised by stack:

- `manifests/core/deploy/kubernetes/` — Core MCP server for Kubernetes (apply with `kubectl apply -k`)
- `manifests/core/deploy/openshift/` — Core MCP server for OpenShift (apply with `kubectl apply -k`)
- `manifests/prometheus/deploy/kubernetes/` — NetworkPolicies for kube-prometheus access
- `manifests/prometheus/deploy/openshift/` — RBAC for OpenShift monitoring access
- `manifests/tempo/deploy/` — Tracing RBAC (platform-independent)

These are **reference examples** that you'll need to customize for your environment.

### Using a helper script

`hack/e2e/setup.sh` automates cluster setup for E2E testing and development. It accepts a
**profile**, a set of **stacks**, and a **phase expression**:

```bash
hack/e2e/setup.sh [PHASE_EXP] [--profile PROFILE] [--stacks STACKS]
```

**Profiles** select the target cluster type:

| Profile    | Description |
|------------|-------------|
| `kind`     | Local [Kind](https://kind.sigs.k8s.io/) cluster (default). Provisions and unprovisions the cluster automatically. |
| `k8s`      | Generic upstream Kubernetes cluster (must already exist). |
| `openshift`| OpenShift cluster (must already exist). Uses `oc` instead of `kubectl`. |

**Stacks** are optional observability backends that obs-mcp connects to. Pass a
comma-separated list via `--stacks` (default: `prometheus,tempo`):

| Stack        | What it installs | Toolset enabled |
|--------------|------------------|-----------------|
| `prometheus` | kube-prometheus (k8s) or uses the built-in OpenShift monitoring stack | `metrics` |
| `tempo`      | Tempo + OpenTelemetry operators and a sample tracing app | `traces` |

The enabled stacks determine which `manifests/` subtrees are applied and which `--toolsets`
value is passed to the obs-mcp deployment — no manual editing of manifests is needed.
When deploying via `hack/e2e/setup.sh`, the `otelcol` toolset is always included (it has no
external backend dependency); stack selection adds `metrics` and/or `traces` on top.

**Phases** express what work to perform. The two top-level aliases cover the common cases:

| Alias | Expands to |
|-------|------------|
| `up` (default) | `provision` (kind only) → `prereqs` → `extras` → `upload` → `deploy` |
| `down` | `clean` → `unprovision` (kind only) |

Individual phases can be named explicitly (space-separated) to re-run just part of the
sequence, e.g. `hack/e2e/setup.sh deploy --profile kind`:

| Phase        | Description |
|--------------|-------------|
| `provision`  | Creates the Kind cluster (kind profile only). |
| `prereqs`    | Installs cluster-wide operators and CRDs needed by the enabled stacks (cert-manager, Tempo/OTel operators, kube-prometheus CRDs). |
| `extras`     | Adds sources of observability signal for better e2e testing. |
| `upload`     | Builds/loads or pushes the obs-mcp container image into the cluster. |
| `deploy`     | Deploys the obs-mcp to the cluster. |
| `clean`      | Removes temporary artefacts. |
| `unprovision`| Deletes the Kind cluster (kind profile only). |

**Manifests directory structure** mirrors the phases and stacks:

```
manifests/
├── core/$PHASE/{base,kubernetes,openshift}/    # common manifests for obs-mcp
└── $STACK/$PHASE/{base,kubernetes,openshift}/  # stack-specific manifests
```

### Key Configuration

When deploying in-cluster, you must configure:

1. **`PROMETHEUS_URL`**: Set the environment variable to your Prometheus/Thanos endpoint
2. **`--auth-mode`**: Choose based on your Prometheus authentication requirements:
   - `serviceaccount` if your Prometheus requires RBAC/token auth
   - `header` if your Prometheus doesn't require authentication
3. **ServiceAccount RBAC**: If using `serviceaccount` mode, ensure the ServiceAccount has permissions to query your metrics endpoint

### Configuring the Prometheus URL

The metrics backend URL is determined in the following order:

1. `PROMETHEUS_URL` environment variable (if set, always used regardless of auth mode)
2. Route discovery via the OpenShift Route API (only in `kubeconfig` mode, respects `--metrics-backend`)
3. Fatal error — `serviceaccount` and `header` modes require `PROMETHEUS_URL` to be set explicitly

> [!NOTE]
>
> Auto-discovery only works in `kubeconfig` mode. For `serviceaccount` and `header` modes, the server
> will fail at startup if `PROMETHEUS_URL` is not set. The same applies to `ALERTMANAGER_URL`.

### Guardrails and Thanos Compatibility

obs-mcp includes query guardrails that prevent expensive or unsafe PromQL queries. Two guardrails rely on the `/api/v1/status/tsdb` endpoint:

| Guardrail | What it checks |
|-----------|----------------|
| `max-metric-cardinality` | Rejects queries against metrics with more series than the configured limit |
| `disallow-blanket-regex` (with `max-label-cardinality > 0`)| Rejects blanket regex matchers (`=~".+"`) on high-cardinality labels |

**Thanos compatibility:**

- **Thanos v0.40.0+** (Oct 2025): The Query component exposes `/api/v1/status/tsdb` ([#8484](https://github.com/thanos-io/thanos/pull/8484)), so all guardrails work.
- **Thanos < v0.40.0**: The TSDB status endpoint is not available on the Query component. Use `--guardrails=none` or use the `!tsdb` shortcut to disable only the TSDB-dependent guardrails while keeping the others enabled:

  ```shell
  --guardrails='!tsdb'
  ```

- **Prometheus**: All guardrails work with any supported Prometheus version.

## Traces (Tempo) toolset

Optional MCP tools query [Grafana Tempo](https://grafana.com/docs/tempo/latest/) instances on the same Kubernetes or OpenShift cluster (see [TOOLS.md](../TOOLS.md) for tool names).

- **Enable:** pass `--toolsets metrics,traces`. The default is `metrics` only, so Tempo tools are hidden until `traces` is included. Example `Deployment` manifests under `manifests/core/deploy/` enable `metrics`, `traces`, and `otelcol`.
- **Discovery:** the server lists `TempoStack` / `TempoMonolithic` resources (Grafana Tempo Operator). The workload `ServiceAccount` must be allowed to `get`, `list`, and `watch` those resources in API group `tempo.grafana.com` (see RBAC in the example manifests).
- **Routes:** use `--traces.use-route` when Tempo should be reached via an OpenShift `Route` instead of in-cluster service DNS.
- **Local stack:** for an optional OpenShift tracing demo (Tempo, tenants, OTel), see [`hack/tempo_multitenancy_openshift/README.md`](../hack/tempo_multitenancy_openshift/README.md).

## OpenTelemetry Collector (`otelcol`) toolset

Optional MCP tools assist with [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) configuration: listing components, fetching JSON schemas, validating configs, and listing supported versions (see [TOOLS.md](../TOOLS.md) for tool names).

- **Enable:** pass `--toolsets otelcol` or include it alongside other toolsets (e.g. `--toolsets metrics,traces,otelcol`). The default is `metrics` only. Example `Deployment` manifests under `manifests/core/deploy/` already enable all three toolsets.
- **Dependencies:** none — component schemas are embedded in the binary; no cluster-side Collector instance is required.
