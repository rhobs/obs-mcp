# MCPChecker Evals

Evaluations for obs-mcp using [mcpchecker](https://github.com/mcpchecker/mcpchecker) — tests that AI agents can discover and correctly use obs-mcp tools against a live Prometheus/Alertmanager backend.

## Pre-requisites

- [mcpchecker](https://github.com/mcpchecker/mcpchecker#install) installed (v0.0.15+)
- A Kubernetes cluster with Prometheus and Alertmanager running
- obs-mcp server deployed and accessible (see [Testing Guide — MCPChecker Evals](../../TESTING.md#mcpchecker-evals))

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

### 1. Ensure obs-mcp is running

Port-forward to the obs-mcp service:

```bash
kubectl port-forward -n obs-mcp svc/obs-mcp 9100:9100
```

Or if running elsewhere, update `mcp-config.yaml` with the correct URL.

### 2. Set environment variables

```bash
export OPENAI_API_KEY="sk-..."   # used by both agent and LLM judge
```

### 3. Run the evals

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

Tasks with `runs` configured will automatically execute multiple times for consistency testing. To override the run count for all tasks:

```bash
mcpchecker check eval.yaml --parallel 4 --runs 5
```

### Running a Single Task

Use `-r / --run` to filter tasks by name (regex, like `go test -run`):

```bash
# Run only the cpu-usage task
mcpchecker check eval.yaml --run "cpu-usage"

# Single run (overrides the task's configured runs: 3)
mcpchecker check eval.yaml --run "cpu-usage" --runs 1

# Run all alert-related tasks
mcpchecker check eval.yaml --run "alert|silence"

# Verbose output to see tool calls
mcpchecker check eval.yaml --run "cpu-usage" --runs 1 --verbose
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

## Using a Different Agent

By default, the evals use `builtin.llm-agent` with `openai:gpt-5-nano`. To use a different provider or model, edit the `agent` and `llmJudge.ref` sections in `eval.yaml`. The multi-provider `llm-agent` supports `provider:model-id` format:

```yaml
# eval.yaml
config:
  agent:
    type: "builtin.llm-agent"
    model: "anthropic:claude-3-haiku-20240307"
  llmJudge:
    ref:
      type: builtin.llm-agent
      model: "anthropic:claude-3-haiku-20240307"
```

Supported providers: `openai`, `anthropic`, `gemini`, `google` (Vertex AI). Set the corresponding environment variable:

```bash
# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Gemini
export GEMINI_API_KEY="..."

# Gemini via Vertex AI
export GEMINI_USE_VERTEX=1
export GOOGLE_CLOUD_PROJECT="your-project"
export GOOGLE_CLOUD_LOCATION="us-central1"
```

See the [mcpchecker agent docs](https://github.com/mcpchecker/mcpchecker/blob/main/docs/how-to/configure-agents.md) for all agent types and configuration options.

## Coverage

17 eval tasks across 4 categories and 3 difficulty levels:

| Category          | Tasks                                                                                                                  | Difficulty | Tools Tested                                        |
|-------------------|------------------------------------------------------------------------------------------------------------------------|------------|-----------------------------------------------------|
| Metrics discovery | list kube metrics, list node metrics                                                                                   | easy       | `list_metrics`                                      |
| Label exploration | label names, label values, series cardinality                                                                          | easy-medium | `get_label_names`, `get_label_values`, `get_series` |
| PromQL queries    | CPU usage, pending pods, crashlooping pods, pods created, network traffic, Prometheus internals (head series, requests, WAL size) | easy-medium | `execute_instant_query`, `execute_range_query`      |
| Multi-step queries | namespace resource usage, cluster health diagnosis                                                                    | hard       | Multiple tools chained together                     |
| Alertmanager      | firing alerts, alert investigation, silences                                                                           | easy-medium | `get_alerts`, `get_silences`                        |

Each task verifies:

- The agent selects the correct tool(s)
- Tool call count stays within bounds
- Tool call order follows the mandatory `list_metrics`-first workflow (via `callOrder`)
- Response contains expected content (via LLM judge with specific metric name checks)
- Consistency across multiple runs (via `runs` metadata)

All tasks include `labels` for filtering with `labelSelector`:
- `category`: `metrics`, `labels`, `queries`, `alerts`
- `toolType`: `discovery`, `exploration`, `instant-query`, `range-query`, `alertmanager`, `multi-step`, `diagnostic`

> **Note:** Areas for future improvement:
>
> - **Error handling** — agent recovery from invalid queries or missing metrics
> - **Guardrail behavior** — agent response when dangerous queries are blocked (use [`toolsNotUsed`](https://github.com/mcpchecker/mcpchecker/blob/main/docs/how-to/use-assertions.md#forbidden-tools) to enforce)
> - **Redundancy checks** — use [`noDuplicateCalls`](https://github.com/mcpchecker/mcpchecker/blob/main/docs/how-to/use-assertions.md#no-duplicate-calls) for simple tasks
> - **Parameter coverage** — testing less-used params like `silenced`, `inhibited`, `receiver`, `filter`, time ranges

## Task Structure

| Directory          | Description                              | Tools Tested                                        |
|--------------------|------------------------------------------|-----------------------------------------------------|
| `tasks/metrics/`   | Metric discovery and listing             | `list_metrics`                                      |
| `tasks/labels/`    | Label names, values, and series          | `get_label_names`, `get_label_values`, `get_series` |
| `tasks/queries/`   | PromQL queries and multi-step diagnostics | `execute_instant_query`, `execute_range_query`      |
| `tasks/alerts/`    | Alertmanager alerts, investigation, silences | `get_alerts`, `get_silences`                     |

## Adding New Tasks

Create a new YAML file under the appropriate `tasks/` subdirectory:

```yaml
kind: Task
apiVersion: mcpchecker/v1alpha2
metadata:
  name: "my-new-task"
  difficulty: medium
  parallel: true
  runs: 3
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
