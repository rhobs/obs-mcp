package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// ListMetricsOutput defines the output schema for the list_metrics tool.
type ListMetricsOutput struct {
	Metrics []string `json:"metrics" jsonschema:"description=List of all available metric names"`
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
	Value  []any             `json:"value" jsonschema:"description=[timestamp, value] pair for the instant query"`
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
	ResultType string         `json:"resultType" jsonschema:"description=The type of result returned: matrix or vector or scalar"`
	Result     []SeriesResult `json:"result" jsonschema:"description=The query results as an array of time series"`
	Warnings   []string       `json:"warnings,omitempty" jsonschema:"description=Any warnings generated during query execution"`
}

// SeriesResult represents a single time series result from a range query.
type SeriesResult struct {
	Metric map[string]string `json:"metric" jsonschema:"description=The metric labels"`
	Values [][]any           `json:"values" jsonschema:"description=Array of [timestamp, value] pairs"`
}

// AllTools returns all available MCP tools.
// When adding a new tool, add it here to keep documentation in sync.
func AllTools() []mcp.Tool {
	return []mcp.Tool{
		CreateListMetricsTool(),
		CreateExecuteRangeQueryTool(),
	}
}

func CreateListMetricsTool() mcp.Tool {
	tool := mcp.NewTool("list_metrics",
		mcp.WithDescription(`MANDATORY FIRST STEP: List all available metric names in Prometheus.

YOU MUST CALL THIS TOOL BEFORE ANY OTHER QUERY TOOL

This tool MUST be called first for EVERY observability question to:
1. Discover what metrics actually exist in this environment
2. Find the EXACT metric name to use in queries
3. Avoid querying non-existent metrics

NEVER skip this step. NEVER guess metric names. Metric names vary between environments.

After calling this tool:
1. Search the returned list for relevant metrics
2. Use the EXACT metric name found in subsequent queries
3. If no relevant metric exists, inform the user
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
		mcp.WithDescription(`Execute a PromQL instant query to get current/point-in-time values.

PREREQUISITE: You MUST call list_metrics first to verify the metric exists

WHEN TO USE:
- Current state questions: "What is the current error rate?"
- Point-in-time snapshots: "How many pods are running?"
- Latest values: "Which pods are in Pending state?"

The 'query' parameter MUST use metric names that were returned by list_metrics.
`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("PromQL query string using metric names verified via list_metrics"),
		),
		mcp.WithString("time",
			mcp.Description("Evaluation time as RFC3339 or Unix timestamp. Omit or use 'NOW' for current time."),
		),
		mcp.WithOutputSchema[InstantQueryOutput](),
	)
}

func CreateExecuteRangeQueryTool() mcp.Tool {
	return mcp.NewTool("execute_range_query",
		mcp.WithDescription(`Execute a PromQL range query to get time-series data over a period.

PREREQUISITE: You MUST call list_metrics first to verify the metric exists

WHEN TO USE:
- Trends over time: "What was CPU usage over the last hour?"
- Rate calculations: "How many requests per second?"
- Historical analysis: "Were there any restarts in the last 5 minutes?"

TIME PARAMETERS:
- 'duration': Look back from now (e.g., "5m", "1h", "24h")
- 'step': Data point resolution (e.g., "1m" for 1-hour duration, "5m" for 24-hour duration)

The 'query' parameter MUST use metric names that were returned by list_metrics.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("PromQL query string using metric names verified via list_metrics"),
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
		mcp.WithDescription(`Get all label names (dimensions) available for filtering a metric.

WHEN TO USE (after calling list_metrics):
- To discover how to filter metrics (by namespace, pod, service, etc.)
- Before constructing label matchers in PromQL queries

The 'metric' parameter should use a metric name from list_metrics output.`),
		mcp.WithString("metric",
			mcp.Description("Metric name (from list_metrics) to get label names for. Leave empty for all metrics."),
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
		mcp.WithDescription(`Get all unique values for a specific label.

WHEN TO USE (after calling list_metrics and get_label_names):
- To find exact label values for filtering (namespace names, pod names, etc.)
- To see what values exist before constructing queries

The 'metric' parameter should use a metric name from list_metrics output.`),
		mcp.WithString("label",
			mcp.Required(),
			mcp.Description("Label name (from get_label_names) to get values for"),
		),
		mcp.WithString("metric",
			mcp.Description("Metric name (from list_metrics) to scope the label values to. Leave empty for all metrics."),
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
		mcp.WithDescription(`Get time series matching selectors and preview cardinality.

WHEN TO USE (optional, after calling list_metrics):
- To verify label filters match expected series before querying
- To check cardinality and avoid slow queries

CARDINALITY GUIDANCE:
- <100 series: Safe
- 100-1000: Usually fine
- >1000: Add more label filters

The selector should use metric names from list_metrics output.`),
		mcp.WithString("matches",
			mcp.Required(),
			mcp.Description("PromQL series selector using metric names from list_metrics"),
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
