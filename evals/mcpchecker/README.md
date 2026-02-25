# MCPChecker Evals

Evaluations for obs-mcp using [mcpchecker](https://github.com/mcpchecker/mcpchecker) — tests that AI agents can discover and correctly use obs-mcp tools against a live Prometheus/Alertmanager backend.

## Pre-requisites

- [mcpchecker](https://github.com/mcpchecker/mcpchecker#installation) installed
- A Kubernetes cluster with Prometheus and Alertmanager running
- obs-mcp server deployed and accessible (see [Deployment Guide](../../docs/DEPLOYMENT.md))

## Environment Variables

### LLM Judge (required)

All tasks use LLM judge verification to semantically check agent responses. These must be set:

```bash
export JUDGE_BASE_URL="https://api.openai.com/v1"   # OpenAI-compatible API endpoint
export JUDGE_API_KEY="sk-..."                         # API key for the judge model
export JUDGE_MODEL_NAME="gpt-4o"                      # Model to use as judge
```

### Agent-specific

**Claude Code** (default agent — `builtin.claude-code`):

- The [`claude`](https://docs.anthropic.com/en/docs/claude-code) CLI must be installed and in your `PATH`
- Authentication is managed by the Claude Code CLI itself

**OpenAI-compatible agent** (`builtin.openai-agent`):

```bash
export MODEL_BASE_URL="https://api.openai.com/v1"   # OpenAI-compatible API endpoint
export MODEL_KEY="sk-..."                             # API key for the agent model
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
export JUDGE_BASE_URL="https://api.openai.com/v1"
export JUDGE_API_KEY="sk-..."
export JUDGE_MODEL_NAME="gpt-4o"
```

### 3. Run the evals

```bash
cd evals/mcpchecker
mcpchecker eval eval.yaml
```

### 4. View results

```bash
mcpchecker summary mcpchecker-obs-mcp-tools-out.json
```

## Using a Different Agent

By default, the evals use `builtin.claude-code`. To use an OpenAI-compatible agent, edit `eval.yaml`:

```yaml
config:
  agent:
    type: "builtin.openai-agent"
    model: "gpt-4o"
```

## Task Structure

| Directory        | Description                              | Tools Tested                             |
|------------------|------------------------------------------|------------------------------------------|
| `tasks/metrics/` | Metric discovery and listing             | `list_metrics`                           |
| `tasks/labels/`  | Label names, values, and series          | `get_label_names`, `get_label_values`, `get_series` |
| `tasks/queries/` | Instant and range PromQL queries         | `execute_instant_query`, `execute_range_query` |
| `tasks/alerts/`  | Alertmanager alerts and silences         | `get_alerts`, `get_silences`             |

## Adding New Tasks

Create a new YAML file under the appropriate `tasks/` subdirectory:

```yaml
kind: Task
apiVersion: mcpchecker/v1alpha2
metadata:
  name: "my-new-task"
  difficulty: medium
spec:
  verify:
    - llmJudge:
        contains: "expected content"
  prompt:
    inline: |
      Your natural language question here.
```

Then add a corresponding `taskSet` entry in `eval.yaml` pointing to the new file.
