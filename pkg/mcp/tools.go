package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// ListMetricsOutput defines the output schema for the list_metrics tool.
type ListMetricsOutput struct {
	Metrics []string `json:"metrics" jsonschema:"description=List of all available metric names in Prometheus"`
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
		mcp.WithDescription("List all available metrics in Prometheus"),
		mcp.WithOutputSchema[ListMetricsOutput](),
	)
	// workaround for tool with no parameter
	// see https://github.com/containers/kubernetes-mcp-server/pull/341/files#diff-8f8a99cac7a7cbb9c14477d40539efa1494b62835603244ba9f10e6be1c7e44c
	tool.InputSchema = mcp.ToolInputSchema{}
	tool.RawInputSchema = []byte(`{"type":"object","properties":{}}`)
	return tool
}

func CreateExecuteRangeQueryTool() mcp.Tool {
	return mcp.NewTool("execute_range_query",
		mcp.WithDescription(`Execute a PromQL range query with flexible time specification.

For current time data queries, use only the 'duration' parameter to specify how far back
to look from now (e.g., '1h' for last hour, '30m' for last 30 minutes). In that case
SET 'end' to 'NOW' and leave 'start' empty.

For historical data queries, use explicit 'start' and 'end' times.
`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("PromQL query string"),
		),
		mcp.WithString("step",
			mcp.Required(),
			mcp.Description("Query resolution step width (e.g., '15s', '1m', '1h')"),
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
