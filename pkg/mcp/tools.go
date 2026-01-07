package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// ListMetricsOutput defines the output schema for the list_metrics tool.
type ListMetricsOutput struct {
	Metrics []string `json:"metrics" jsonschema:"description=List of all available metric names in Prometheus"`
}

// InstantQueryOutput defines the output schema for the execute_instant_query tool.
type InstantQueryOutput struct {
	ResultType string          `json:"resultType" jsonschema:"description=The type of result returned (e.g. vector, scalar, string)"`
	Result     []InstantResult `json:"result" jsonschema:"description=The query results as an array of instant values"`
	Warnings   []string        `json:"warnings,omitempty" jsonschema:"description=Any warnings generated during query execution"`
}

// InstantResult represents a single instant query result.
type InstantResult struct {
	Metric map[string]string `json:"metric" jsonschema:"description=The metric labels"`
	Value  []interface{}     `json:"value" jsonschema:"description=[timestamp, value] pair for the instant query"`
}

// LabelNamesOutput defines the output schema for the get_label_names tool.
type LabelNamesOutput struct {
	Labels []string `json:"labels" jsonschema:"description=List of label names available for the specified metric or all metrics"`
}

// LabelValuesOutput defines the output schema for the get_label_values tool.
type LabelValuesOutput struct {
	Values []string `json:"values" jsonschema:"description=List of unique values for the specified label"`
}

// SeriesOutput defines the output schema for the get_series tool.
type SeriesOutput struct {
	Series      []map[string]string `json:"series" jsonschema:"description=List of time series matching the selector, each series is a map of label names to values"`
	Cardinality int                 `json:"cardinality" jsonschema:"description=Total number of series matching the selector"`
}

// RangeQueryOutput defines the output schema for the execute_range_query tool.
type RangeQueryOutput struct {
	ResultType string         `json:"resultType" jsonschema:"description=The type of result returned (e.g. matrix, vector, scalar)"`
	Result     []SeriesResult `json:"result" jsonschema:"description=The query results as an array of time series"`
	Warnings   []string       `json:"warnings,omitempty" jsonschema:"description=Any warnings generated during query execution"`
}

// SeriesResult represents a single time series result from a range query.
type SeriesResult struct {
	Metric map[string]string `json:"metric" jsonschema:"description=The metric labels"`
	Values [][]any           `json:"values" jsonschema:"description=Array of [timestamp, value] pairs"`
}

func CreateListMetricsTool() mcp.Tool {
	tool := mcp.NewTool("list_metrics",
		mcp.WithDescription(`List all available metric names in Prometheus. This is your starting point for ALL observability questions.

WHEN TO USE:
- User asks about error rates, latency, performance, availability, or any system behavior
- You don't know what metrics are available
- Starting any new investigation

TYPICAL WORKFLOW FOR "Why is X system having high error rates?":
1. Call list_metrics to find relevant metrics (look for: http_requests_total, errors_total, failed_requests, etc.)
2. Use get_label_names to find how to filter by system/service
3. Use get_label_values to find the specific system name
4. Craft PromQL query to calculate error rate

COMMON METRIC PATTERNS TO LOOK FOR:

Application Metrics:
- Error rates: *_errors_total, *_requests_total (with status labels), *_failed_*, *_failures_total
- Latency: *_duration_*, *_latency_*, *_request_seconds_*, *_response_time_* (often histograms)
- Availability: up, *_available, *_health, *_up
- Throughput: *_requests_total, *_bytes_*, *_messages_*, *_transactions_total

Kubernetes/OpenShift Core Metrics (from kube-state-metrics):
- Pod state: kube_pod_status_phase, kube_pod_status_ready, kube_pod_container_status_restarts_total
- Deployments: kube_deployment_status_replicas, kube_deployment_status_replicas_available
- Node capacity: kube_node_status_capacity, kube_node_status_allocatable
- Resource requests/limits: kube_pod_container_resource_requests, kube_pod_container_resource_limits
- PVC: kube_persistentvolumeclaim_status_phase
- Namespace quotas: kube_resourcequota

Container Metrics (from kubelet/cAdvisor):
- CPU: container_cpu_usage_seconds_total, container_cpu_cfs_throttled_seconds_total
- Memory: container_memory_usage_bytes, container_memory_working_set_bytes, container_memory_rss
- Network: container_network_receive_bytes_total, container_network_transmit_bytes_total
- Filesystem: container_fs_usage_bytes, container_fs_limit_bytes
- OOMKills: container_oom_events_total (critical for memory issues)

Node Metrics (from node-exporter):
- CPU: node_cpu_seconds_total, node_load1, node_load5, node_load15
- Memory: node_memory_MemAvailable_bytes, node_memory_MemTotal_bytes
- Disk: node_filesystem_avail_bytes, node_disk_io_time_seconds_total
- Network: node_network_receive_bytes_total, node_network_transmit_bytes_total

OpenShift-Specific:
- Routes: haproxy_backend_http_responses_total (if using OpenShift Router)
- Build metrics: openshift_build_*
- Image registry: openshift_registry_*
`),
		mcp.WithOutputSchema[ListMetricsOutput](),
	)
	// workaround for tool with no parameter
	// see https://github.com/containers/kubernetes-mcp-server/pull/341/files#diff-8f8a99cac7a7cbb9c14477d40539efa1494b62835603244ba9f10e6be1c7e44c
	tool.InputSchema = mcp.ToolInputSchema{}
	tool.RawInputSchema = []byte(`{"type":"object","properties":{}}`)
	return tool
}

func CreateExecuteInstantQueryTool() mcp.Tool {
	return mcp.NewTool("execute_instant_query",
		mcp.WithDescription(`Execute a PromQL instant query to get current/point-in-time values. Use for "what is the current state?" questions.

WHEN TO USE:
- Questions about current state: "What is the current error rate?"
- Latest value queries: "How much memory is the API using right now?"
- Point-in-time snapshots: "How many pods are running?"
- Comparisons at a specific time: "Which services have the highest CPU usage?"

WHEN NOT TO USE:
- Trends over time: Use execute_range_query
- Rate calculations over time: Use execute_range_query
- Historical analysis: Use execute_range_query

ANSWERING "What is the current error rate for the API service?":

After discovery workflow:
- Found metric: http_requests_total (counter)
- Found labels: service="api-gateway", status codes

Query for current error rate (errors per second right now):
query: rate(http_requests_total{service="api-gateway",status=~"5.."}[5m]) / rate(http_requests_total{service="api-gateway"}[5m])
time: omit or "NOW"

Returns: Single value (e.g., 0.023 = 2.3% error rate)

EXAMPLE QUERIES BY QUESTION TYPE:

APPLICATION METRICS:

1. "What is current error rate?" (counter):
   (rate(http_requests_total{namespace="prod",status=~"5.."}[5m]) / rate(http_requests_total{namespace="prod"}[5m])) * 100

2. "What is current request rate?" (counter):
   sum(rate(http_requests_total{namespace="production"}[5m])) by (service)

3. "Which endpoints have highest latency?" (histogram):
   histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{namespace="prod"}[5m])) by (le, handler))

4. "Top 5 services by traffic?" (counter):
   topk(5, sum(rate(http_requests_total{namespace="prod"}[5m])) by (service))

KUBERNETES POD/CONTAINER METRICS:

5. "How much memory are my pods using?" (gauge):
   sum(container_memory_working_set_bytes{namespace="production",container!="",container!="POD"}) by (pod)
   Note: container!="" and container!="POD" filters are CRITICAL to avoid double-counting

6. "Which pods are using the most memory?" (gauge):
   topk(10, sum(container_memory_working_set_bytes{namespace="prod",container!="",container!="POD"}) by (pod))

7. "What is current CPU usage per pod?" (counter):
   sum(rate(container_cpu_usage_seconds_total{namespace="production",container!="",container!="POD"}[5m])) by (pod)

8. "Which pods are being CPU throttled?" (counter):
   sum(rate(container_cpu_cfs_throttled_seconds_total{namespace="prod"}[5m])) by (pod) > 0.1
   Note: >0.1 means throttled >10% of the time, indicates CPU limits too low

9. "How many pods are running in namespace?" (gauge):
   count(kube_pod_status_phase{namespace="production",phase="Running"})

10. "How many pods are NOT ready?" (gauge):
    count(kube_pod_status_ready{namespace="prod",condition="false"})

11. "Which pods are crashlooping?" (gauge):
    kube_pod_container_status_waiting_reason{namespace="production",reason="CrashLoopBackOff"} == 1

12. "How many times has pod restarted recently?" (counter):
    kube_pod_container_status_restarts_total{namespace="prod",pod="my-pod-xyz"}

13. "Is pod being OOMKilled?" (counter):
    rate(container_oom_events_total{namespace="prod",pod=~"my-pod.*"}[5m]) > 0

KUBERNETES NODE METRICS:

14. "What is current node memory usage?" (gauge):
    (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100

15. "What is current node CPU usage?" (counter):
    (1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) by (instance)) * 100

16. "Which nodes are full on disk?" (gauge):
    (node_filesystem_size_bytes{mountpoint="/"} - node_filesystem_avail_bytes{mountpoint="/"}) / node_filesystem_size_bytes{mountpoint="/"} * 100 > 80

KUBERNETES DEPLOYMENT/WORKLOAD:

17. "Are all deployment replicas ready?" (gauge):
    kube_deployment_status_replicas_available{namespace="prod",deployment="api"} / kube_deployment_spec_replicas{namespace="prod",deployment="api"}
    Note: Returns 1.0 if all replicas ready, <1.0 if some unavailable

18. "Which deployments have missing replicas?" (gauge):
    kube_deployment_spec_replicas{namespace="production"} - kube_deployment_status_replicas_available{namespace="production"} > 0

19. "How many replicas should deployment have?" (gauge):
    kube_deployment_spec_replicas{namespace="prod",deployment="api"}

AVAILABILITY/HEALTH:

20. "Which services are down?" (gauge):
    up{job=~"kubernetes-.*"} == 0

21. "What is current uptime?" (gauge):
    up{job="my-service"}
    Note: Returns 1 if up, 0 if down

TIME WINDOW NOTE:
Even for "instant" queries, rate() still needs a time window like [5m].
This means "rate over last 5 minutes evaluated at this instant".

INTERPRETING RESULTS:
Results are single values at the query time.
- One series per unique label combination
- Values represent the state at that moment
- For counters with rate(): value is per-second rate
- For gauges: value is the actual measurement

COMPARISON WITH RANGE QUERY:
- Instant: "What is the error rate RIGHT NOW?" → Single number per series
- Range: "What was the error rate over the last hour?" → Time series with many points

AGGREGATIONS:
Since results are instant, you often want to aggregate:
- sum by (service): Total across all instances, grouped by service
- avg by (namespace): Average per namespace
- max by (pod): Maximum value per pod
- topk(5, ...): Top 5 highest values
- count(...): Count how many series match a condition

NEXT STEP: Present the current values to the user. If they need historical context or trends, use execute_range_query.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("PromQL query string to evaluate at a single point in time"),
		),
		mcp.WithString("time",
			mcp.Description("Evaluation time as RFC3339 or Unix timestamp. Omit or use 'NOW' for current time."),
		),
		mcp.WithOutputSchema[InstantQueryOutput](),
	)
}

func CreateExecuteRangeQueryTool() mcp.Tool {
	return mcp.NewTool("execute_range_query",
		mcp.WithDescription(`Execute a PromQL range query to get time-series data over a period. This is the FINAL step after metric discovery and understanding.

WHEN TO USE:
- Answering questions about trends, rates, patterns over time
- Calculating error rates, request rates, throughput
- Computing latency percentiles over time
- Analyzing SLI/SLO compliance
- Investigating "why" questions that need historical context

TIME PARAMETERS:
- For recent data: Use 'duration' (e.g., duration="1h" for last hour, "6h" for last 6 hours)
- For specific periods: Use 'start' and 'end' as RFC3339 timestamps
- 'step': Resolution of data points. For 1h duration use step="15s" or "30s", for 24h use step="5m"

COMPLETE EXAMPLE - "Why is the API service having high error rates?":

After discovery workflow (list_metrics → get_label_names → get_label_values):
- Found metric: http_requests_total (counter type)
- Found labels: service="api-gateway", status codes 200,500,502,503
- Want: Error rate over last 6 hours

Query construction:
1. Error requests per second: rate(http_requests_total{service="api-gateway",status=~"5.."}[5m])
2. Total requests per second: rate(http_requests_total{service="api-gateway"}[5m])
3. Error percentage: (rate(http_requests_total{service="api-gateway",status=~"5.."}[5m]) / rate(http_requests_total{service="api-gateway"}[5m])) * 100

Execute:
query: (rate(http_requests_total{service="api-gateway",status=~"5.."}[5m]) / rate(http_requests_total{service="api-gateway"}[5m])) * 100
step: "30s"
duration: "6h"

PROMQL PATTERNS BY QUESTION TYPE:

APPLICATION METRICS:

1. ERROR RATE (counter metric):
   Query: (rate(http_requests_total{service="X",status=~"5.."}[5m]) / rate(http_requests_total{service="X"}[5m])) * 100
   Returns: Error percentage over time

2. REQUEST RATE/THROUGHPUT (counter metric):
   Query: sum(rate(http_requests_total{namespace="production"}[5m])) by (service)
   Returns: Requests per second per service

3. P95/P99 LATENCY (histogram metric):
   Query: histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{namespace="prod"}[5m])) by (le, service))
   Returns: 95th percentile latency per service

4. ERROR BREAKDOWN BY TYPE:
   Query: sum(rate(http_requests_total{namespace="prod",status=~"5.."}[5m])) by (status, handler)
   Returns: Error rate grouped by status code and endpoint

KUBERNETES POD/CONTAINER METRICS:

5. POD MEMORY USAGE (gauge):
   Query: sum(container_memory_working_set_bytes{namespace="production",container!="",container!="POD"}) by (pod)
   Returns: Memory usage per pod (working set is what Kubernetes uses for OOM decisions)
   Note: Exclude container="" and container="POD" to avoid double counting

6. POD MEMORY USAGE vs LIMIT (gauge):
   Query: sum(container_memory_working_set_bytes{namespace="prod",pod=~"api.*"}) by (pod) / sum(container_spec_memory_limit_bytes{namespace="prod",pod=~"api.*"}) by (pod) * 100
   Returns: Memory usage as percentage of limit per pod

7. POD CPU USAGE (gauge counter):
   Query: sum(rate(container_cpu_usage_seconds_total{namespace="production",container!="",container!="POD"}[5m])) by (pod)
   Returns: CPU cores used per pod

8. POD CPU THROTTLING (counter):
   Query: sum(rate(container_cpu_cfs_throttled_seconds_total{namespace="prod"}[5m])) by (pod, container)
   Returns: Seconds per second of throttling (>0.1 indicates CPU limits too low)

9. POD RESTART RATE (counter):
   Query: sum(rate(kube_pod_container_status_restarts_total{namespace="production"}[15m])) by (pod, container)
   Returns: Container restarts per second (any value >0 indicates problems)

10. POD NETWORK TRAFFIC (counter):
    Query: sum(rate(container_network_receive_bytes_total{namespace="prod"}[5m])) by (pod)
    Returns: Network receive rate in bytes/sec per pod

11. PODS NOT READY (gauge):
    Query: count(kube_pod_status_phase{namespace="production",phase!="Running"}) by (phase)
    Returns: Count of pods in non-Running phases

12. PODS IN CRASHLOOPBACKOFF (gauge):
    Query: kube_pod_container_status_waiting_reason{namespace="prod",reason="CrashLoopBackOff"}
    Returns: Pods stuck in CrashLoopBackOff

KUBERNETES DEPLOYMENT/WORKLOAD METRICS:

13. DEPLOYMENT DESIRED vs AVAILABLE REPLICAS (gauge):
    Query: (kube_deployment_status_replicas_available{namespace="prod"} / kube_deployment_spec_replicas{namespace="prod"}) * 100
    Returns: Percentage of desired replicas that are available

14. DEPLOYMENT ROLLOUT ISSUES (gauge):
    Query: kube_deployment_spec_replicas{namespace="prod"} - kube_deployment_status_replicas_available{namespace="prod"}
    Returns: Number of missing replicas per deployment (>0 indicates rollout problems)

NODE METRICS:

15. NODE MEMORY AVAILABLE (gauge):
    Query: node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes * 100
    Returns: Percentage of available memory per node

16. NODE CPU USAGE (gauge counter):
    Query: (1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) by (instance)) * 100
    Returns: CPU usage percentage per node

17. NODE FILESYSTEM USAGE (gauge):
    Query: (node_filesystem_size_bytes{mountpoint="/"} - node_filesystem_avail_bytes{mountpoint="/"}) / node_filesystem_size_bytes{mountpoint="/"} * 100
    Returns: Root filesystem usage percentage per node

18. NODE NETWORK ERRORS (counter):
    Query: rate(node_network_transmit_errs_total[5m]) + rate(node_network_receive_errs_total[5m])
    Returns: Network errors per second per interface

RESOURCE QUOTAS AND LIMITS:

19. NAMESPACE CPU REQUEST USAGE (gauge):
    Query: sum(kube_pod_container_resource_requests{namespace="prod",resource="cpu"}) by (namespace)
    Returns: Total CPU requests in namespace (in cores)

20. NAMESPACE MEMORY REQUEST USAGE (gauge):
    Query: sum(kube_pod_container_resource_requests{namespace="prod",resource="memory"}) by (namespace) / 1024 / 1024 / 1024
    Returns: Total memory requests in namespace (in GiB)

21. POD MEMORY OOM KILLS (counter):
    Query: sum(rate(container_oom_events_total{namespace="prod"}[5m])) by (pod, container)
    Returns: OOM kill events per second (any value indicates memory limits too low)

OPENSHIFT-SPECIFIC:

22. ROUTE ERROR RATE (if using OpenShift Router):
    Query: sum(rate(haproxy_backend_http_responses_total{route="my-route",code="5xx"}[5m])) / sum(rate(haproxy_backend_http_responses_total{route="my-route"}[5m]))
    Returns: Error rate for OpenShift route

TROUBLESHOOTING PATTERNS:

23. "Which pods are using the most memory?":
    Query: topk(10, sum(container_memory_working_set_bytes{namespace="prod",container!="",container!="POD"}) by (pod))
    Returns: Top 10 pods by memory usage

24. "Which deployments have pods not ready?":
    Query: kube_deployment_status_replicas{namespace="prod"} - kube_deployment_status_replicas_available{namespace="prod"} > 0
    Returns: Deployments with unavailable pods

25. "What's the pod restart rate over last hour?" (looking for crashers):
    Query: increase(kube_pod_container_status_restarts_total{namespace="prod"}[1h]) > 0
    Returns: Number of restarts in last hour (filter >0 to show only restarting pods)

STEP SIZE RECOMMENDATIONS:
- Last 1 hour: step="15s" or "30s"
- Last 6 hours: step="30s" or "1m"
- Last 24 hours: step="1m" or "5m"
- Last 7 days: step="5m" or "15m"
- Last 30 days: step="1h"

INTERPRETING RESULTS:
Results are time-series with timestamps and values. Each series has labels identifying it.
- Multiple series: Different label combinations (different pods, handlers, etc.)
- Aggregate if too many series: Use sum/avg by (key_labels)
- Look for spikes, trends, correlations in values over time

COMPLETE KUBERNETES TROUBLESHOOTING WORKFLOWS:

SCENARIO 1: "Why are my pods crashing?"
Step 1: Check restart rate
  Query: increase(kube_pod_container_status_restarts_total{namespace="prod"}[1h]) > 0
Step 2: Check OOM kills
  Query: rate(container_oom_events_total{namespace="prod"}[15m])
Step 3: Check memory usage vs limit
  Query: container_memory_working_set_bytes{namespace="prod",pod=~"problematic.*",container!=""} / container_spec_memory_limit_bytes{namespace="prod",pod=~"problematic.*",container!=""}
Step 4: Check pod phase over time
  Query: kube_pod_status_phase{namespace="prod",pod=~"problematic.*"}
Conclusion: If OOM kills >0 and memory near limit → increase memory limits

SCENARIO 2: "Why is my service slow?"
Step 1: Check if pods are ready
  Query: kube_deployment_status_replicas_available{namespace="prod",deployment="api"} / kube_deployment_spec_replicas{namespace="prod",deployment="api"}
Step 2: Check CPU throttling
  Query: rate(container_cpu_cfs_throttled_seconds_total{namespace="prod",pod=~"api.*"}[5m])
Step 3: Check request latency
  Query: histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{namespace="prod",service="api"}[5m])) by (le))
Step 4: Check error rate
  Query: rate(http_requests_total{namespace="prod",service="api",status=~"5.."}[5m]) / rate(http_requests_total{namespace="prod",service="api"}[5m])
Conclusion: If CPU throttling >0.1 → increase CPU limits; if errors high → check logs

SCENARIO 3: "Why is my node running out of memory?"
Step 1: Node memory available
  Query: node_memory_MemAvailable_bytes{instance=~"node-xyz.*"} / node_memory_MemTotal_bytes{instance=~"node-xyz.*"}
Step 2: Top memory consumers on node
  Query: topk(10, sum(container_memory_working_set_bytes{node="node-xyz",container!="",container!="POD"}) by (namespace, pod))
Step 3: Compare requests vs usage on node
  Query: sum(kube_pod_container_resource_requests{node="node-xyz",resource="memory"}) vs sum(container_memory_working_set_bytes{node="node-xyz",container!=""})
Conclusion: If usage >> requests → pods need better memory requests; if many pods → node needs more capacity

SCENARIO 4: "Why is my deployment not scaling?"
Step 1: Check desired vs available replicas
  Query: kube_deployment_spec_replicas{namespace="prod",deployment="api"} - kube_deployment_status_replicas_available{namespace="prod",deployment="api"}
Step 2: Check pending pods
  Query: kube_pod_status_phase{namespace="prod",phase="Pending"}
Step 3: Check node capacity
  Query: sum(kube_node_status_allocatable{resource="memory"}) - sum(kube_pod_container_resource_requests{resource="memory"})
Conclusion: If pending pods and low node capacity → need more nodes; if no pending pods → check HPA settings

SCENARIO 5: "Why is my namespace hitting quota?"
Step 1: Check current CPU requests
  Query: sum(kube_pod_container_resource_requests{namespace="prod",resource="cpu"})
Step 2: Check quota limit
  Query: kube_resourcequota{namespace="prod",resource="requests.cpu",type="hard"}
Step 3: Top CPU requesters
  Query: topk(10, sum(kube_pod_container_resource_requests{namespace="prod",resource="cpu"}) by (pod))
Conclusion: Compare current vs quota → either increase quota or reduce requests

NEXT STEP: Analyze results and present findings to user. If investigating further, may need more queries with different filters or metrics.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("PromQL query string to evaluate over a time range"),
		),
		mcp.WithString("step",
			mcp.Required(),
			mcp.Description("Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps."),
			mcp.Pattern(`^\d+[smhdwy]$`),
		),
		mcp.WithString("start",
			mcp.Description("Start time as RFC3339 or Unix timestamp (optional)"),
		),
		mcp.WithString("end",
			mcp.Description("End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time."),
		),
		mcp.WithString("duration",
			mcp.Description("Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional)"),
			mcp.Pattern(`^\d+[smhdwy]$`),
		),
		mcp.WithOutputSchema[RangeQueryOutput](),
	)
}

func CreateGetLabelNamesTool() mcp.Tool {
	return mcp.NewTool("get_label_names",
		mcp.WithDescription(`Get all label names (dimensions) available for filtering a metric. Essential for scoping queries to specific systems, services, or conditions.

WHEN TO USE:
- After identifying relevant metrics and understanding their types
- To discover how to filter metrics to a specific system/service
- Before constructing label matchers in PromQL

ANSWERING "Why is X system having high error rates?":
- Call this with the error metric (e.g., metric="http_requests_total")
- Look for labels that identify systems: service, app, deployment, job, namespace
- Look for labels that identify error conditions: status, code, result, error_type
- Then use get_label_values to find the exact value for "X system"

COMMON LABEL PATTERNS IN KUBERNETES/OPENSHIFT:

Identity labels (WHO/WHAT):
- namespace: Kubernetes namespace (e.g., "production", "staging", "openshift-monitoring")
- pod: Specific pod name (e.g., "api-gateway-7d9f8b6c5-x9k2l") - usually too granular for aggregation
- container: Container name within a pod (e.g., "app", "sidecar", "istio-proxy")
- deployment: Deployment name (better for aggregation than pod)
- statefulset: StatefulSet name
- daemonset: DaemonSet name
- job/cronjob: Job or CronJob name
- service: Kubernetes service name
- app/app_kubernetes_io_name: Application name (common label)
- component: Component type (e.g., "database", "cache", "api")
- created_by_kind: Resource type that created the pod (Deployment, DaemonSet, etc.)
- created_by_name: Name of the parent resource
- job (Prometheus): Prometheus scrape job name (e.g., "kubernetes-pods", "kubernetes-nodes")
- instance: Scrape target, usually pod IP:port or node name

Classification labels (TYPE/CATEGORY):
- status/status_code/code: HTTP status code (200, 404, 500)
- method: HTTP method (GET, POST, PUT, DELETE)
- handler/endpoint/path/route: API endpoint or URL path
- result/outcome: success, failure, error
- error_type: Specific error classification
- phase: Pod phase (Running, Pending, Failed, Succeeded, Unknown)
- reason: Reason for pod state (OOMKilled, CrashLoopBackOff, ImagePullBackOff)
- condition: Node/pod condition type (Ready, DiskPressure, MemoryPressure, PIDPressure)

Resource labels (WHAT KIND):
- resource: Resource type (cpu, memory, ephemeral-storage, hugepages-*)
- unit: Unit of measurement (bytes, cores, etc.)
- le: Histogram bucket upper bound (less than or equal)
- quantile: Summary quantile value (0.5, 0.9, 0.95, 0.99)

Location labels (WHERE):
- node: Kubernetes node name (e.g., "ip-10-0-1-45.ec2.internal")
- cluster: Cluster identifier
- region/zone: Cloud region/availability zone (e.g., "us-east-1a")
- topology_kubernetes_io_region: Standard topology label
- topology_kubernetes_io_zone: Standard topology label

OpenShift-Specific:
- route: OpenShift route name
- project: OpenShift project (same as namespace)
- openshift_io_*: Various OpenShift metadata labels

FILTERING STRATEGY:
1. Use identity labels to scope to the system: {namespace="prod", service="api"}
2. Use classification labels to filter condition: {status=~"5..", method="POST"}
3. Aggregate using by() or without(): rate(...)[5m]) by (status, handler)

EXAMPLE for "Why is the API service having high error rates?":
- Found metric: http_requests_total (counter)
- get_label_names(metric="http_requests_total") returns: [namespace, service, status, method, handler]
- Next: get_label_values(label="service") to find if "api" or "api-service" is the exact name
- Next: get_label_values(label="status") to see what status codes exist
- Query: rate(http_requests_total{service="api",status=~"5.."}[5m])

NEXT STEP: Use get_label_values for each relevant label to find exact values.`),
		mcp.WithString("metric",
			mcp.Description("Metric name to get label names for. Leave empty to get all label names across all metrics."),
		),
		mcp.WithString("start",
			mcp.Description("Start time for label discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)"),
		),
		mcp.WithString("end",
			mcp.Description("End time for label discovery as RFC3339 or Unix timestamp (optional, defaults to now)"),
		),
		mcp.WithOutputSchema[LabelNamesOutput](),
	)
}

func CreateGetLabelValuesTool() mcp.Tool {
	return mcp.NewTool("get_label_values",
		mcp.WithDescription(`Get all unique values for a specific label. CRITICAL for finding exact names and constructing accurate label matchers.

WHEN TO USE:
- After discovering label names with get_label_names
- To find the exact name of a system/service (user might say "API" but actual label is "api-gateway")
- To see what values exist for filtering (which status codes, which namespaces, etc.)
- To understand cardinality before querying

ANSWERING "Why is X system having high error rates?":
Step 4 in workflow - Finding exact system identifier:
1. User asks about "API service"
2. You found label "service" exists
3. Call get_label_values(label="service", metric="http_requests_total")
4. Results: ["api-gateway", "auth-service", "frontend", "backend"]
5. Match user's "API service" to "api-gateway"
6. Also call get_label_values(label="status") to see error codes
7. Results: ["200", "201", "400", "404", "500", "502", "503"]
8. Now you can construct: rate(http_requests_total{service="api-gateway",status=~"5.."}[5m])

MATCHING USER INPUT TO LABEL VALUES:
Users often use informal names. Find the correct match:
- User: "API" → Label values: ["api-gateway", "api-v2", "public-api"] → Ask user which one or match context
- User: "production" → Label values: ["prod", "production", "prod-us-east"] → "prod" likely matches
- User: "errors" → Label values: ["200", "500", "502", "503"] → Status 5xx codes are errors

COMMON PATTERNS:

Status codes (for error analysis):
- 2xx: Success
- 4xx: Client errors (often not service's fault)
- 5xx: Server errors (SERVICE'S FAULT - use these for error rates)
- Use regex: status=~"5.." for all 5xx errors

Namespaces (for system identification):
- Often: "production", "prod", "staging", "default", "kube-system"
- Scope to relevant environment first

Services/Apps (for system identification):
- Actual deployed service names
- May have prefixes/suffixes: "api-gateway", "auth-service-v2"
- Use contains matching or ask user if ambiguous

REGEX MATCHING IN PROMQL:
- Exact: {service="api-gateway"}
- Regex: {service=~"api.*"} (matches api-gateway, api-v2, etc.)
- Multiple: {status=~"500|502|503"} (specific codes)
- Range: {status=~"5.."} (all 5xx codes)
- Negative: {status!~"2.."} (exclude 2xx success)

NEXT STEP: After getting all label values, construct the PromQL query with execute_range_query or execute_instant_query.`),
		mcp.WithString("label",
			mcp.Required(),
			mcp.Description("Label name to get values for (e.g., 'namespace', 'status', 'method')"),
		),
		mcp.WithString("metric",
			mcp.Description("Metric name to scope the label values to. Leave empty to get values across all metrics."),
		),
		mcp.WithString("start",
			mcp.Description("Start time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)"),
		),
		mcp.WithString("end",
			mcp.Description("End time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to now)"),
		),
		mcp.WithOutputSchema[LabelValuesOutput](),
	)
}

func CreateGetSeriesTool() mcp.Tool {
	return mcp.NewTool("get_series",
		mcp.WithDescription(`Get time series matching selectors and preview cardinality. Use to verify query correctness and avoid expensive operations.

WHEN TO USE:
- Before executing queries with complex label matchers
- When you want to verify your label filters match the right series
- To check cardinality and avoid slow queries
- To see all unique label combinations that match your selector

CARDINALITY GUIDANCE:
- Low (<100 series): Safe, query will be fast
- Medium (100-1000 series): Usually fine, query may take a few seconds
- High (1000-10000 series): May be slow, consider adding more label filters
- Very High (>10000 series): Likely too expensive, MUST add more specific filters

ANSWERING "Why is X system having high error rates?":
Optional verification step before final query:
- You constructed: http_requests_total{service="api-gateway",status=~"5.."}
- Call get_series(matches="http_requests_total{service=\"api-gateway\",status=~\"5..\"}")
- Check cardinality: If 50 series, good! If 5000 series, need to filter more
- Review series to see all label combinations (different methods, handlers, instances)
- Decide if you need to aggregate: sum by (status, handler) or just by (status)

EXAMPLES:

Example 1 - Verify service filter:
get_series(matches="http_requests_total{service=\"api-gateway\"}")
Result: cardinality=120, shows all combinations with different status, method, handler
→ Good cardinality, proceed with query

Example 2 - Check error series:
get_series(matches="http_requests_total{service=\"api-gateway\",status=~\"5..\"}")
Result: cardinality=30, shows series for status=500,502,503 across different handlers
→ Perfect, now you know exactly what will be queried

Example 3 - Too broad:
get_series(matches="container_memory_usage_bytes")
Result: cardinality=50000
→ Too high! Need to add namespace or pod filters

Example 4 - Refined:
get_series(matches="container_memory_usage_bytes{namespace=\"production\"}")
Result: cardinality=500
→ Much better, safe to query

UNDERSTANDING OUTPUT:
Each series shows all its labels. Example:
{__name__="http_requests_total", service="api-gateway", status="500", method="POST", handler="/api/v1/users"}
{__name__="http_requests_total", service="api-gateway", status="502", method="GET", handler="/api/v1/orders"}

This tells you:
- What label combinations exist
- Whether you need to aggregate (multiple handlers → group by handler)
- If your filters are too broad or too narrow

NEXT STEP: If cardinality is acceptable, execute your query with execute_range_query or execute_instant_query.`),
		mcp.WithString("matches",
			mcp.Required(),
			mcp.Description("Series selector as a PromQL series selector (e.g., 'up{job=\"prometheus\"}' or 'http_requests_total{namespace=\"prod\"}'). Can be a comma-separated list of selectors."),
		),
		mcp.WithString("start",
			mcp.Description("Start time for series discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)"),
		),
		mcp.WithString("end",
			mcp.Description("End time for series discovery as RFC3339 or Unix timestamp (optional, defaults to now)"),
		),
		mcp.WithOutputSchema[SeriesOutput](),
	)
}
