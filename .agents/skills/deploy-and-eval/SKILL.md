---
name: deploy-and-eval
description: Build, deploy, and run evals on the obs-mcp test cluster
user-invocable: true
---

# Deploy and Evaluate obs-mcp

Follow these steps to build, deploy, and run evals.

## Step 1: Build and deploy

Build and deploy changes to the test cluster:
```
TAG=$(date +%s) make test-e2e-deploy
```

## Step 2: Set up port forwarding

Set up port forwarding to the deployed service (long-running task, run in background):
```
make test-e2e-pf
```

## Step 3: Run evaluations

Run evals. Ask the user which task, category, or all evals to run, then execute:
```
# Run a specific task:
make run-mcpchecker-eval TASK=<task> RUNS=<number of iterations, default: 1>
# Run all tasks in a category:
make run-mcpchecker-eval CATEGORY=<category> RUNS=<number of iterations, default: 1>
# Run all evals:
make run-mcpchecker-eval RUNS=<number of iterations, default: 1>
```
