# MCPChecker Evals

Evaluations for obs-mcp using [mcpchecker](https://github.com/mcpchecker/mcpchecker) — tests that AI agents can discover and correctly use obs-mcp tools against a live Prometheus/Alertmanager backend.

## Pre-requisites

- [mcpchecker](https://github.com/mcpchecker/mcpchecker#install) installed (v0.0.15+) — run `make install-mcpchecker` from the repo root
- A Kubernetes/OpenShift cluster with Prometheus and Alertmanager running
- obs-mcp server deployed and accessible (see [Backend Setup](#backend-setup))

## Environment Variables

mcpchecker uses two separate LLM roles:

- **Agent** — the LLM that interacts with obs-mcp: discovers tools, makes tool calls, and reasons about responses. This is the model being evaluated.
- **Judge** — a separate LLM that evaluates the agent's output against the expected criteria defined in each task.

Both are configured as `builtin.llm-agent` with `openai:gpt-5-nano` by default and share the same API key.

### OpenAI (default)

```bash
export OPENAI_API_KEY="sk-..."
```

This single key is used for both the agent and the LLM judge.

### Other providers

For Anthropic, Gemini, or custom endpoints, see [Using a Different Agent](#using-a-different-agent). Update the `agent` and `llmJudge.ref` sections in `eval.yaml` accordingly.

## Quick Start

### 1. Start obs-mcp locally

```bash
make run
```

On OpenShift <= 4.21 (Thanos Querier backend), disable guardrails since Thanos does not support the TSDB stats endpoint required by cardinality guardrails. Note that the `high-cardinality-rejection` task will not pass without guardrails:

```bash
make run-no-guardrails
```

This uses the default `kubeconfig` auth mode with route auto-discovery. See [Backend Setup](#backend-setup) for other options (Kind cluster, OpenShift). Update `mcp-config.yaml` if obs-mcp is not at `http://localhost:9100/mcp`.

### 2. Set environment variables

```bash
export OPENAI_API_KEY="sk-..."   # used by both agent and LLM judge
```

### 3. Verify connectivity

Run the smoke test first to confirm the metrics backend is reachable. This avoids wasting tokens on evals that will all fail due to connectivity issues:

```bash
make run-mcpchecker-eval TASK=backend-reachability
```

### 4. Run the evals

From the repo root using Makefile targets:

```bash
make run-mcpchecker-eval                           # all tasks in parallel (1 run each)
make run-mcpchecker-eval CATEGORY=metrics          # run by category (metrics, labels, queries, alerts)
make run-mcpchecker-eval TASK=cpu-usage            # single task, verbose
make run-mcpchecker-eval RUNS=3                    # all tasks, 3 runs each for consistency testing
make run-mcpchecker-eval CATEGORY=alerts RUNS=3    # category with multiple runs
```

Or directly:

```bash
cd evals/mcpchecker
mcpchecker check eval.yaml
```

Run tasks in parallel (recommended — all tasks are marked `parallel: true`):

```bash
mcpchecker check eval.yaml --parallel 4
```

Override the MCP config file (e.g., to point at a different obs-mcp instance):

```bash
mcpchecker check eval.yaml --mcp-config-file /path/to/other-mcp-config.yaml
```

The Makefile defaults to `RUNS=1`. Override with `RUNS=N` for consistency testing:

```bash
make run-mcpchecker-eval TASK=cpu-usage RUNS=3
```

### Running a Single Task

Use `TASK` to filter by name or `CATEGORY` to filter by category:

```bash
make run-mcpchecker-eval TASK=cpu-usage            # single task, verbose
make run-mcpchecker-eval TASK="alert|silence"      # regex match
make run-mcpchecker-eval CATEGORY=alerts           # all alert tasks
```

Or directly with `mcpchecker`:

```bash
mcpchecker check eval.yaml --run "cpu-usage" --verbose
```

Use `-l / --label-selector` to filter by task labels:

```bash
# Run only metric discovery tasks
mcpchecker check eval.yaml --label-selector "category=metrics"

# Run only alertmanager tasks
mcpchecker check eval.yaml --label-selector "category=alerts"
```

### 4. View results

```bash
mcpchecker summary mcpchecker-obs-mcp-tools-out.json
```

Compare results between runs:

```bash
mcpchecker diff baseline-out.json current-out.json
```

## Backend Setup

### Kind cluster — obs-mcp deployed in-cluster

Reuses the e2e setup:

```bash
make test-e2e-setup                        # create Kind cluster with kube-prometheus
make deploy-more-kube-prom-targets         # deploy kube-state-metrics, node-exporter, kubelet scrape configs
make test-e2e-deploy                       # build and deploy obs-mcp
kubectl port-forward -n obs-mcp svc/obs-mcp 9100:9100 &

export OPENAI_API_KEY="sk-..."

make run-mcpchecker-eval                   # run all tasks in parallel
make run-mcpchecker-eval TASK=cpu-usage    # single task, verbose
```

### Kind cluster — obs-mcp running locally

```bash
make test-e2e-setup                        # create Kind cluster with kube-prometheus (skip if already running)
make deploy-more-kube-prom-targets         # deploy kube-state-metrics, node-exporter, kubelet scrape configs
kubectl port-forward -n monitoring svc/prometheus-k8s 9090:9090 &
kubectl port-forward -n monitoring svc/alertmanager-main 9093:9093 &
PROMETHEUS_URL=http://localhost:9090 ALERTMANAGER_URL=http://localhost:9093 AUTH_MODE=header make run

# In another terminal:
export OPENAI_API_KEY="sk-..."

make run-mcpchecker-eval                   # all tasks
make run-mcpchecker-eval TASK=cpu-usage    # single task, verbose
```

### OpenShift

Start obs-mcp locally in one terminal, run evals in another:

```bash
make run                          # via route auto-discovery (OpenShift >= 4.22)
# or
make run-openshift-pf-prometheus  # via port-forward
```

On OpenShift <= 4.21, the default backend is Thanos Querier which does not support `/api/v1/status/tsdb` (required by the `max-metric-cardinality` and `max-label-cardinality` guardrails). Either disable all guardrails or keep only the static checks:

```bash
make run-no-guardrails
# or selectively keep static guardrails:
./obs-mcp --listen :9100 --auth-mode kubeconfig --guardrails require-label-matcher,disallow-blanket-regex
```

```bash
export OPENAI_API_KEY="sk-..."

make run-mcpchecker-eval                   # all tasks
make run-mcpchecker-eval TASK=cpu-usage    # single task, verbose
```

Update `mcp-config.yaml` if obs-mcp is not at `http://localhost:9100/mcp`.

> **Note:** Once the obs-mcp container image is published or you build one yourself, evals can also run against an in-cluster deployment on OpenShift via `kubectl port-forward -n obs-mcp svc/obs-mcp 9100:9100`.

## Using a Different Agent

By default, the evals use `builtin.llm-agent` with `openai:gpt-5-nano`. To use a different provider or model, edit the `agent` and `llmJudge.ref` sections in `eval.yaml`. See the [mcpchecker agent docs](https://github.com/mcpchecker/mcpchecker/blob/main/docs/how-to/configure-agents.md) for all supported providers and configuration options.

## Task Structure

Tasks are organized by category under `tasks/`:

| Directory          | Description                                           |
|--------------------|-------------------------------------------------------|
| `tasks/metrics/`   | Metric discovery and listing                          |
| `tasks/labels/`    | Label names, values, and series                       |
| `tasks/queries/`   | PromQL queries and multi-step diagnostics             |
| `tasks/alerts/`    | Alertmanager alerts, investigation, silences          |
| `tasks/otelcol/`   | OpenTelemetry Collector components, schemas, configs  |

Each task YAML defines the prompt, expected tools, call bounds, and LLM judge criteria. All tasks include `labels` for filtering with `--label-selector` (e.g. `category=metrics`, `category=alerts`).

## Adding New Tasks

Create a new YAML file under the appropriate `tasks/` subdirectory:

```yaml
kind: Task
apiVersion: mcpchecker/v1alpha2
metadata:
  name: "my-new-task"
  difficulty: medium
  parallel: true
  runs: 1
  labels:
    category: queries
    toolType: instant-query
spec:
  verify:
    - llmJudge:
        contains: "expected_metric_name"
        reason: "Verify the agent used the correct metric"
  prompt:
    inline: |
      Your natural language question here.
```

Then add a corresponding `taskSet` entry in `eval.yaml` pointing to the new file.

> **TODO:** All tasks currently use `runs: 1` to reduce token cost while iterating. Once evals are stable, bump to `runs: 3` for consistency testing.
