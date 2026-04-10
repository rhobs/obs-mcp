# Lightspeed Evals

The evaluations testset for the obs-mcp based on [lightspeed-evaluation](https://github.com/lightspeed-core/lightspeed-evaluation).

## Configuration Files

| File           | Description                          |
|----------------|--------------------------------------|
| `system.yaml`  | System prompt and LLM configuration  |
| `evals.yaml`   | Test cases and evaluation criteria   |

## Pre-requisites

- [uv](https://docs.astral.sh/uv/)
- OpenShift cluster with:
  - Thanos Querier or Prometheus accessible
  - Valid kubeconfig or service account credentials
- obs-mcp server running and connected to Prometheus/Thanos Querier, [check readme for the instructions](../../README.md)
- OpenAI API key

## Quickstart

### Install dependencies

``` bash
uv sync
```

### Setup the lightspeed-stack

On a another terminal:

```shell
git clone https://github.com/lightspeed-core/lightspeed-stack.git
cd lightspeed-stack
```

Copy the lightspeed configs from this [repo](../../hack/lightspeed-stack) to above directory

**Note:** Adjust the path where obs-mcp is located accordingly in the below command

```shell
cp ../obs-mcp/hack/lightspeed-stack/lightspeed-stack.yaml lightspeed-stack.yaml
cp ../obs-mcp/hack/lightspeed-stack/run.yaml run.yaml
```

```shell
uv sync --group dev --group llslibdev
export OPENAI_API_KEY="your-api-key-here"
make run
```

### Run the evaluations

> [!TIP]
>
> Keep .caches when tweaking evaluation criteria to avoid re-running expensive LLM calls.

``` shell
export OPENAI_API_KEY="your-api-key-here"
# Deleting the .caches to avoid using old data: might be helpful to keep when
# tweaking the evaluation criteria.
rm -rf .caches
uv run lightspeed-eval --system-config system.yaml --eval-data evals.yaml
```
