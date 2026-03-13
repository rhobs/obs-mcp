# Prompts You Can Try

This document lists example prompts you can use to test obs-mcp when connected to Cursor or another MCP client. These prompts align with the [MCPChecker evals](../evals/mcpchecker/) and exercise different obs-mcp tools.

For metric discovery tips (e.g. regex behavior, common question → metric mapping), see [METRICS_REFERENCE.md](METRICS_REFERENCE.md).

## Metric Discovery

- List all available Prometheus metrics that contain 'kube' in the name.
- What node-related metrics are available in Prometheus?

## Label Exploration

- What labels are available for the kube_pod_info metric?
- What are the unique namespace values for the kube_pod_info metric?
- How many time series exist for the kube_pod_info metric? Show the cardinality.

## Queries

- Which pods are using the most CPU?
- Which pods are stuck in pending state?
- Which pods are receiving the most network traffic?
- How many head series does Prometheus have?
- What is the current storage size of the Prometheus WAL?
- How many requests per second are being made to Prometheus?
- How many pods were created in the last 5 minutes?
- Which pods were crashlooping in the last 5 minutes?

## Alerts

- Are there any currently firing alerts in the cluster?
- Are there any active silences in Alertmanager?
- Check if there are any firing alerts. If there are, investigate the related metrics for the most critical alert and summarize what's happening.

## Multi-Step Investigation

These prompts are part of the eval suite (hard difficulty) and test complex reasoning:

- Which namespace is consuming the most CPU and memory? Show me the top namespace for each.
- Is the cluster healthy? Give me an overview of any issues.

## Bonus: Additional Prompts

These prompts go beyond the eval suite and test more complex workflows:

- What's the memory usage of pods in the monitoring namespace?
- Show me the container restart count for all pods over the last hour.
- Which nodes have the highest CPU utilization?
- What's the disk usage on the cluster nodes?
- Are any containers in OOMKilled state?
- How many pods are running in the cluster?
