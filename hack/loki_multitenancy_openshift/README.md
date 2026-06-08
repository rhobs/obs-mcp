# LokiStack validation stack (OpenShift)

Manifests and scripts to stand up a local Loki Operator + LokiStack test environment for `obs-mcp` Loki discovery.

## What this hack provides

- Loki Operator installation via OLM Subscription
- a project/namespace for test data (`obs-mcp-loki`)
- in-cluster MinIO S3-compatible storage (no cloud credentials)
- a `LokiStack` custom resource backed by MinIO
- RBAC for `obs-mcp` service account to discover `LokiStack` and `Route` objects
- optional ClusterRole for querying NetObserv flow logs through the Loki gateway (OpenShift auth)
- a log generator workload
- a verification script that checks:
  - `LokiStack` visibility
  - gateway route/service resolution
  - basic `loki_list_instances` and `loki_query_range` calls through MCP

## Prerequisites

- OpenShift cluster
- `oc`, `jq`, and `curl` available locally
- Running `obs-mcp` with `logs` toolset enabled

Example server startup (start this **before** running `03_verify.sh`):

```bash
go run ./cmd/obs-mcp --listen 127.0.0.1:9100 --toolsets logs --auth-mode kubeconfig --insecure --loki.use-route
```

If the LokiStack has no OpenShift Route, use port-forward instead of `--loki.use-route`:

```bash
oc port-forward -n obs-mcp-loki svc/obs-mcp-loki-gateway-http 8080:8080
go run ./cmd/obs-mcp --listen 127.0.0.1:9100 --toolsets logs --auth-mode kubeconfig --insecure --loki-url http://127.0.0.1:8080
```

With `--auth-mode header`, set `PROMETHEUS_URL` and `ALERTMANAGER_URL` if you also enable the `metrics` toolset. For Loki-only testing, `logs` + `kubeconfig` is enough (LokiStack discovery uses your current `oc` credentials).

## Install test resources

Apply base resources first (project, OperatorGroup, MinIO, log generator). The verify script creates the Loki Operator Subscription using the newest `stable-*` channel from the package manifest (for example `stable-6.5`), then applies the `LokiStack` CR after CRDs are installed:

```bash
oc apply -f hack/loki_multitenancy_openshift/install/
```

If you previously ran `oc apply -f hack/loki_multitenancy_openshift/` and only `03_lokistack.yaml` failed, the install step above is already done — run verify below to create the stack.

## Verify discovery and queries

```bash
hack/loki_multitenancy_openshift/03_verify.sh
```

Environment overrides:

- `MCP_URL` (default: `http://127.0.0.1:9100/mcp`)
- `LOKI_NS` (default: `obs-mcp-loki`)
- `STACK_NS` (default: `obs-mcp-loki`)
- `STACK_NAME` (default: `obs-mcp-loki`)
- `LOKI_TENANT` (default: `network`, required for `openshift-network` mode)
- `NETOBSERV_NS` (default: `netobserv`, used in NetObserv LogQL filters)
- `LOKI_QUERY` (default: `{SrcK8S_Namespace="<NETOBSERV_NS>"} or {DstK8S_Namespace="<NETOBSERV_NS>"}`)
- `VERIFY_DURATION` (default: `1h`)
- `LOKI_STORAGE_CLASS` (default: auto-detected from the cluster default StorageClass)
- `LOKI_OPERATOR_CHANNEL` (default: newest `stable-*` from `packagemanifest/loki-operator`, e.g. `stable-6.5`)
- `LOKI_OPERATOR_NS` (default: `openshift-loki-operator`)
- `LOKI_OPERATOR_CATALOG` (default: `redhat-operators`)
- `LOKI_OPERATOR_SOURCE_NAMESPACE` (default: `openshift-marketplace`)

### NetObserv flow logs and RBAC

NetObserv stores **flow logs** in the `network` tenant. They use indexed labels such as `SrcK8S_Namespace` and `DstK8S_Namespace`, **not** `kubernetes_namespace_name` (that label is for container/application logs).

The Loki gateway enforces OpenShift RBAC: your `oc` user (passed via `--auth-mode kubeconfig`) must be allowed to view pods in namespaces you filter. If you see empty `streams` with no error, fix the query labels and/or bind query RBAC:

```bash
oc apply -f hack/loki_multitenancy_openshift/install/02_gateway_query_rbac.yaml
oc create clusterrolebinding "obs-mcp-loki-gateway-read-$(oc whoami | tr '@:' '-')" \
  --clusterrole=obs-mcp-loki-gateway-read \
  --user="$(oc whoami)"
```

Ensure your `FlowCollector` exports to this LokiStack (`obs-mcp-loki` in `obs-mcp-loki` namespace) and set `NETOBSERV_NS` to the namespace you care about (often `netobserv`).

## Cleanup

```bash
oc delete -f hack/loki_multitenancy_openshift/03_lokistack.yaml --ignore-not-found
oc delete -f hack/loki_multitenancy_openshift/install/ --ignore-not-found
oc delete clusterrole/obs-mcp-loki-gateway-read --ignore-not-found
```
