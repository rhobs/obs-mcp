# Available Tools

This MCP server exposes the following tools for interacting with Prometheus/Thanos:

## `list_metrics`

> List all available metrics

|                |      |
| :------------- | :--- |
| **Parameters** | None |

**Output Schema:**

| Field     | Type       | Description                        |
| :-------- | :--------- | :--------------------------------- |
| `metrics` | `string[]` | List of all available metric names |

---

## `execute_range_query`

> Execute a PromQL range query with flexible time specification.

**Usage Tips:**

- For current time data queries, use only the 'duration' parameter to specify how far back to look from now (e.g., '1h' for last hour, '30m' for last 30 minutes). In that case SET 'end' to 'NOW' and leave 'start' empty.
- For historical data queries, use explicit 'start' and 'end' times.

**Parameters:**

| Parameter  | Type     | Required | Description                                                                   |
| :--------- | :------- | :------: | :---------------------------------------------------------------------------- |
| `query`    | `string` | ✅        | PromQL query string                                                           |
| `step`     | `string` | ✅        | Query resolution step width (e.g., '15s', '1m', '1h')                         |
| `duration` | `string` |          | Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional)     |
| `end`      | `string` |          | End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time. |
| `start`    | `string` |          | Start time as RFC3339 or Unix timestamp (optional)                            |

> [!NOTE]
> Parameters with patterns must match: `^\d+[smhdwy]$`

**Output Schema:**

| Field        | Type       | Description                                             |
| :----------- | :--------- | :------------------------------------------------------ |
| `result`     | `object[]` | The query results as an array of time series            |
| `resultType` | `string`   | The type of result returned: matrix or vector or scalar |
| `warnings`   | `string[]` | Any warnings generated during query execution           |

