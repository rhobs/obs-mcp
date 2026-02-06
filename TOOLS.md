<!-- This file is auto-generated. Do not edit manually. -->
<!-- Run 'make generate-tools-doc' to regenerate. -->

# Available Tools

This MCP server exposes the following tools for interacting with Prometheus/Thanos:

## `list_metrics`

> MANDATORY FIRST STEP: List all available metric names in Prometheus.

**Usage Tips:**

- YOU MUST CALL THIS TOOL BEFORE ANY OTHER QUERY TOOL
- This tool MUST be called first for EVERY observability question to: 1. Discover what metrics actually exist in this environment 2. Find the EXACT metric name to use in queries 3. Avoid querying non-existent metrics
- NEVER skip this step. NEVER guess metric names. Metric names vary between environments.
- After calling this tool: 1. Search the returned list for relevant metrics 2. Use the EXACT metric name found in subsequent queries 3. If no relevant metric exists, inform the user

|                |      |
| :------------- | :--- |
| **Parameters** | None |

**Output Schema:**

| Field     | Type       | Description                        |
| :-------- | :--------- | :--------------------------------- |
| `metrics` | `string[]` | List of all available metric names |

---

## `execute_instant_query`

> Execute a PromQL instant query to get current/point-in-time values.

**Usage Tips:**

- PREREQUISITE: You MUST call list_metrics first to verify the metric exists
- WHEN TO USE: - Current state questions: "What is the current error rate?" - Point-in-time snapshots: "How many pods are running?" - Latest values: "Which pods are in Pending state?"
- The 'query' parameter MUST use metric names that were returned by list_metrics.

**Parameters:**

| Parameter | Type     | Required | Description                                                                       |
| :-------- | :------- | :------: | :-------------------------------------------------------------------------------- |
| `query`   | `string` | ✅        | PromQL query string using metric names verified via list_metrics                  |
| `time`    | `string` |          | Evaluation time as RFC3339 or Unix timestamp. Omit or use 'NOW' for current time. |

**Output Schema:**

| Field        | Type       | Description                                     |
| :----------- | :--------- | :---------------------------------------------- |
| `result`     | `object[]` | The query results as an array of instant values |
| `resultType` | `string`   | The type of result returned (e.g. vector        |
| `warnings`   | `string[]` | Any warnings generated during query execution   |

---

## `execute_range_query`

> Execute a PromQL range query to get time-series data over a period.

**Usage Tips:**

- PREREQUISITE: You MUST call list_metrics first to verify the metric exists
- WHEN TO USE: - Trends over time: "What was CPU usage over the last hour?" - Rate calculations: "How many requests per second?" - Historical analysis: "Were there any restarts in the last 5 minutes?"
- TIME PARAMETERS: - 'duration': Look back from now (e.g., "5m", "1h", "24h") - 'step': Data point resolution (e.g., "1m" for 1-hour duration, "5m" for 24-hour duration)
- The 'query' parameter MUST use metric names that were returned by list_metrics.

**Parameters:**

| Parameter  | Type     | Required | Description                                                                                                          |
| :--------- | :------- | :------: | :------------------------------------------------------------------------------------------------------------------- |
| `query`    | `string` | ✅        | PromQL query string using metric names verified via list_metrics                                                     |
| `step`     | `string` | ✅        | Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps. |
| `duration` | `string` |          | Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional)                                            |
| `end`      | `string` |          | End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time.                                        |
| `start`    | `string` |          | Start time as RFC3339 or Unix timestamp (optional)                                                                   |

> [!NOTE]
> Parameters with patterns must match: `^\d+[smhdwy]$`

**Output Schema:**

| Field        | Type       | Description                                             |
| :----------- | :--------- | :------------------------------------------------------ |
| `result`     | `object[]` | The query results as an array of time series            |
| `resultType` | `string`   | The type of result returned: matrix or vector or scalar |
| `warnings`   | `string[]` | Any warnings generated during query execution           |

---

## `get_label_names`

> Get all label names (dimensions) available for filtering a metric.

**Usage Tips:**

- WHEN TO USE (after calling list_metrics): - To discover how to filter metrics (by namespace, pod, service, etc.) - Before constructing label matchers in PromQL queries
- The 'metric' parameter should use a metric name from list_metrics output.

**Parameters:**

| Parameter | Type     | Required | Description                                                                                    |
| :-------- | :------- | :------: | :--------------------------------------------------------------------------------------------- |
| `end`     | `string` |          | End time for label discovery as RFC3339 or Unix timestamp (optional, defaults to now)          |
| `metric`  | `string` |          | Metric name (from list_metrics) to get label names for. Leave empty for all metrics.           |
| `start`   | `string` |          | Start time for label discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago) |

**Output Schema:**

| Field    | Type       | Description                                                           |
| :------- | :--------- | :-------------------------------------------------------------------- |
| `labels` | `string[]` | List of label names available for the specified metric or all metrics |

---

## `get_label_values`

> Get all unique values for a specific label.

**Usage Tips:**

- WHEN TO USE (after calling list_metrics and get_label_names): - To find exact label values for filtering (namespace names, pod names, etc.) - To see what values exist before constructing queries
- The 'metric' parameter should use a metric name from list_metrics output.

**Parameters:**

| Parameter | Type     | Required | Description                                                                                          |
| :-------- | :------- | :------: | :--------------------------------------------------------------------------------------------------- |
| `label`   | `string` | ✅        | Label name (from get_label_names) to get values for                                                  |
| `end`     | `string` |          | End time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to now)          |
| `metric`  | `string` |          | Metric name (from list_metrics) to scope the label values to. Leave empty for all metrics.           |
| `start`   | `string` |          | Start time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago) |

**Output Schema:**

| Field    | Type       | Description                                   |
| :------- | :--------- | :-------------------------------------------- |
| `values` | `string[]` | List of unique values for the specified label |

---

## `get_series`

> Get time series matching selectors and preview cardinality.

**Usage Tips:**

- WHEN TO USE (optional, after calling list_metrics): - To verify label filters match expected series before querying - To check cardinality and avoid slow queries
- CARDINALITY GUIDANCE: - <100 series: Safe - 100-1000: Usually fine - >1000: Add more label filters
- The selector should use metric names from list_metrics output.

**Parameters:**

| Parameter | Type     | Required | Description                                                                                     |
| :-------- | :------- | :------: | :---------------------------------------------------------------------------------------------- |
| `matches` | `string` | ✅        | PromQL series selector using metric names from list_metrics                                     |
| `end`     | `string` |          | End time for series discovery as RFC3339 or Unix timestamp (optional, defaults to now)          |
| `start`   | `string` |          | Start time for series discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago) |

**Output Schema:**

| Field         | Type       | Description                                  |
| :------------ | :--------- | :------------------------------------------- |
| `cardinality` | `integer`  | Total number of series matching the selector |
| `series`      | `object[]` | List of time series matching the selector    |

---

## `get_alerts`

> Get alerts from Alertmanager.

**Usage Tips:**

- WHEN TO USE: - START HERE when investigating issues: if the user asks about things breaking, errors, failures, outages, services being down, or anything going wrong in the cluster - When the user mentions a specific alert name - use this tool to get the alert's full labels (namespace, pod, service, etc.) which are essential for further investigation with other tools - To see currently firing alerts in the cluster - To check which alerts are active, silenced, or inhibited - To understand what's happening before diving into metrics or logs
- INVESTIGATION TIP: Alert labels often contain the exact identifiers (pod names, namespaces, job names) needed for targeted queries with prometheus tools.
- FILTERING: - Use 'active' to filter for only active alerts (not resolved) - Use 'silenced' to filter for silenced alerts - Use 'inhibited' to filter for inhibited alerts - Use 'filter' to apply label matchers (e.g., "alertname=HighCPU") - Use 'receiver' to filter alerts by receiver name
- All filter parameters are optional. Without filters, all alerts are returned.

**Parameters:**

| Parameter     | Type      | Required | Description                                                           |
| :------------ | :-------- | :------: | :-------------------------------------------------------------------- |
| `active`      | `boolean` |          | Filter for active alerts only (true/false, optional)                  |
| `filter`      | `string`  |          | Label matchers to filter alerts (e.g., 'alertname=HighCPU', optional) |
| `inhibited`   | `boolean` |          | Filter for inhibited alerts only (true/false, optional)               |
| `receiver`    | `string`  |          | Receiver name to filter alerts (optional)                             |
| `silenced`    | `boolean` |          | Filter for silenced alerts only (true/false, optional)                |
| `unprocessed` | `boolean` |          | Filter for unprocessed alerts only (true/false, optional)             |

**Output Schema:**

| Field    | Type       | Description                      |
| :------- | :--------- | :------------------------------- |
| `alerts` | `object[]` | List of alerts from Alertmanager |

---

## `get_silences`

> Get silences from Alertmanager.

**Usage Tips:**

- WHEN TO USE: - To see which alerts are currently silenced - To check active, pending, or expired silences - To investigate why certain alerts are not firing notifications
- FILTERING: - Use 'filter' to apply label matchers to find specific silences
- Silences are used to temporarily mute alerts based on label matchers. This tool helps you understand what is currently silenced in your environment.

**Parameters:**

| Parameter | Type     | Required | Description                                                             |
| :-------- | :------- | :------: | :---------------------------------------------------------------------- |
| `filter`  | `string` |          | Label matchers to filter silences (e.g., 'alertname=HighCPU', optional) |

**Output Schema:**

| Field      | Type       | Description                        |
| :--------- | :--------- | :--------------------------------- |
| `silences` | `object[]` | List of silences from Alertmanager |

---

## `get_current_time`

> Get the current date and time in RFC3339 format

|                |      |
| :------------- | :--- |
| **Parameters** | None |

---

## `tempo_list_instances`

> List all Tempo instances. The assistant should display the instances in a table.

|                |      |
| :------------- | :--- |
| **Parameters** | None |

---

## `tempo_get_trace_by_id`

> Get a trace by trace ID

**Parameters:**

| Parameter        | Type     | Required | Description                                  |
| :--------------- | :------- | :------: | :------------------------------------------- |
| `tempoName`      | `string` | ✅        | The name of the Tempo instance to query      |
| `tempoNamespace` | `string` | ✅        | The namespace of the Tempo instance to query |
| `tenant`         | `string` | ✅        | The tenant to query                          |
| `traceid`        | `string` | ✅        | TraceID of the trace                         |
| `end`            | `string` |          | End time in RFC 3339 format                  |
| `start`          | `string` |          | Start time in RFC 3339 format                |

---

## `tempo_search_traces`

> Search for traces in Tempo

**Parameters:**

| Parameter        | Type     | Required | Description                                  |
| :--------------- | :------- | :------: | :------------------------------------------- |
| `query`          | `string` | ✅        | Search query in the TraceQL query language   |
| `tempoName`      | `string` | ✅        | The name of the Tempo instance to query      |
| `tempoNamespace` | `string` | ✅        | The namespace of the Tempo instance to query |
| `tenant`         | `string` | ✅        | The tenant to query                          |
| `end`            | `string` |          | End time in RFC 3339 format                  |
| `limit`          | `number` |          | Maximum search results                       |
| `spss`           | `number` |          | Spans per span-set limit                     |
| `start`          | `string` |          | Start time in RFC 3339 format                |

---

## `tempo_search_tags`

> Search for tag names in Tempo

**Parameters:**

| Parameter        | Type     | Required | Description                                                                      |
| :--------------- | :------- | :------: | :------------------------------------------------------------------------------- |
| `tempoName`      | `string` | ✅        | The name of the Tempo instance to query                                          |
| `tempoNamespace` | `string` | ✅        | The namespace of the Tempo instance to query                                     |
| `tenant`         | `string` | ✅        | The tenant to query                                                              |
| `end`            | `string` |          | End time in RFC 3339 format                                                      |
| `limit`          | `number` |          | Maximum number of tag names per scope                                            |
| `maxStaleValues` | `number` |          | Search termination threshold for stale values                                    |
| `query`          | `string` |          | TraceQL query for filtering tag names                                            |
| `scope`          | `string` |          | Scope to filter tags: resource, span, intrinsic, event, link, or instrumentation |
| `start`          | `string` |          | Start time in RFC 3339 format                                                    |

---

## `tempo_search_tag_values`

> Search for tag values in Tempo

**Parameters:**

| Parameter        | Type     | Required | Description                                   |
| :--------------- | :------- | :------: | :-------------------------------------------- |
| `tag`            | `string` | ✅        | The tag name to get values for                |
| `tempoName`      | `string` | ✅        | The name of the Tempo instance to query       |
| `tempoNamespace` | `string` | ✅        | The namespace of the Tempo instance to query  |
| `tenant`         | `string` | ✅        | The tenant to query                           |
| `end`            | `string` |          | End time in RFC 3339 format                   |
| `limit`          | `number` |          | Maximum number of tag values to return        |
| `maxStaleValues` | `number` |          | Search termination threshold for stale values |
| `query`          | `string` |          | TraceQL query for filtering tag values        |
| `start`          | `string` |          | Start time in RFC 3339 format                 |

