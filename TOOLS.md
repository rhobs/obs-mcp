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

