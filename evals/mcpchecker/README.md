# MCPChecker Evals

Evaluations for obs-mcp using [mcpchecker](https://github.com/mcpchecker/mcpchecker) — tests that AI agents can discover and correctly use obs-mcp tools against a live Prometheus/Alertmanager backend.

## Pre-requisites

- [mcpchecker](https://github.com/mcpchecker/mcpchecker#installation) installed
- A Kubernetes cluster with Prometheus and Alertmanager running
- obs-mcp server deployed and accessible (see [Testing Guide — MCPChecker Evals](../../TESTING.md#mcpchecker-evals))

## Environment Variables

### LLM Judge (required)

All tasks use LLM judge verification to semantically check agent responses. These must be set:

```bash
export JUDGE_BASE_URL="https://api.openai.com/v1"   # OpenAI-compatible API endpoint
export JUDGE_API_KEY="sk-..."                         # API key for the judge model
export JUDGE_MODEL_NAME="gpt-4o-mini"                 # Model to use as judge
```

### Agent-specific

**OpenAI** (default agent — `builtin.llm-agent` with `openai:gpt-4o-mini`):

```bash
export OPENAI_API_KEY="sk-..."
```

**Other providers** — edit `eval.yaml` to change the model (see [Using a Different Agent](#using-a-different-agent)):

```bash
export ANTHROPIC_API_KEY="sk-..."     # for anthropic:* models
export GEMINI_API_KEY="..."           # for gemini:* models
```

## Quick Start

### 1. Ensure obs-mcp is running

Port-forward to the obs-mcp service:

```bash
kubectl port-forward -n obs-mcp svc/obs-mcp 9100:9100
```

Or if running elsewhere, update `mcp-config.yaml` with the correct URL.

### 2. Set environment variables

```bash
export OPENAI_API_KEY="sk-..."                       # for the default openai:gpt-4o-mini agent
export JUDGE_BASE_URL="https://api.openai.com/v1"
export JUDGE_API_KEY="sk-..."
export JUDGE_MODEL_NAME="gpt-4o-mini"   # Model to use as judge
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

### 4. View results

```bash
mcpchecker summary mcpchecker-obs-mcp-tools-out.json
```

## Using a Different Agent

By default, the evals use `builtin.llm-agent` with `openai:gpt-4o-mini`. To use a different provider or model, edit the `agent` section in `eval.yaml`. See the [mcpchecker agent documentation](https://github.com/mcpchecker/mcpchecker#agents) for available agent types and configuration options.

## Coverage

16 eval tasks across 4 categories:

| Category          | Tasks                                                                                                                  | Tools Tested                                        |
|-------------------|------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------|
| Metrics discovery | list kube metrics, list node metrics                                                                                   | `list_metrics`                                      |
| Label exploration | label names, label values, series cardinality                                                                          | `get_label_names`, `get_label_values`, `get_series` |
| PromQL queries    | CPU usage, pending pods, crashlooping pods, pods created, network traffic, Prometheus internals (head series, requests, WAL size) | `execute_instant_query`, `execute_range_query`      |
| Alertmanager      | firing alerts, active alerts, silences                                                                                 | `get_alerts`, `get_silences`                        |

Each task verifies:

- The agent selects the correct tool(s)
- Tool call count stays within bounds
- Response contains expected content (via LLM judge)

> **Note:** This is a smoke-test level evaluation covering basic tool discovery and usage. We need to add:
>
> - **Multi-step reasoning** — tasks requiring 3+ chained tools (e.g., discover metric → query → analyze trend)
> - **Error handling** — agent recovery from invalid queries or missing metrics
> - **Guardrail behavior** — agent response when dangerous queries are blocked
> - **Parameter coverage** — testing less-used params like `silenced`, `inhibited`, `receiver`, `filter`, time ranges
> - **Ambiguous prompts** — vague diagnostic questions (e.g., "Why is my app slow?") requiring the agent to choose the right tools
> - **Hard difficulty tasks** — complex multi-tool diagnostic scenarios

## Task Structure

| Directory          | Description                      | Tools Tested                                        |
|--------------------|----------------------------------|-----------------------------------------------------|
| `tasks/metrics/`   | Metric discovery and listing     | `list_metrics`                                      |
| `tasks/labels/`    | Label names, values, and series  | `get_label_names`, `get_label_values`, `get_series` |
| `tasks/queries/`   | Instant and range PromQL queries | `execute_instant_query`, `execute_range_query`      |
| `tasks/alerts/`    | Alertmanager alerts and silences | `get_alerts`, `get_silences`                        |

## Adding New Tasks

Create a new YAML file under the appropriate `tasks/` subdirectory:

```yaml
kind: Task
apiVersion: mcpchecker/v1alpha2
metadata:
  name: "my-new-task"
  difficulty: medium
  parallel: true
spec:
  verify:
    - llmJudge:
        contains: "expected content"
  prompt:
    inline: |
      Your natural language question here.
```

Then add a corresponding `taskSet` entry in `eval.yaml` pointing to the new file.
