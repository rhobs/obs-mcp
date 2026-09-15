<!-- This file is auto-generated. Do not edit manually. -->
<!-- Run 'make generate-tools-doc' to regenerate. -->

# Available Tools

This MCP server exposes the following tools for Prometheus/Thanos, Alertmanager, Loki, Tempo, and OpenTelemetry Collector configuration.

## Quick Reference

| Tool | Category | Description |
| :--- | :--- | :--- |
| [`list_metrics`](#list_metrics) | 📈 Prometheus / Thanos | MANDATORY FIRST STEP: List all available metric names in Prometheus. |
| [`execute_instant_query`](#execute_instant_query) | 📈 Prometheus / Thanos | Execute a PromQL instant query to get current/point-in-time values. |
| [`execute_range_query`](#execute_range_query) | 📈 Prometheus / Thanos | Execute a PromQL range query to get time-series data over a period. |
| [`show_timeseries`](#show_timeseries) | 📈 Prometheus / Thanos | Display the results as an interactive timeseries chart. |
| [`get_label_names`](#get_label_names) | 📈 Prometheus / Thanos | Get all label names (dimensions) available for filtering a metric. |
| [`get_label_values`](#get_label_values) | 📈 Prometheus / Thanos | Get all unique values for a specific label. |
| [`get_series`](#get_series) | 📈 Prometheus / Thanos | Get time series matching selectors and preview cardinality. |
| [`get_alerts`](#get_alerts) | 🔔 Alertmanager | Get alerts from Alertmanager. |
| [`get_silences`](#get_silences) | 🔔 Alertmanager | Get silences from Alertmanager. |
| [`create_silence`](#create_silence) | 🔔 Alertmanager | Create an Alertmanager silence that mutes matching alerts. |
| [`update_silence`](#update_silence) | 🔔 Alertmanager | Update an existing Alertmanager silence (POST /api/v2/silences with silence_id). |
| [`delete_silence`](#delete_silence) | 🔔 Alertmanager | Expire (delete) an Alertmanager silence by UUID. |
| [`tempo_list_instances`](#tempo_list_instances) | 🔍 Tempo (Distributed Tracing) | List all Tempo instances available in the Kubernetes cluster. |
| [`tempo_get_trace_by_id`](#tempo_get_trace_by_id) | 🔍 Tempo (Distributed Tracing) | Retrieve a single distributed trace by its trace ID from Tempo. |
| [`tempo_search_traces`](#tempo_search_traces) | 🔍 Tempo (Distributed Tracing) | Search for distributed traces in Tempo using TraceQL. |
| [`tempo_search_tags`](#tempo_search_tags) | 🔍 Tempo (Distributed Tracing) | List available tag names (attribute keys) in Tempo, grouped by scope. |
| [`tempo_search_tag_values`](#tempo_search_tag_values) | 🔍 Tempo (Distributed Tracing) | List the known values for a specific tag (attribute key) in Tempo. |
| [`loki_list_instances`](#loki_list_instances) | 📋 Loki (Log Management) | List LokiStack instances available in the Kubernetes cluster. |
| [`loki_label_names`](#loki_label_names) | 📋 Loki (Log Management) | List available Loki label names for a time range. |
| [`loki_label_values`](#loki_label_values) | 📋 Loki (Log Management) | List possible values for a Loki label key. |
| [`loki_query_range`](#loki_query_range) | 📋 Loki (Log Management) | Execute a Loki LogQL range query and return matching log streams and lines. |
| [`otelcol_list_components`](#otelcol_list_components) | ⚙️ OpenTelemetry Collector | List available OpenTelemetry Collector components (receivers, processors, exporters, extensions, connectors) for a given version. |
| [`otelcol_get_component_schema`](#otelcol_get_component_schema) | ⚙️ OpenTelemetry Collector | Get the JSON schema for an OpenTelemetry Collector component's configuration options. |
| [`otelcol_validate_config`](#otelcol_validate_config) | ⚙️ OpenTelemetry Collector | Validate an OpenTelemetry Collector component configuration against its JSON schema. |
| [`otelcol_get_versions`](#otelcol_get_versions) | ⚙️ OpenTelemetry Collector | List available OpenTelemetry Collector versions and identify the latest. |
| [`list_alert_rules`](#list_alert_rules) | 🛡️ Alert Management | List managed OpenShift alert rules from the monitoring-plugin management API. |
| [`preview_alert_rule`](#preview_alert_rule) | 🛡️ Alert Management | Dry-run a create or update without persisting cluster changes via POST /api/v1/alerting/rules/preview. |
| [`create_alert_rule`](#create_alert_rule) | 🛡️ Alert Management | Create an OpenShift alert rule via POST /api/v1/alerting/rules. |
| [`update_alert_rule`](#update_alert_rule) | 🛡️ Alert Management | Update alert rule labels, severity, classification, or drop/restore (alerting_rule_enabled) via PATCH /api/v1/alerting/rules. |
| [`delete_alert_rules`](#delete_alert_rules) | 🛡️ Alert Management | Delete alert rules by stable ID via DELETE /api/v1/alerting/rules. |

> [!NOTE]
> **Types in the tables** follow JSON Schema: `object` is a JSON object (string keys with JSON values); `object[]` is an array of those objects. Scalar types use their usual names (`string`, `number`, `boolean`, and so on). When a field has no explicit schema type (for example a Go `any` payload), this document shows `object` as shorthand for "structured JSON," not a guarantee that only objects are returned at runtime.

## Table of Contents

- **📈 [Prometheus / Thanos](#prometheus-thanos)** (7 tools)
  - [`list_metrics`](#list_metrics)
  - [`execute_instant_query`](#execute_instant_query)
  - [`execute_range_query`](#execute_range_query)
  - [`show_timeseries`](#show_timeseries)
  - [`get_label_names`](#get_label_names)
  - [`get_label_values`](#get_label_values)
  - [`get_series`](#get_series)
- **🔔 [Alertmanager](#alertmanager)** (5 tools)
  - [`get_alerts`](#get_alerts)
  - [`get_silences`](#get_silences)
  - [`create_silence`](#create_silence)
  - [`update_silence`](#update_silence)
  - [`delete_silence`](#delete_silence)
- **🔍 [Tempo (Distributed Tracing)](#tempo-distributed-tracing)** (5 tools)
  - [`tempo_list_instances`](#tempo_list_instances)
  - [`tempo_get_trace_by_id`](#tempo_get_trace_by_id)
  - [`tempo_search_traces`](#tempo_search_traces)
  - [`tempo_search_tags`](#tempo_search_tags)
  - [`tempo_search_tag_values`](#tempo_search_tag_values)
- **📋 [Loki (Log Management)](#loki-log-management)** (4 tools)
  - [`loki_list_instances`](#loki_list_instances)
  - [`loki_label_names`](#loki_label_names)
  - [`loki_label_values`](#loki_label_values)
  - [`loki_query_range`](#loki_query_range)
- **⚙️ [OpenTelemetry Collector](#opentelemetry-collector)** (4 tools)
  - [`otelcol_list_components`](#otelcol_list_components)
  - [`otelcol_get_component_schema`](#otelcol_get_component_schema)
  - [`otelcol_validate_config`](#otelcol_validate_config)
  - [`otelcol_get_versions`](#otelcol_get_versions)
- **🛡️ [Alert Management](#alert-management)** (5 tools)
  - [`list_alert_rules`](#list_alert_rules)
  - [`preview_alert_rule`](#preview_alert_rule)
  - [`create_alert_rule`](#create_alert_rule)
  - [`update_alert_rule`](#update_alert_rule)
  - [`delete_alert_rules`](#delete_alert_rules)

---

<a id="prometheus-thanos"></a>

## 📈 Prometheus / Thanos

### `list_metrics`

> MANDATORY FIRST STEP: List all available metric names in Prometheus.

<details>
<summary><strong>Usage Tips</strong></summary>

- YOU MUST CALL THIS TOOL BEFORE ANY OTHER QUERY TOOL
- This tool MUST be called first for EVERY observability question to: 1. Discover what metrics actually exist in this environment 2. Find the EXACT metric name to use in queries 3. Avoid querying non-existent metrics 4. The 'name_regex' parameter should always be provided, and be a best guess of what the metric would be named like. 5. Do not use a blanket regex like .* or .+ in the 'name_regex' parameter. Use specific ones like kube.*, node.*, etc.
- REGEX PATTERN GUIDANCE: - Prometheus metrics are typically prefixed (e.g., 'prometheus_tsdb_head_series', 'kube_pod_status_phase') - To match metrics CONTAINING a substring, use wildcards: '.*tsdb.*' matches 'prometheus_tsdb_head_series' - Without wildcards, the pattern matches EXACTLY: 'tsdb' only matches a metric literally named 'tsdb' (which rarely exists) - Common patterns: 'kube_pod.*' (pods), '.*memory.*' (memory-related), 'node_.*' (node metrics) - If you get empty results, try adding '.*' before/after your search term
- NEVER skip this step. NEVER guess metric names. Metric names vary between environments.
- After calling this tool: 1. Search the returned list for relevant metrics 2. Use the EXACT metric name found in subsequent queries 3. If no relevant metric exists, inform the user

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `name_regex` | `string` | Regex pattern to filter metric names. IMPORTANT: Metric names are typically prefixed (e.g., 'prometheus_tsdb_head_series'). Use wildcards to match substrings: '.*tsdb.*' matches any metric containing 'tsdb', while 'tsdb' only matches the exact string 'tsdb'. Examples: 'http_.*' (starts with http_), '.*memory.*' (contains memory), 'node_.*' (starts with node_). This parameter is required. Don't pass in blanket regex like '.*' or '.+'. |

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `metrics` | `string[]` | List of all available metric names |

</details>

---

### `execute_instant_query`

> Execute a PromQL instant query to get current/point-in-time values.

<details>
<summary><strong>Usage Tips</strong></summary>

- PREREQUISITE: You MUST call list_metrics first to verify the metric exists
- WHEN TO USE: - Current state questions: "What is the current error rate?" - Point-in-time snapshots: "How many pods are running?" - Latest values: "Which pods are in Pending state?"
- The 'query' parameter MUST use metric names that were returned by list_metrics.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `query` | `string` | PromQL query string using metric names verified via list_metrics |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `time` | `string` | Evaluation time as RFC3339 or Unix timestamp. Omit or use 'NOW' for current time. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `result` | `object[]` | The query results as an array of instant values |
| `resultType` | `string` | The type of result returned (e.g. vector, scalar, string) |
| `warnings` | `string[]` | Any warnings generated during query execution |

</details>

---

### `execute_range_query`

> Execute a PromQL range query to get time-series data over a period.

<details>
<summary><strong>Usage Tips</strong></summary>

- PREREQUISITE: You MUST call list_metrics first to verify the metric exists
- WHEN TO USE: - Trends over time: "What was CPU usage over the last hour?" - Rate calculations: "How many requests per second?" - Historical analysis: "Were there any restarts in the last 5 minutes?"
- TIME PARAMETERS: - 'duration': Look back from now (e.g., "5m", "1h", "24h") - 'step': Data point resolution (e.g., "1m" for 1-hour duration, "5m" for 24-hour duration)
- The 'query' parameter MUST use metric names that were returned by list_metrics.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `query` | `string` | PromQL query string using metric names verified via list_metrics |
| `step` | `string` | Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps. |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `duration` | `string` | Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional) |
| `end` | `string` | End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time. |
| `start` | `string` | Start time as RFC3339 or Unix timestamp (optional) |

</details>

> [!NOTE]
> Parameters with patterns must match: `^\d+[smhdwy]$`

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `result` | `object[]` | The query results as an array of time series |
| `resultType` | `string` | The type of result returned: matrix or vector or scalar |
| `summary` | `object[]` | Summary statistics for each time series (when summarize flag is enabled) |
| `warnings` | `string[]` | Any warnings generated during query execution |

</details>

---

### `show_timeseries`

> Display the results as an interactive timeseries chart.

<details>
<summary><strong>Usage Tips</strong></summary>

- This tool works like execute_range_query but renders the results as a visual chart in the UI clients. Use it when the user wants to see a graph or visualization of time-series data and to use visuals to provide the answer. Use the show_timeseries as the last tool call after all the other Prometheus tool calls where finalized.
- TIME PARAMETERS: - 'duration': Look back from now (e.g., "5m", "1h", "24h") - 'step': Data point resolution (e.g., "1m" for 1-hour duration, "5m" for 24-hour duration) - 'title': A descriptive chart title (e.g., "API Error Rate Over Last Hour") - 'description': An explanation of the chart's meaning or context (e.g., "Shows the rate of HTTP 5xx errors per second, broken down by pod")
- The 'query' parameter MUST be a range query and must use metric names that were returned by list_metrics.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `query` | `string` | PromQL query string using metric names verified via list_metrics |
| `step` | `string` | Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps. |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `description` | `string` | Explanation of the chart's meaning or context (e.g., 'Shows the rate of HTTP 5xx errors per second, broken down by pod'). Displayed below the title when provided. |
| `duration` | `string` | Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional) |
| `end` | `string` | End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time. |
| `start` | `string` | Start time as RFC3339 or Unix timestamp (optional) |
| `title` | `string` | Human-readable chart title describing what the query shows (e.g., 'API Error Rate Over Last Hour'). Displayed above the chart when provided. |

</details>

> [!NOTE]
> Parameters with patterns must match: `^\d+[smhdwy]$`

---

### `get_label_names`

> Get all label names (dimensions) available for filtering a metric.

<details>
<summary><strong>Usage Tips</strong></summary>

- WHEN TO USE (after calling list_metrics): - To discover how to filter metrics (by namespace, pod, service, etc.) - Before constructing label matchers in PromQL queries
- The 'metric' parameter should use a metric name from list_metrics output.

</details>

**Parameters:**

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `end` | `string` | End time for label discovery as RFC3339 or Unix timestamp (optional, defaults to now) |
| `metric` | `string` | Metric name (from list_metrics) to get label names for. Leave empty for all metrics. |
| `start` | `string` | Start time for label discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago) |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `labels` | `string[]` | List of label names available for the specified metric or all metrics |

</details>

---

### `get_label_values`

> Get all unique values for a specific label.

<details>
<summary><strong>Usage Tips</strong></summary>

- WHEN TO USE (after calling list_metrics and get_label_names): - To find exact label values for filtering (namespace names, pod names, etc.) - To see what values exist before constructing queries
- The 'metric' parameter should use a metric name from list_metrics output.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `label` | `string` | Label name (from get_label_names) to get values for |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `end` | `string` | End time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to now) |
| `metric` | `string` | Metric name (from list_metrics) to scope the label values to. Leave empty for all metrics. |
| `start` | `string` | Start time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago) |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `values` | `string[]` | List of unique values for the specified label |

</details>

---

### `get_series`

> Get time series matching selectors and preview cardinality.

<details>
<summary><strong>Usage Tips</strong></summary>

- WHEN TO USE (optional, after calling list_metrics): - To verify label filters match expected series before querying - To check cardinality and avoid slow queries
- CARDINALITY GUIDANCE: - <100 series: Safe - 100-1000: Usually fine - >1000: Add more label filters
- The selector should use metric names from list_metrics output.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `matches` | `string` | PromQL series selector using metric names from list_metrics |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `end` | `string` | End time for series discovery as RFC3339 or Unix timestamp (optional, defaults to now) |
| `start` | `string` | Start time for series discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago) |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `cardinality` | `integer` | Total number of series matching the selector |
| `series` | `object[]` | List of time series matching the selector, each series is a map of label names to values |

</details>

---

<a id="alertmanager"></a>

## 🔔 Alertmanager

### `get_alerts`

> Get alerts from Alertmanager.

<details>
<summary><strong>Usage Tips</strong></summary>

- WHEN TO USE: - START HERE when investigating issues: if the user asks about things breaking, errors, failures, outages, services being down, or anything going wrong in the cluster - When the user mentions a specific alert name - use this tool to get the alert's full labels (namespace, pod, service, etc.) which are essential for further investigation with other tools - To see currently firing alerts in the cluster - To check which alerts are active, silenced, or inhibited - To understand what's happening before diving into metrics or logs
- INVESTIGATION TIP: Alert labels often contain the exact identifiers (pod names, namespaces, job names) needed for targeted queries with prometheus tools.
- FILTERING: - Use 'active' to filter for only active alerts (not resolved) - Use 'silenced' to filter for silenced alerts - Use 'inhibited' to filter for inhibited alerts - Use 'filter' to apply label matchers (e.g., "alertname=HighCPU") - Use 'receiver' to filter alerts by receiver name
- All filter parameters are optional. Without filters, all alerts are returned.

</details>

**Parameters:**

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `active` | `boolean` | Filter for active alerts only (true/false, optional) |
| `filter` | `string` | Label matchers to filter alerts (e.g., 'alertname=HighCPU', optional) |
| `inhibited` | `boolean` | Filter for inhibited alerts only (true/false, optional) |
| `receiver` | `string` | Receiver name to filter alerts (optional) |
| `silenced` | `boolean` | Filter for silenced alerts only (true/false, optional) |
| `unprocessed` | `boolean` | Filter for unprocessed alerts only (true/false, optional) |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `alerts` | `object[]` | List of alerts from Alertmanager |

</details>

---

### `get_silences`

> Get silences from Alertmanager.

<details>
<summary><strong>Usage Tips</strong></summary>

- WHEN TO USE: - To see which alerts are currently silenced - To check active, pending, or expired silences - To investigate why certain alerts are not firing notifications
- FILTERING: - Use 'filter' to apply label matchers to find specific silences
- Silences are used to temporarily mute alerts based on label matchers. This tool helps you understand what is currently silenced in your environment. To create, change, or expire a silence, use create_silence, update_silence, or delete_silence.

</details>

**Parameters:**

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `filter` | `string` | Label matchers to filter silences (e.g., 'alertname=HighCPU', optional) |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `silences` | `object[]` | List of silences from Alertmanager |

</details>

---

### `create_silence`

> Create an Alertmanager silence that mutes matching alerts.

<details>
<summary><strong>Usage Tips</strong></summary>

- WHEN TO USE: - The user asks to silence or mute an alert or set of alerts - Operator-managed or GitOps-managed rules cannot be edited; mute notifications with a silence instead
- BEFORE CALLING: - Call get_alerts (and get_silences) so matchers are specific. Prefer alertname plus namespace and other labels from the firing alert. - Tell the user which label matchers will apply and for how long. Wait for explicit agreement. - alertname alone silences every instance of that alert name.
- PARAMETERS: - comment is required. - Provide labels (equality map) and/or matchers. At least one matcher is required after combining them. - duration defaults to 2h when endsAt is omitted. Do not set both duration and endsAt. - createdBy defaults to obs-mcp if omitted.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `comment` | `string` | Reason for the silence. Required on create; optional on update (keeps the existing comment). |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `createdBy` | `string` | Who created the silence. Defaults to obs-mcp if omitted. |
| `duration` | `string` | How long the silence lasts from startsAt (for example 2h). Default 2h when endsAt is omitted. Do not set together with endsAt. |
| `endsAt` | `string` | Silence end time. Do not set together with duration. |
| `labels` | `object` | Equality matchers as a label map (for example alertname=Watchdog, namespace=app). Combined with matchers. |
| `matchers` | `object[]` | Alertmanager matchers. Use with or instead of labels. alertname alone mutes every instance of that alert. |
| `startsAt` | `string` | Silence start time (RFC3339, Unix, NOW, or NOW±duration). Defaults to now. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `silence_id` | `string` | Alertmanager silence ID |

</details>

---

### `update_silence`

> Update an existing Alertmanager silence (POST /api/v2/silences with silence_id).

<details>
<summary><strong>Usage Tips</strong></summary>

- WHEN TO USE: - Extend, shorten, or retarget a silence returned by get_silences - silence_id is required (UUID)
- Omitted comment, createdBy, matchers/labels, and times keep the existing silence values. duration replaces endsAt relative to startsAt. Explain the change and wait for the user to agree before calling.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `silence_id` | `string` | Alertmanager silence UUID from get_silences or create_silence. |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `comment` | `string` | Reason for the silence. Required on create; optional on update (keeps the existing comment). |
| `createdBy` | `string` | Who created the silence. Defaults to obs-mcp if omitted. |
| `duration` | `string` | How long the silence lasts from startsAt (for example 2h). Default 2h when endsAt is omitted. Do not set together with endsAt. |
| `endsAt` | `string` | Silence end time. Do not set together with duration. |
| `labels` | `object` | Equality matchers as a label map (for example alertname=Watchdog, namespace=app). Combined with matchers. |
| `matchers` | `object[]` | Alertmanager matchers. Use with or instead of labels. alertname alone mutes every instance of that alert. |
| `startsAt` | `string` | Silence start time (RFC3339, Unix, NOW, or NOW±duration). Defaults to now. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `silence_id` | `string` | Alertmanager silence ID |

</details>

---

### `delete_silence`

> Expire (delete) an Alertmanager silence by UUID.

<details>
<summary><strong>Usage Tips</strong></summary>

- WHEN TO USE: - The user asks to un-silence, expire, or remove a silence - silence_id is required (from get_silences)
- Explain which silence will be removed and wait for the user to agree. This calls DELETE /api/v2/silence/{id}.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `silence_id` | `string` | Alertmanager silence UUID to expire (delete). |

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `silence_id` | `string` | Alertmanager silence ID |

</details>

---

<a id="tempo-distributed-tracing"></a>

## 🔍 Tempo (Distributed Tracing)

### `tempo_list_instances`

> List all Tempo instances available in the Kubernetes cluster.
> Call this tool first to discover available Tempo instances before using other Tempo tools,
> as the returned namespace, name, and tenant values are required parameters for all other Tempo tools.
> Always print the output of this tool in a table.

_No parameters._

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `instances` | `object[]` | List of available Tempo instances |

</details>

---

### `tempo_get_trace_by_id`

> Retrieve a single distributed trace by its trace ID from Tempo.
> Returns the full trace with all its spans, including service names, operation names, durations, and attributes.
> Use this tool when you already have a specific trace ID, e.g. from search results or logs.

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `tempoName` | `string` | The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names. |
| `tempoNamespace` | `string` | The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces. |
| `traceid` | `string` | The trace ID to retrieve, e.g. "26dad4a0e2b0dd9a440dd5ff203a24a4". |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `end` | `string` | Optional end of the time range in RFC 3339 format, e.g. "2025-01-02T00:00:00Z".<br>Narrows the time range to improve query performance. |
| `start` | `string` | Optional start of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".<br>Narrows the time range to improve query performance. |
| `tenant` | `string` | The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `trace` | `object` | The trace data with services, scopes and spans |

</details>

---

### `tempo_search_traces`

> Search for distributed traces in Tempo using TraceQL.
> Use this tool to find traces matching specific criteria such as service name, HTTP status code, duration, or other span or resource attributes.

<details>
<summary><strong>Usage Tips</strong></summary>

- IMPORTANT — "slow" or "long" trace requests: Do NOT guess a duration threshold. First call this tool WITHOUT a duration filter to establish a latency baseline, then use that baseline to set a sensible threshold. Both steps are required — do NOT skip the second search with the duration filter. Skip this two-step process only when the user provides an explicit duration (e.g. "find traces slower than 2s").

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `query` | `string` | A TraceQL query expression. Format:<br>query: "{ <filters joined by &&> }"<br><br>Filters:<br>- service name:     resource.service.name="<value>" (string, use quotes)<br>- HTTP status code: span.http.response.status_code=<code> (number, no quotes)<br>- duration:         duration><value like 100ms, 2s, 5m> (no quotes)<br>- error status:     status=error (keyword, NO quotes — do NOT write status="error")<br><br>IMPORTANT: status values (error, ok, unset) are keywords, NOT strings. Write status=error, NEVER status="error".<br><br>Operators: =, !=, >, <, >=, <=<br><br>Common attributes:<br>- resource.service.name (service name)<br>- resource.k8s.namespace.name (Kubernetes namespace)<br>- resource.k8s.deployment.name (Kubernetes deployment)<br>- resource.k8s.statefulset.name (Kubernetes statefulset)<br>- resource.k8s.daemonset.name (Kubernetes daemonset)<br>- resource.k8s.replicaset.name (Kubernetes replicaset)<br>- resource.k8s.pod.name (Kubernetes pod)<br>- resource.k8s.container.name (Kubernetes container)<br>- resource.k8s.job.name (Kubernetes job)<br>- resource.k8s.cronjob.name (Kubernetes cronjob)<br>- resource.k8s.node.name (Kubernetes node)<br>- resource.k8s.cluster.name (Kubernetes cluster)<br>- span.http.response.status_code (HTTP response code)<br>- span.http.request.method (HTTP method like GET, POST)<br>- span.url.full (request URL)<br>- name (span name / operation name, e.g. "GET /api/users")<br>- duration (trace duration, e.g. 100ms, 2s)<br>- status (trace status: ok, error, unset)<br><br>Note: older instrumentation may use legacy HTTP attribute names (e.g. span.http.status_code instead of span.http.response.status_code).<br>If a query returns no results, try tempo_search_tags to check which attributes exist.<br><br>IMPORTANT:<br>- Always wrap filters in curly braces { }.<br>- Do NOT use SQL, PromQL, or Lucene syntax.<br>- Do NOT omit the "resource." or "span." prefix from attribute names<br>- When the user refers to a Kubernetes resource type (deployment, pod, namespace, etc.), use the matching resource.k8s.* attribute, NOT resource.service.name.<br><br>Examples:<br>- { resource.service.name="frontend" }<br>- { resource.k8s.deployment.name="checkout" && span.http.response.status_code>=500 }<br>- { status=error && duration>2s }<br><br>If unsure which attributes to filter on, use tempo_search_tags to discover available attributes before building a query. |
| `tempoName` | `string` | The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names. |
| `tempoNamespace` | `string` | The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces. |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `end` | `string` | End of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".<br>Use "NOW" for current time.<br>Both start and end should be provided to search the full time range; if omitted, only a small window of recent data is searched. |
| `limit` | `integer` | Maximum number of traces to return. Defaults to the server-side limit if not specified. |
| `spss` | `integer` | Maximum number of matching spans to return per trace. |
| `start` | `string` | Start of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".<br>Use "NOW" for current time.<br>Both start and end should be provided to search the full time range; if omitted, only a small window of recent data is searched. |
| `tenant` | `string` | The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `metrics` | `object` | Query performance metrics |
| `traces` | `object[]` | List of matching traces with metadata |

</details>

---

### `tempo_search_tags`

> List available tag names (attribute keys) in Tempo, grouped by scope.
> Use this tool to discover which attributes are available for building TraceQL queries with tempo_search_traces.
> For example, this tool may reveal tag names like "service.name" (in the "resource" scope) or "http.response.status_code" (in the "span" scope).
> To use these in TraceQL queries, prefix them with their scope, e.g. "resource.service.name" or "span.http.response.status_code".

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `tempoName` | `string` | The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names. |
| `tempoNamespace` | `string` | The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces. |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `end` | `string` | Optional end of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing tags. |
| `limit` | `integer` | Maximum number of tag names to return per scope. |
| `maxStaleValues` | `integer` | Maximum number of consecutive blocks without new tag names before the search stops early. Higher values are more thorough but slower. |
| `query` | `string` | Optional TraceQL query to filter which traces are considered when listing tags,<br>e.g. '{ resource.service.name="payment-service" }' to only show tags present in traces from the 'payment-service' service. |
| `scope` | `string` | Filter tags to a specific scope. One of:<br>"resource" (service-level attributes like service.name),<br>"span" (individual span attributes like http.response.status_code),<br>"intrinsic" (built-in fields like duration, status, name).<br>If omitted, tags from all scopes are returned. |
| `start` | `string` | Optional start of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing tags. |
| `tenant` | `string` | The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `scopes` | `object[]` | List of tag scopes with their tag names |

</details>

---

### `tempo_search_tag_values`

> List the known values for a specific tag (attribute key) in Tempo.
> Use this tool to discover what values exist for a given tag, e.g. to find all service names (values of "resource.service.name") or all HTTP methods (values of "span.http.request.method").
> This is useful for building accurate TraceQL queries with tempo_search_traces.

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `tag` | `string` | The fully qualified tag name to get values for, including its scope prefix, e.g. "resource.service.name" or "span.http.response.status_code".<br>Use tempo_search_tags to discover available tag names. |
| `tempoName` | `string` | The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names. |
| `tempoNamespace` | `string` | The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces. |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `end` | `string` | Optional end of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing values. |
| `limit` | `integer` | Maximum number of tag values to return. |
| `maxStaleValues` | `integer` | Maximum number of consecutive blocks without new values before the search stops early. Higher values are more thorough but slower. |
| `query` | `string` | Optional TraceQL query to filter which traces are considered when listing values,<br>e.g. '{ resource.service.name="payment-service" }' to only show tag values from the 'payment-service' service. |
| `start` | `string` | Optional start of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing values. |
| `tenant` | `string` | The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `tagValues` | `object` | Known values for the specified tag, keyed by type |

</details>

---

<a id="loki-log-management"></a>

## 📋 Loki (Log Management)

### `loki_list_instances`

> List LokiStack instances available in the Kubernetes cluster.
> Call this first when using Loki Operator managed stacks so you can pass lokiNamespace and lokiName to other Loki tools.

_No parameters._

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `instances` | `object[]` |  |

</details>

---

### `loki_label_names`

> List available Loki label names for a time range. Use this before writing LogQL queries.

**Parameters:**

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `end` | `string` | End time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional). |
| `lokiName` | `string` | Name of the LokiStack. Use loki_list_instances to discover valid values. |
| `lokiNamespace` | `string` | Kubernetes namespace of the LokiStack. Use loki_list_instances to discover valid values. |
| `start` | `string` | Start time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional). |
| `tenant` | `string` | Loki tenant ID (X-Scope-OrgID). For LokiStack gateway modes (e.g. openshift-network) this selects the `/api/logs/v1/<tenant>` path; use `network` for openshift-network. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `labels` | `string[]` |  |

</details>

---

### `loki_label_values`

> List possible values for a Loki label key. Use this to build precise label matchers in LogQL.

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `label` | `string` | Label key to inspect (for example namespace, pod, container). |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `end` | `string` | End time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional). |
| `lokiName` | `string` | Name of the LokiStack. Use loki_list_instances to discover valid values. |
| `lokiNamespace` | `string` | Kubernetes namespace of the LokiStack. Use loki_list_instances to discover valid values. |
| `start` | `string` | Start time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional). |
| `tenant` | `string` | Loki tenant ID (X-Scope-OrgID). For LokiStack gateway modes (e.g. openshift-network) this selects the `/api/logs/v1/<tenant>` path; use `network` for openshift-network. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `values` | `string[]` |  |

</details>

---

### `loki_query_range`

> Execute a Loki LogQL range query and return matching log streams and lines.

<details>
<summary><strong>Usage Tips</strong></summary>

- Use precise label matchers and a short time window first.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `query` | `string` | LogQL query string. |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `direction` | `string` | Search direction: backward (default) or forward. |
| `duration` | `string` | Lookback duration from now when start/end are omitted (for example 5m, 1h). Defaults to 15m. |
| `end` | `string` | End time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional). |
| `limit` | `integer` | Maximum number of log lines to return. Defaults to 100, max 1000. |
| `lokiName` | `string` | Name of the LokiStack. Use loki_list_instances to discover valid values. |
| `lokiNamespace` | `string` | Kubernetes namespace of the LokiStack. Use loki_list_instances to discover valid values. |
| `start` | `string` | Start time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional). |
| `tenant` | `string` | Loki tenant ID (X-Scope-OrgID). For LokiStack gateway modes (e.g. openshift-network) this selects the `/api/logs/v1/<tenant>` path; use `network` for openshift-network. |

</details>

> [!NOTE]
> Parameters with patterns must match: `^\d+[smhdwy]$`

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `resultType` | `string` |  |
| `streams` | `object[]` |  |

</details>

---

<a id="opentelemetry-collector"></a>

## ⚙️ OpenTelemetry Collector

### `otelcol_list_components`

> List available OpenTelemetry Collector components (receivers, processors, exporters, extensions, connectors) for a given version.

**Parameters:**

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `version` | `string` | Collector version (e.g., 'v0.100.0'). Defaults to latest available. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `components` | `object` | Map of component type to component names |
| `connectors` | `string[]` | List of available connector component names |
| `exporters` | `string[]` | List of available exporter component names |
| `extensions` | `string[]` | List of available extension component names |
| `processors` | `string[]` | List of available processor component names |
| `receivers` | `string[]` | List of available receiver component names |
| `version` | `string` | The OpenTelemetry Collector version |

</details>

---

### `otelcol_get_component_schema`

> Get the JSON schema for an OpenTelemetry Collector component's configuration options.

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `component_name` | `string` | Component name from otelcol_list_components (e.g., 'otlp', 'batch', 'debug') |
| `component_type` | `string` | Component type: receiver, processor, exporter, extension, connector |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `version` | `string` | Collector version (e.g., 'v0.100.0'). Defaults to latest available. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | `string` | The component name |
| `schema` | `object` | The JSON schema for the component configuration |
| `type` | `string` | The component type (receiver, processor, exporter, extension, connector) |
| `version` | `string` | The OpenTelemetry Collector version |

</details>

---

### `otelcol_validate_config`

> Validate an OpenTelemetry Collector component configuration against its JSON schema.

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `component_name` | `string` | Component name from otelcol_list_components (e.g., 'otlp', 'batch', 'debug') |
| `component_type` | `string` | Component type: receiver, processor, exporter, extension, connector |
| `config` | `string` | Configuration to validate as YAML or JSON string |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `format` | `string` | Config format: 'yaml' (default) or 'json' |
| `version` | `string` | Collector version (e.g., 'v0.100.0'). Defaults to latest available. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `errors` | `object[]` | List of validation errors if invalid |
| `valid` | `boolean` | Whether the configuration is valid |
| `version` | `string` | The OpenTelemetry Collector version used for validation |

</details>

---

### `otelcol_get_versions`

> List available OpenTelemetry Collector versions and identify the latest.

_No parameters._

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `latest_version` | `string` | The latest available version |
| `versions` | `string[]` | List of available OpenTelemetry Collector versions |

</details>

---

<a id="alert-management"></a>

## 🛡️ Alert Management

### `list_alert_rules`

> List managed OpenShift alert rules from the monitoring-plugin management API.

<details>
<summary><strong>Usage Tips</strong></summary>

- Returns each rule's stable ID, PromQL expression, severity, namespace, source (platform or user), and management_status (user-created, gitops, or operator). This is rule definitions, not firing instances (use get_alerts for those). Use this before create (to detect the same expr) and before update/delete (to resolve names to ids). Alert name is not unique: if several rows share a name, present them to the user and wait; do not pick one or change all of them. If id is empty, do not invent an id. Surface warnings from the response. GitOps-managed and operator-managed rules cannot be updated or deleted through this API; explain that and point GitOps users at editing the rule in Git.
- Optional filters: namespace, severity, state (pending, firing, silenced), source, cluster, cluster_labels, matchers, and extra label equality filters.

</details>

**Parameters:**

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `cluster` | `string` | Optional cluster name filter (query parameter cluster). |
| `cluster_labels` | `object` | Optional cluster label filters, forwarded as cluster_labels=<key>=<value>. |
| `labels` | `object` | Additional label equality filters forwarded as query parameters. |
| `matchers` | `string[]` | Prometheus-style match[] selectors (for example severity="critical"). |
| `namespace` | `string` | Filter by namespace label (Thanos tenancy for user-workload rules). |
| `severity` | `string` | Alert severity: critical, warning, info, or none. |
| `source` | `string` | Filter by openshift_io_alert_source: platform or user. |
| `state` | `string` | Filter by alert state: pending, firing, or silenced. Omit for all states. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `rules` | `object[]` | Managed alert rules matching the filters |
| `warnings` | `string[]` | Non-fatal backend warnings from the management API |

</details>

---

### `preview_alert_rule`

> Dry-run a create or update without persisting cluster changes via POST /api/v1/alerting/rules/preview.

<details>
<summary><strong>Usage Tips</strong></summary>

- Create preview: set alert and expr. Include prometheus_rule_name and namespace for a user-defined rule; omit prometheus_rule_name for a platform rule. Do not invent a PrometheusRule name. Update preview: set rule_id plus labels, severity, classification, or alerting_rule_enabled. Update cannot preview expr, alert name, for, or annotation changes. Do not pass cluster or cluster_labels on this operation.
- Required before create_alert_rule or update_alert_rule. After preview, explain the plan (writable, managedBy, resources, desiredRule) and wait for the user to agree. If writable is false, do not call create_alert_rule or update_alert_rule. If managedBy is gitops, tell the user to make the change in Git (PrometheusRule or AlertingRule source of truth). If managedBy is operator, explain that the operator owns the rule and this API will not persist the spec change.

</details>

**Parameters:**

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `alert` | `string` | Alert name for create preview. |
| `alerting_rule_enabled` | `boolean` | When false, drop a platform alert rule via AlertRelabelConfig. When true, restore a previously dropped rule. Cannot be combined with labels or classification. |
| `annotations` | `object` | Annotations for create preview. |
| `classification_component` | `string` | Set openshift_io_alert_rule_component. Empty or null clears the override. |
| `classification_component_from` | `string` | Set openshift_io_alert_rule_component_from. Empty or null clears the override. |
| `classification_layer` | `string` | Set openshift_io_alert_rule_layer. Empty or null clears the override. |
| `classification_layer_from` | `string` | Set openshift_io_alert_rule_layer_from. Empty or null clears the override. |
| `cluster` | `string` | Not supported on this operation; omit this field. |
| `cluster_labels` | `object` | Not supported on this operation; omit this field. |
| `expr` | `string` | PromQL expression for create preview. |
| `for` | `string` | Pending duration for create preview (for example 5m). |
| `group_name` | `string` | Optional rule group name for user-defined create preview. |
| `labels` | `object` | Label key/value pairs to set. |
| `labels_to_remove` | `string[]` | Label keys to remove (sent as null values on the management API). |
| `namespace` | `string` | PrometheusRule namespace for user-defined create preview. |
| `prometheus_rule_name` | `string` | PrometheusRule name for user-defined create preview. Omit for platform rules. |
| `rule_id` | `string` | Single stable alert rule ID. |
| `severity` | `string` | Alert severity: critical, warning, info, or none. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `desiredRule` | `object` | Desired alerting rule after the planned change |
| `managedBy` | `string` | gitops or operator when the target is not writable |
| `resources` | `object[]` | Kubernetes resources that would be created or modified |
| `writable` | `boolean` | Whether the management API can persist this change |

</details>

---

### `create_alert_rule`

> Create an OpenShift alert rule via POST /api/v1/alerting/rules.

<details>
<summary><strong>Usage Tips</strong></summary>

- Set prometheus_rule_name and namespace for a user-defined rule in that PrometheusRule. Omit prometheus_rule_name to create a platform alerting rule. Do not invent a PrometheusRule name; if the target is unclear, ask before calling. Do not put secrets in labels or annotations. Do not pass cluster or cluster_labels on this operation.
- Before calling this tool: 1. list_alert_rules and check whether any rule already has the same PromQL expression. If one does, ask the user whether to update that rule instead of creating a duplicate. 2. If observability/metrics is enabled, verify expr metric names with list_metrics. 3. preview_alert_rule for the create payload. 4. Explain the planned create (user-defined vs platform) and wait for the user to agree.
- Do not retry a 409/405 GitOps or operator-managed conflict. If the target PrometheusRule is GitOps-managed, tell the user to add the alert in Git instead. Do not retry 400, 404, or 413.

</details>

**Parameters:**

**Required:**

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `alert` | `string` | Alert name (PrometheusRule alert field). |
| `expr` | `string` | PromQL expression to evaluate. |

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `annotations` | `object` | Annotations to attach to alerts produced by the rule. |
| `cluster` | `string` | Not supported on this operation; omit this field. |
| `cluster_labels` | `object` | Not supported on this operation; omit this field. |
| `for` | `string` | Duration the condition must be true before firing (for example 5m). |
| `group_name` | `string` | Optional rule group name within the PrometheusRule. |
| `labels` | `object` | Labels to attach to the rule. severity is merged from the severity parameter when set. |
| `namespace` | `string` | PrometheusRule namespace for user-defined rules. Required with prometheus_rule_name. Otherwise stored as a namespace label on a platform rule. |
| `prometheus_rule_name` | `string` | PrometheusRule resource name for a user-defined rule. Omit to create a platform rule. |
| `severity` | `string` | Alert severity: critical, warning, info, or none. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | `string` | Computed stable ID for the created alert rule |

</details>

---

### `update_alert_rule`

> Update alert rule labels, severity, classification, or drop/restore (alerting_rule_enabled) via PATCH /api/v1/alerting/rules.

<details>
<summary><strong>Usage Tips</strong></summary>

- Cannot change expr, alert name, for, or annotations. If the user wants a new expression, do not call this tool. Provide rule_id or rule_ids (1-100) of rules the user has confirmed. At least one mutation field is required. alerting_rule_enabled cannot be combined with labels, severity, or classification in the same request. Drop/restore (alerting_rule_enabled) is platform-only. Do not set it on source=user rules. Do not use this tool to silence an alert; use create_silence. Do not pass cluster or cluster_labels on this operation.
- Never resolve an alert name to every matching id. If list_alert_rules returns more than one rule with that name, ask which id to use. Do not update all matches unless the user explicitly asks for that and confirms. Call preview_alert_rule, explain the change, and wait for agreement before this call. Read each per-rule statusCode. After a label update, use the returned id; it may differ from the id you sent.
- management_status gitops: do not call this tool; tell the user to edit the PrometheusRule or AlertingRule in Git. management_status operator or preview writable=false: do not retry; explain operator ownership. Silences may mute notifications. Do not retry 400, 404, or 413.

</details>

**Parameters:**

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `alerting_rule_enabled` | `boolean` | When false, drop a platform alert rule via AlertRelabelConfig. When true, restore a previously dropped rule. Cannot be combined with labels or classification. |
| `classification_component` | `string` | Set openshift_io_alert_rule_component. Empty or null clears the override. |
| `classification_component_from` | `string` | Set openshift_io_alert_rule_component_from. Empty or null clears the override. |
| `classification_layer` | `string` | Set openshift_io_alert_rule_layer. Empty or null clears the override. |
| `classification_layer_from` | `string` | Set openshift_io_alert_rule_layer_from. Empty or null clears the override. |
| `cluster` | `string` | Not supported on this operation; omit this field. |
| `cluster_labels` | `object` | Not supported on this operation; omit this field. |
| `labels` | `object` | Label key/value pairs to set. |
| `labels_to_remove` | `string[]` | Label keys to remove (sent as null values on the management API). |
| `rule_id` | `string` | Single stable alert rule ID. |
| `rule_ids` | `string[]` | Stable alert rule IDs (at most 100 combined with rule_id). |
| `severity` | `string` | Alert severity: critical, warning, info, or none. |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `rules` | `object[]` | Per-rule update results |

</details>

---

### `delete_alert_rules`

> Delete alert rules by stable ID via DELETE /api/v1/alerting/rules.

<details>
<summary><strong>Usage Tips</strong></summary>

- Use this to remove a user-created rule, not to mute notifications (create_silence) and not to drop a platform rule (alerting_rule_enabled=false). Provide rule_id or rule_ids (1-100) the user has confirmed. The response always includes per-rule statusCode and optional message so partial success is visible; report each failure. GitOps-managed and operator-managed rules cannot be deleted. Do not pass cluster or cluster_labels on this operation.
- If the user asked by name and several rules share that name, ask which id to delete. Do not delete all matches unless the user explicitly asks for that and confirms. If management_status is gitops, tell the user to remove the rule in Git instead of calling this tool. Do not retry 400, 404, or 413.

</details>

**Parameters:**

<details>
<summary><strong>Optional Parameters</strong></summary>

| Parameter | Type | Description |
| :--- | :--- | :--- |
| `cluster` | `string` | Not supported on this operation; omit this field. |
| `cluster_labels` | `object` | Not supported on this operation; omit this field. |
| `rule_id` | `string` | Single stable alert rule ID to delete. |
| `rule_ids` | `string[]` | Stable alert rule IDs to delete (at most 100 combined with rule_id). |

</details>

<details>
<summary><strong>Output Schema</strong></summary>

| Field | Type | Description |
| :--- | :--- | :--- |
| `rules` | `object[]` | Per-rule deletion results |

</details>

