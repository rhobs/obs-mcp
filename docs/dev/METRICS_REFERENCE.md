# Metrics Reference

A quick reference mapping common questions to Prometheus metrics. Use this when `list_metrics` returns no relevant results—try the suggested regex patterns. Metrics vary by deployment (kube-prometheus, OpenShift, etc.); not all may exist in your cluster.

## list_metrics Regex Tips

Prometheus uses **full-string** regex matching. `kube_pod` does not match `kube_pod_container_status_terminated`. Use:

- **Prefix search:** `kube_pod_container_status.*` (matches any metric starting with that prefix)
- **Substring search:** `.*terminated.*` (matches any metric containing "terminated")

## Common Questions → Metrics

| Question | Suggested Metric(s) | list_metrics regex | Notes |
|----------|---------------------|--------------------|-------|
| OOMKilled containers | `kube_pod_container_status_last_terminated_reason` | `.*terminated_reason.*` | Check `reason="OOMKilled"` label. May not exist in all kube-state-metrics setups. |
| Pending pods | `kube_pod_status_phase` | `kube_pod_status_phase` | Filter `phase="Pending"` |
| Running pods | `kube_pod_status_phase` | `kube_pod_status_phase` | Filter `phase="Running"` |
| Crashlooping pods | `kube_pod_container_status_restarts_total` | `.*restarts.*` | Use range query with `increase()` |
| Pods created | `kube_pod_created` | `kube_pod_created` | Timestamp of pod creation |
| CPU usage (pods) | `container_cpu_usage_seconds_total` or `node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate` | `.*cpu.*` | Raw metric or pre-aggregated recording rule |
| Memory usage (pods) | `container_memory_working_set_bytes` or `node_namespace_pod_container:container_memory_working_set_bytes` | `.*memory.*` | Raw metric or pre-aggregated recording rule |
| Network traffic | `node_network_receive_bytes_total`, `node_network_transmit_bytes_total` | `node_network.*` | |
| Prometheus head series | `prometheus_tsdb_head_series` | `prometheus_tsdb.*` | |
| Prometheus WAL size | `prometheus_tsdb_wal_storage_size_bytes` | `prometheus_tsdb.*` | |
| Prometheus request rate | `prometheus_http_requests_total` | `prometheus_http.*` | Use `rate()` |

## Query Efficiency

Agents should prefer aggregated PromQL over querying individual series. For example:

| Goal | Inefficient (N queries) | Efficient (1 query) |
|------|------------------------|---------------------|
| Top CPU pods | One `execute_range_query` per pod | `topk(5, sum by (pod) (rate(container_cpu_usage_seconds_total[5m])))` |
| Namespace resource usage | One query per namespace | `sum by (namespace) (container_memory_working_set_bytes)` |
| Pod restart rate | One query per pod | `topk(10, increase(kube_pod_container_status_restarts_total[1h]))` |

Use `topk()`, `bottomk()`, `sum by()`, `avg by()`, and `rate()` to answer questions in 1-3 queries instead of one per entity.

## When a Metric Doesn't Exist

If `list_metrics` with the suggested regex returns nothing:

1. The metric may not be scraped in your setup (e.g. `kube_pod_container_status_last_terminated_reason` requires specific kube-state-metrics config).
2. Try broader patterns: `kube.*`, `node.*`, `container.*`.
3. Inform the user that the metric is not available in their cluster.
