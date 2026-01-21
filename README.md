# obs mcp server

[![lint](https://github.com/rhobs/obs-mcp/actions/workflows/lint.yaml/badge.svg)](https://github.com/rhobs/obs-mcp/actions/workflows/lint.yaml)
[![unit](https://github.com/rhobs/obs-mcp/actions/workflows/unit.yaml/badge.svg)](https://github.com/rhobs/obs-mcp/actions/workflows/unit.yaml)
[![e2e](https://github.com/rhobs/obs-mcp/actions/workflows/e2e.yaml/badge.svg)](https://github.com/rhobs/obs-mcp/actions/workflows/e2e.yaml)

obs-mcp is a [mcp](https://modelcontextprotocol.io/introduction) server to allow LLMs to interact with [Prometheus](https://prometheus.io/) or [Thanos Querier](https://thanos.io/) instances via the API.

> [!NOTE]
> This project is moved from [jhadvig/genie-plugin](https://github.com/jhadvig/genie-plugin/tree/main/obs-mcp) preserving the history of commits.

## Quickstart

### 1. Using Kubeconfig (OpenShift)

The easiest way to get the obs-mcp connected to the cluster is via a kubeconfig:

 1. Login into your OpenShift cluster
 2. Run the server with

 ```shell
 go run ./cmd/obs-mcp/ --listen 127.0.0.1:9100 --auth-mode kubeconfig --insecure
 ```

This will auto-discover the metrics backend in OpenShift. By default, it tries `thanos-querier` route first, then falls back to `prometheus-k8s` route. Use `--metrics-backend` to control which route is preferred.

Use the `--metrics-backend` flag to specify which metrics backend to discover:

| Flag Value           | Behavior                                                              |
|----------------------|-----------------------------------------------------------------------|
| `thanos` (default)   | Tries `thanos-querier` route first, falls back to `prometheus-k8s`    |
| `prometheus`         | Uses `prometheus-k8s` route only (no fallback)                        |

> [!WARNING]
> This procedure would not work if you're not using token-based auth (`oc > whoami -t` to validate).
> In that case, consider using serviceaccount + token auth.

**Example using Prometheus as the preferred backend:**

```shell
go run ./cmd/obs-mcp/ --listen 127.0.0.1:9100 --auth-mode kubeconfig --metrics-backend prometheus --insecure
```

**Example using Thanos as the preferred backend:**

> [!NOTE]
>
> Thanos in OpenShift doesn't expose the TSDB endpoint, so guardrails that rely on TSDB stats won't work. Use `--guardrails=none` when using Thanos.

```shell
go run ./cmd/obs-mcp/ --listen 127.0.0.1:9100 --auth-mode kubeconfig --metrics-backend thanos --insecure --guardrails=none
```

> [!IMPORTANT]
> **How the Metrics Backend URL is Determined:**
>
> 1. `PROMETHEUS_URL` environment variable (if set, always used)
> 2. `--metrics-backend` flag route discovery (only in `kubeconfig` mode)
> 3. Default: `http://localhost:9090`
>
>
> **Example using explicit PROMETHEUS_URL:**
>
  ```shell
  export PROMETHEUS_URL=https://thanos-querier.openshift-monitoring.svc.cluster.local:9091/
  go run ./cmd/obs-mcp/ --listen 127.0.0.1:9100 --auth-mode kubeconfig --insecure
  ```

### 2. Port-forwarding alternative

This scenario opens a local port via port-forward that the obs-mcp will connect to:

 1. Log into your OpenShift cluster

 2. Port forward the OpenShift in-cluster Prometheus instance to a local port

  ```shell
  PROM_POD=$(kubectl get pods -n openshift-monitoring -l app.kubernetes.io/instance=k8s -l app.kubernetes.io/component=prometheus -o jsonpath="{.items[0].metadata.name}")

  kubectl port-forward -n openshift-monitoring $PROM_POD 9090:9090
  ```

  Run the server with:

  ```shell
  export PROMETHEUS_URL=http://localhost:9090
  go run ./cmd/obs-mcp/ --listen 127.0.0.1:9100 --auth-mode header
  ```

### 3. Local Development with Kind (using E2E test infrastructure)

Use the E2E test infrastructure for a fully working local environment with Prometheus:

#### Setup Kind cluster with Prometheus

```bash
make test-e2e-setup
```

This creates a Kind cluster with:

- Prometheus Operator
- Prometheus (accessible at `prometheus-k8s.monitoring.svc.cluster.local:9090`)
- Alertmanager

#### Build and deploy obs-mcp

```bash
make test-e2e-deploy
```

#### Port forward obs-mcp

```bash
kubectl port-forward -n obs-mcp svc/obs-mcp 9100:9100
```

To connect an MCP client, use `http://localhost:9100/mcp`.

When done:

```bash
make test-e2e-teardown
```

See [TESTING.md](TESTING.md) for more details.

### 4. Using prometheus helm chart in local Kubernetes cluster

```shell
# sets up Prometheus (and exporters) on your local single-node k8s cluster
helm install prometheus-community/prometheus --name-template <prefix>

export POD_NAME=$(kubectl get pods --namespace default -l "app.kubernetes.io/name=alertmanager,app.kubernetes.io/instance=local" -o jsonpath="{.items[0].metadata.name}") && kubectl --namespace default port-forward $POD_NAME 9090

go run ./cmd/obs-mcp/ --auth-mode header --insecure --listen :9100 
```

### Testing with curl

You can test the MCP server using curl. The server uses `JSON-RPC 2.0` over `HTTP`.

> [!TIP]
> For formatted JSON output, pipe the response to `jq`:
>
> curl ... | jq
>

**List available tools:**

```shell
curl -X POST http://localhost:9100/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'|jq
```

**Call the list_metrics tool:**

```shell
curl -X POST http://localhost:9100/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_metrics","arguments":{}}}' | jq
```
  
**Execute a range query (e.g., get up metrics for the last hour):**

```shell
curl -X POST http://localhost:9100/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"execute_range_query","arguments":{"query":"up{job=\"prometheus\"}","step":"1m","end":"NOW","duration":"1h"}}}' | jq
```

## License

[Apache 2.0](LICENSE)
