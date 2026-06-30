# MCPChecker Evals

Evaluations for obs-mcp using [mcpchecker](https://github.com/mcpchecker/mcpchecker) — tests that AI agents can discover and correctly use obs-mcp tools against a live Prometheus/Alertmanager backend.

## Pre-requisites

- [mcpchecker](https://github.com/mcpchecker/mcpchecker#install) installed (v0.0.16+) — run `make install-mcpchecker` from the repo root
- **Metrics / alerts / traces / otelcol:** Kubernetes or OpenShift cluster with Prometheus and Alertmanager (see [Setup the cluster](#setup-the-cluster)). Toolsets are namespaced as `observability/metrics`, `observability/traces`, `observability/logs`, `observability/otelcol`.
- **obs-mcp** running at `http://localhost:9100/mcp` before any mcpchecker run

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

### Setup the cluster

It's recommended to leverage the e2e test setup and making sure
the tests are passing before doing the evaluation.

The tests setup supports multiple profiles passed via `E2E_PROFILE` variable:
- `kind` - provision a local kind cluster and deploy the workloads to it.
- `k8s` - deploy the workloads to a remote Kubernetes cluster
- `openshift` - deploy the workloads to a remote OpenShift cluster

```bash
export E2E_PROFILE=kind
# prepare the cluster prerequisites
make test-e2e-setup
# deploys obs-mcp to the cluster
make test-e2e-deploy
# (optional) run the e2e tests to check everything works
make test-e2e
# port forward the deployment locally
make test-e2e-pf
```

### Run evals

``` bash
export OPENAI_API_KEY="sk-..."

make run-mcpchecker-eval                       # run all tasks in parallel
make run-mcpchecker-eval TASK=cpu-usage        # single task, verbose
make run-mcpchecker-eval TASK="alert|silence"  # regex match
make run-mcpchecker-eval CATEGORY=alerts       # all alert tasks
```

### Triaging evals results

You can use the `/mcpchecker-results-triage` skill in an AI assistant
to triage the results. It's useful way to identify potential causes for the failures 
propose fixes.

### Additional mcpchecker tools

```bash
cd evals/mcpchecker
mcpchecker summary mcpchecker-obs-mcp-tools-out.json
```

Compare results between runs:

```bash
cd evals/mcpchecker
mcpchecker diff baseline-out.json current-out.json
```

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
| `tasks/traces/`    | Tempo trace search and latency investigation          |
| `tasks/logs/`      | LokiStack discovery, labels, and LogQL queries        |
| `tasks/otelcol/`   | OpenTelemetry Collector components, schemas, configs  |

Each task YAML defines the prompt, expected tools, call bounds, and LLM judge criteria. All tasks include `labels` for filtering with `--label-selector` (e.g. `category=metrics`, `category=alerts`, `category=logs`).

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

## Keeping Evals in Sync with openshift-mcp-server

The observability eval tasks in this repo (`evals/mcpchecker/tasks/`) are the **source of truth** for task definitions. The same tasks are mirrored in [openshift-mcp-server](https://github.com/openshift/openshift-mcp-server/tree/main/evals/tasks/observability) under `evals/tasks/observability/`.

When updating eval tasks, changes must be synced between both repos to avoid config drift:

1. **obs-mcp → openshift-mcp-server**: After updating tasks here, copy them to `evals/tasks/observability/` in openshift-mcp-server and open a PR.
2. **openshift-mcp-server → obs-mcp**: If tasks are updated there first (e.g. after running evals on an OpenShift cluster), copy them back here.

Everything under `evals/mcpchecker/tasks/` in this repo maps to `evals/tasks/observability/` in openshift-mcp-server.

To check for divergence between the two repos:

```bash
diff -r evals/mcpchecker/tasks/ /path/to/openshift-mcp-server/evals/tasks/observability/
```

> [!NOTE]
> The directory layout in openshift-mcp-server may change over time, but the goal is to always keep the observability eval tasks in sync with this repo.

> **TODO:** All tasks currently use `runs: 1` to reduce token cost while iterating. Once evals are stable, bump to `runs: 3` for consistency testing.
