<!-- This file is auto-generated. Do not edit manually. -->
<!-- Run 'make generate-tools-doc' to regenerate. -->

# Available Tools

This MCP server exposes the following tools for Prometheus/Thanos, Alertmanager, and Tempo:

> [!NOTE]
> **Types in the tables** follow JSON Schema: `object` is a JSON object (string keys with JSON values); `object[]` is an array of those objects. Scalar types use their usual names (`string`, `number`, `boolean`, and so on). When a field has no explicit schema type (for example a Go `any` payload), this document shows `object` as shorthand for "structured JSON," not a guarantee that only objects are returned at runtime.

## `list_metrics`

> MANDATORY FIRST STEP: List all available metric names in Prometheus.

**Usage Tips:**

- YOU MUST CALL THIS TOOL BEFORE ANY OTHER QUERY TOOL
- This tool MUST be called first for EVERY observability question to: 1. Discover what metrics actually exist in this environment 2. Find the EXACT metric name to use in queries 3. Avoid querying non-existent metrics 4. The 'name_regex' parameter should always be provided, and be a best guess of what the metric would be named like. 5. Do not use a blanket regex like .* or .+ in the 'name_regex' parameter. Use specific ones like kube.*, node.*, etc.
- REGEX PATTERN GUIDANCE: - Prometheus metrics are typically prefixed (e.g., 'prometheus_tsdb_head_series', 'kube_pod_status_phase') - To match metrics CONTAINING a substring, use wildcards: '.*tsdb.*' matches 'prometheus_tsdb_head_series' - Without wildcards, the pattern matches EXACTLY: 'tsdb' only matches a metric literally named 'tsdb' (which rarely exists) - Common patterns: 'kube_pod.*' (pods), '.*memory.*' (memory-related), 'node_.*' (node metrics) - If you get empty results, try adding '.*' before/after your search term
- NEVER skip this step. NEVER guess metric names. Metric names vary between environments.
- After calling this tool: 1. Search the returned list for relevant metrics 2. Use the EXACT metric name found in subsequent queries 3. If no relevant metric exists, inform the user

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `name_regex` | `string` | ✅ | Regex pattern to filter metric names. IMPORTANT: Metric names are typically prefixed (e.g., 'prometheus_tsdb_head_series'). Use wildcards to match substrings: '.*tsdb.*' matches any metric containing 'tsdb', while 'tsdb' only matches the exact string 'tsdb'. Examples: 'http_.*' (starts with http_), '.*memory.*' (contains memory), 'node_.*' (starts with node_). This parameter is required. Don't pass in blanket regex like '.*' or '.+'. |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `metrics` | `string[]` | List of all available metric names |

---

## `execute_instant_query`

> Execute a PromQL instant query to get current/point-in-time values.

**Usage Tips:**

- PREREQUISITE: You MUST call list_metrics first to verify the metric exists
- WHEN TO USE: - Current state questions: "What is the current error rate?" - Point-in-time snapshots: "How many pods are running?" - Latest values: "Which pods are in Pending state?"
- The 'query' parameter MUST use metric names that were returned by list_metrics.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `query` | `string` | ✅ | PromQL query string using metric names verified via list_metrics |
| `time` | `string` |  | Evaluation time as RFC3339 or Unix timestamp. Omit or use 'NOW' for current time. |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `result` | `object[]` | The query results as an array of instant values |
| `resultType` | `string` | The type of result returned (e.g. vector, scalar, string) |
| `warnings` | `string[]` | Any warnings generated during query execution |

---

## `execute_range_query`

> Execute a PromQL range query to get time-series data over a period.

**Usage Tips:**

- PREREQUISITE: You MUST call list_metrics first to verify the metric exists
- WHEN TO USE: - Trends over time: "What was CPU usage over the last hour?" - Rate calculations: "How many requests per second?" - Historical analysis: "Were there any restarts in the last 5 minutes?"
- TIME PARAMETERS: - 'duration': Look back from now (e.g., "5m", "1h", "24h") - 'step': Data point resolution (e.g., "1m" for 1-hour duration, "5m" for 24-hour duration)
- The 'query' parameter MUST use metric names that were returned by list_metrics.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `query` | `string` | ✅ | PromQL query string using metric names verified via list_metrics |
| `step` | `string` | ✅ | Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps. |
| `duration` | `string` |  | Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional) |
| `end` | `string` |  | End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time. |
| `start` | `string` |  | Start time as RFC3339 or Unix timestamp (optional) |

> [!NOTE]
> Parameters with patterns must match: `^\d+[smhdwy]$`

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `result` | `object[]` | The query results as an array of time series |
| `resultType` | `string` | The type of result returned: matrix or vector or scalar |
| `summary` | `object[]` | Summary statistics for each time series (when summarize flag is enabled) |
| `warnings` | `string[]` | Any warnings generated during query execution |

---

## `show_timeseries`

> Display the results as an interactive timeseries chart.

**Usage Tips:**

- This tool works like execute_range_query but renders the results as a visual chart in the UI clients. Use it when the user wants to see a graph or visualization of time-series data and to use visuals to provide the answer. Use the show_timeseries as the last tool call after all the other Prometheus tool calls where finalized.
- TIME PARAMETERS: - 'duration': Look back from now (e.g., "5m", "1h", "24h") - 'step': Data point resolution (e.g., "1m" for 1-hour duration, "5m" for 24-hour duration) - 'title': A descriptive chart title (e.g., "API Error Rate Over Last Hour") - 'description': An explanation of the chart's meaning or context (e.g., "Shows the rate of HTTP 5xx errors per second, broken down by pod")
- The 'query' parameter MUST be a range query and must use metric names that were returned by list_metrics.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `query` | `string` | ✅ | PromQL query string using metric names verified via list_metrics |
| `step` | `string` | ✅ | Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps. |
| `description` | `string` |  | Explanation of the chart's meaning or context (e.g., 'Shows the rate of HTTP 5xx errors per second, broken down by pod'). Displayed below the title when provided. |
| `duration` | `string` |  | Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional) |
| `end` | `string` |  | End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time. |
| `start` | `string` |  | Start time as RFC3339 or Unix timestamp (optional) |
| `title` | `string` |  | Human-readable chart title describing what the query shows (e.g., 'API Error Rate Over Last Hour'). Displayed above the chart when provided. |

> [!NOTE]
> Parameters with patterns must match: `^\d+[smhdwy]$`

---

## `get_label_names`

> Get all label names (dimensions) available for filtering a metric.

**Usage Tips:**

- WHEN TO USE (after calling list_metrics): - To discover how to filter metrics (by namespace, pod, service, etc.) - Before constructing label matchers in PromQL queries
- The 'metric' parameter should use a metric name from list_metrics output.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `end` | `string` |  | End time for label discovery as RFC3339 or Unix timestamp (optional, defaults to now) |
| `metric` | `string` |  | Metric name (from list_metrics) to get label names for. Leave empty for all metrics. |
| `start` | `string` |  | Start time for label discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago) |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `labels` | `string[]` | List of label names available for the specified metric or all metrics |

---

## `get_label_values`

> Get all unique values for a specific label.

**Usage Tips:**

- WHEN TO USE (after calling list_metrics and get_label_names): - To find exact label values for filtering (namespace names, pod names, etc.) - To see what values exist before constructing queries
- The 'metric' parameter should use a metric name from list_metrics output.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `label` | `string` | ✅ | Label name (from get_label_names) to get values for |
| `end` | `string` |  | End time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to now) |
| `metric` | `string` |  | Metric name (from list_metrics) to scope the label values to. Leave empty for all metrics. |
| `start` | `string` |  | Start time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago) |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `values` | `string[]` | List of unique values for the specified label |

---

## `get_series`

> Get time series matching selectors and preview cardinality.

**Usage Tips:**

- WHEN TO USE (optional, after calling list_metrics): - To verify label filters match expected series before querying - To check cardinality and avoid slow queries
- CARDINALITY GUIDANCE: - <100 series: Safe - 100-1000: Usually fine - >1000: Add more label filters
- The selector should use metric names from list_metrics output.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `matches` | `string` | ✅ | PromQL series selector using metric names from list_metrics |
| `end` | `string` |  | End time for series discovery as RFC3339 or Unix timestamp (optional, defaults to now) |
| `start` | `string` |  | Start time for series discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago) |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `cardinality` | `integer` | Total number of series matching the selector |
| `series` | `object[]` | List of time series matching the selector, each series is a map of label names to values |

---

## `get_alerts`

> Get alerts from Alertmanager.

**Usage Tips:**

- WHEN TO USE: - START HERE when investigating issues: if the user asks about things breaking, errors, failures, outages, services being down, or anything going wrong in the cluster - When the user mentions a specific alert name - use this tool to get the alert's full labels (namespace, pod, service, etc.) which are essential for further investigation with other tools - To see currently firing alerts in the cluster - To check which alerts are active, silenced, or inhibited - To understand what's happening before diving into metrics or logs
- INVESTIGATION TIP: Alert labels often contain the exact identifiers (pod names, namespaces, job names) needed for targeted queries with prometheus tools.
- FILTERING: - Use 'active' to filter for only active alerts (not resolved) - Use 'silenced' to filter for silenced alerts - Use 'inhibited' to filter for inhibited alerts - Use 'filter' to apply label matchers (e.g., "alertname=HighCPU") - Use 'receiver' to filter alerts by receiver name
- All filter parameters are optional. Without filters, all alerts are returned.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `active` | `boolean` |  | Filter for active alerts only (true/false, optional) |
| `filter` | `string` |  | Label matchers to filter alerts (e.g., 'alertname=HighCPU', optional) |
| `inhibited` | `boolean` |  | Filter for inhibited alerts only (true/false, optional) |
| `receiver` | `string` |  | Receiver name to filter alerts (optional) |
| `silenced` | `boolean` |  | Filter for silenced alerts only (true/false, optional) |
| `unprocessed` | `boolean` |  | Filter for unprocessed alerts only (true/false, optional) |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `alerts` | `object[]` | List of alerts from Alertmanager |

---

## `get_silences`

> Get silences from Alertmanager.

**Usage Tips:**

- WHEN TO USE: - To see which alerts are currently silenced - To check active, pending, or expired silences - To investigate why certain alerts are not firing notifications
- FILTERING: - Use 'filter' to apply label matchers to find specific silences
- Silences are used to temporarily mute alerts based on label matchers. This tool helps you understand what is currently silenced in your environment.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `filter` | `string` |  | Label matchers to filter silences (e.g., 'alertname=HighCPU', optional) |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `silences` | `object[]` | List of silences from Alertmanager |

---

## `tempo_list_instances`

> List all Tempo instances available in the Kubernetes cluster.
Call this tool first to discover available Tempo instances before using other Tempo tools,
as the returned namespace, name, and tenant values are required parameters for all other Tempo tools.
Always print the output of this tool in a table.

|  |  |
| :--- | :--- |
| **Parameters** | None |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `instances` | `object[]` | List of available Tempo instances |

---

## `tempo_get_trace_by_id`

> Retrieve a single distributed trace by its trace ID from Tempo.
Returns the full trace with all its spans, including service names, operation names, durations, and attributes.
Use this tool when you already have a specific trace ID, e.g. from search results or logs.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `tempoName` | `string` | ✅ | The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names. |
| `tempoNamespace` | `string` | ✅ | The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces. |
| `traceid` | `string` | ✅ | The trace ID to retrieve, e.g. "26dad4a0e2b0dd9a440dd5ff203a24a4". |
| `end` | `string` |  | Optional end of the time range in RFC 3339 format, e.g. "2025-01-02T00:00:00Z".<br>Narrows the time range to improve query performance. |
| `start` | `string` |  | Optional start of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".<br>Narrows the time range to improve query performance. |
| `tenant` | `string` |  | The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance. |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `trace` | `object` | The trace data with services, scopes and spans |

---

## `tempo_search_traces`

> Search for distributed traces in Tempo using TraceQL.
Use this tool to find traces matching specific criteria such as service name, HTTP status code, duration, or other span or resource attributes.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `query` | `string` | ✅ | A TraceQL query expression. Format:<br>query: "{ <filters joined by &&> }"<br><br>Filters:<br>- service name:     resource.service.name="<value>" (string, use quotes)<br>- HTTP status code: span.http.response.status_code=<code> (number, no quotes)<br>- duration:         duration><value like 100ms, 2s, 5m> (no quotes)<br>- error status:     status=error (keyword, NO quotes — do NOT write status="error")<br><br>IMPORTANT: status values (error, ok, unset) are keywords, NOT strings. Write status=error, NEVER status="error".<br><br>Operators: =, !=, >, <, >=, <=<br><br>Common attributes:<br>- resource.service.name (service name)<br>- span.http.response.status_code (HTTP response code)<br>- span.http.request.method (HTTP method like GET, POST)<br>- span.url.full (request URL)<br>- duration (trace duration, e.g. 100ms, 2s)<br>- status (trace status: ok, error, unset)<br><br>IMPORTANT: Always wrap filters in curly braces { }.<br>Do NOT use SQL, PromQL, or Lucene syntax.<br>Do NOT omit the "resource." or "span." prefix from attribute names<br><br>If unsure which attributes to filter on, start with {} to return all traces, then use tempo_search_tags to discover available attributes. |
| `tempoName` | `string` | ✅ | The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names. |
| `tempoNamespace` | `string` | ✅ | The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces. |
| `end` | `string` |  | End of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".<br>Use "NOW" for current time.<br>Both start and end should be provided to search the full time range; if omitted, only a small window of recent data is searched. |
| `limit` | `number` |  | Maximum number of traces to return. Defaults to the server-side limit if not specified. |
| `spss` | `number` |  | Maximum number of matching spans to return per trace. |
| `start` | `string` |  | Start of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".<br>Use "NOW" for current time.<br>Both start and end should be provided to search the full time range; if omitted, only a small window of recent data is searched. |
| `tenant` | `string` |  | The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance. |

---

## `tempo_search_tags`

> List available tag names (attribute keys) in Tempo, grouped by scope.
Use this tool to discover which attributes are available for building TraceQL queries with tempo_search_traces.
For example, this tool may reveal tag names like "service.name" (in the "resource" scope) or "http.response.status_code" (in the "span" scope).
To use these in TraceQL queries, prefix them with their scope, e.g. "resource.service.name" or "span.http.response.status_code".

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `tempoName` | `string` | ✅ | The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names. |
| `tempoNamespace` | `string` | ✅ | The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces. |
| `end` | `string` |  | Optional end of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing tags. |
| `limit` | `number` |  | Maximum number of tag names to return per scope. |
| `maxStaleValues` | `number` |  | Maximum number of consecutive blocks without new tag names before the search stops early. Higher values are more thorough but slower. |
| `query` | `string` |  | Optional TraceQL query to filter which traces are considered when listing tags,<br>e.g. '{ resource.service.name="payment-service" }' to only show tags present in traces from the 'payment-service' service. |
| `scope` | `string` |  | Filter tags to a specific scope. One of:<br>"resource" (service-level attributes like service.name),<br>"span" (individual span attributes like http.response.status_code),<br>"intrinsic" (built-in fields like duration, status, name).<br>If omitted, tags from all scopes are returned. |
| `start` | `string` |  | Optional start of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing tags. |
| `tenant` | `string` |  | The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance. |

---

## `tempo_search_tag_values`

> List the known values for a specific tag (attribute key) in Tempo.
Use this tool to discover what values exist for a given tag, e.g. to find all service names (values of "resource.service.name") or all HTTP methods (values of "span.http.request.method").
This is useful for building accurate TraceQL queries with tempo_search_traces.

**Parameters:**

| Parameter | Type | Required | Description |
| :--- | :--- | :---: | :--- |
| `tag` | `string` | ✅ | The fully qualified tag name to get values for, including its scope prefix, e.g. "resource.service.name" or "span.http.response.status_code".<br>Use tempo_search_tags to discover available tag names. |
| `tempoName` | `string` | ✅ | The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names. |
| `tempoNamespace` | `string` | ✅ | The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces. |
| `end` | `string` |  | Optional end of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing values. |
| `limit` | `number` |  | Maximum number of tag values to return. |
| `maxStaleValues` | `number` |  | Maximum number of consecutive blocks without new values before the search stops early. Higher values are more thorough but slower. |
| `query` | `string` |  | Optional TraceQL query to filter which traces are considered when listing values,<br>e.g. '{ resource.service.name="payment-service" }' to only show tag values from the 'payment-service' service. |
| `start` | `string` |  | Optional start of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing values. |
| `tenant` | `string` |  | The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance. |

**Output Schema:**

| Field | Type | Description |
| :--- | :--- | :--- |
| `tagValues` | `object` | Known values for the specified tag, keyed by type |

