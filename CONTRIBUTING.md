# Contributing to obs-mcp

Thanks for contributing. Please read this before opening a PR — **most observability tool work belongs upstream**, not in this repository.

## Toolsets moved upstream

Observability tool definitions, handlers, and shared backend packages were moved into
[containers/kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server):

| Upstream path | Contents |
|---------------|----------|
| `pkg/toolsets/metrics` | Prometheus / Thanos / Alertmanager tools |
| `pkg/toolsets/logs` | Loki tools |
| `pkg/toolsets/traces` | Tempo tools |
| `pkg/toolsets/otelcol` | OpenTelemetry Collector tools |
| `pkg/obs/` | Auth, discovery, clients, instrumentation |

**Open toolset / backend / tool-schema PRs against kubernetes-mcp-server.** Changes merged only into obs-mcp will not ship for embedders of those packages and will diverge from the shared source of truth.

obs-mcp is a **thin standalone host**: it selects toolsets, adapts them to the Go MCP SDK, and provides CLI, deployment manifests, and evals.

## What belongs in this repo

Contribute here when the change is about the standalone host, for example:

- `cmd/obs-mcp` CLI flags and startup
- `pkg/mcp` SDK adapter / HTTP transport
- `pkg/clusterinspector` stub
- Kubernetes / OpenShift manifests
- mcpchecker eval tasks (`evals/mcpchecker/`)
- Host docs, release process, CI for this binary

## Workflow for tool changes

1. Land the change (and its unit tests) in [kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server).
2. In obs-mcp, bump the pin (`/kube-mcp-sync` or `go get github.com/containers/kubernetes-mcp-server@main && go mod tidy`).
3. Run `make generate-tools-doc` and commit an updated `TOOLS.md` if needed.
4. Run `make test-unit` (and e2e if the host surface changed).

## Before opening a PR here

- [ ] Confirmed the change is host-related, **or** the toolset fix already landed upstream and this PR only bumps / adapts
- [ ] `make test-unit` passes
- [ ] `TOOLS.md` regenerated if tool descriptions changed via the pin bump

## Testing

See [TESTING.md](TESTING.md).

## History

This repository was moved from [jhadvig/genie-plugin](https://github.com/jhadvig/genie-plugin/tree/main/obs-mcp) into `rhobs/obs-mcp`, preserving commit history. Observability toolsets later moved into [kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server) (see [#213](https://github.com/rhobs/obs-mcp/pull/213)); this repo remains the standalone host.
