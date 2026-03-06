# Deployment Guide

This guide covers authentication modes and deploying obs-mcp on Kubernetes/OpenShift clusters.

## Authentication Modes

The `--auth-mode` flag controls how obs-mcp authenticates to Prometheus/Thanos:

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

Example manifests are provided in the `manifests/` directory:

- `manifests/openshift/` - Example for OpenShift with Thanos Querier
- `manifests/kubernetes/` - Example for Kubernetes

These are **reference examples** that you'll need to customize for your environment.

### Key Configuration

When deploying in-cluster, you must configure:

1. **`PROMETHEUS_URL`**: Set the environment variable to your Prometheus/Thanos endpoint
2. **`--auth-mode`**: Choose based on your Prometheus authentication requirements:
   - `serviceaccount` if your Prometheus requires RBAC/token auth
   - `header` if your Prometheus doesn't require authentication
3. **ServiceAccount RBAC**: If using `serviceaccount` mode, ensure the ServiceAccount has permissions to query your metrics endpoint

### Configuring the Prometheus URL

The metrics backend URL is determined in the following order:

1. `PROMETHEUS_URL` environment variable (if set, always used)
2. `--metrics-backend` flag route discovery (only in `kubeconfig` mode)
3. Default: `http://localhost:9090`

> [!NOTE]
>
> Auto-discovery only works in `kubeconfig` mode. For in-cluster deployments, you must set `PROMETHEUS_URL` explicitly.

### Guardrails and Thanos Compatibility

obs-mcp includes query guardrails that prevent expensive or unsafe PromQL queries. Two guardrails rely on the `/api/v1/status/tsdb` endpoint:

| Guardrail | What it checks |
|-----------|----------------|
| `max-metric-cardinality` | Rejects queries against metrics with more series than the configured limit |
| `max-label-cardinality` (with `disallow-blanket-regex`) | Rejects blanket regex matchers (`=~".+"`) on high-cardinality labels |

**Thanos compatibility:**

- **Thanos v0.40.0+** (Oct 2025): The Query component exposes `/api/v1/status/tsdb` ([#8484](https://github.com/thanos-io/thanos/pull/8484)), so all guardrails work.
- **Thanos < v0.40.0**: The TSDB status endpoint is not available on the Query component. Use `--guardrails=none` or disable only the cardinality guardrails while keeping the others enabled:

  ```shell
  --guardrails=disallow-explicit-name-label,require-label-matcher,disallow-blanket-regex --guardrails.max-label-cardinality=0
  ```

- **Prometheus**: All guardrails work with any supported Prometheus version.
