package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/rhobs/obs-mcp/pkg/handlers"
	"github.com/rhobs/obs-mcp/pkg/prompts"
)

// AllTools returns all available MCP tools.
// When adding a new tool, add it here to keep documentation in sync.
func AllTools() []mcp.Tool {
	return []mcp.Tool{
		CreateListMetricsTool(),
		CreateExecuteInstantQueryTool(),
		CreateExecuteRangeQueryTool(),
		CreateGetLabelNamesTool(),
		CreateGetLabelValuesTool(),
		CreateGetSeriesTool(),
		CreateGetAlertsTool(),
		CreateGetSilencesTool(),
	}
}

func CreateListMetricsTool() mcp.Tool {
	tool := mcp.NewTool("list_metrics",
		mcp.WithDescription(prompts.ListMetricsPrompt),
		mcp.WithOutputSchema[handlers.ListMetricsOutput](),
	)
	// workaround for tool with no parameter
	// see https://github.com/containers/kubernetes-mcp-server/pull/341/files#diff-8f8a99cac7a7cbb9c14477d40539efa1494b62835603244ba9f10e6be1c7e44c
	tool.InputSchema = mcp.ToolInputSchema{}
	tool.RawInputSchema = []byte(`{"type":"object","properties":{}}`)
	return tool
}

func CreateExecuteInstantQueryTool() mcp.Tool {
	return mcp.NewTool("execute_instant_query",
		mcp.WithDescription(prompts.ExecuteInstantQueryPrompt),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("PromQL query string using metric names verified via list_metrics"),
		),
		mcp.WithString("time",
			mcp.Description("Evaluation time as RFC3339 or Unix timestamp. Omit or use 'NOW' for current time."),
		),
		mcp.WithOutputSchema[handlers.InstantQueryOutput](),
	)
}

func CreateExecuteRangeQueryTool() mcp.Tool {
	return mcp.NewTool("execute_range_query",
		mcp.WithDescription(prompts.ExecuteRangeQueryPrompt),
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
		mcp.WithOutputSchema[handlers.RangeQueryOutput](),
	)
}

func CreateGetLabelNamesTool() mcp.Tool {
	return mcp.NewTool("get_label_names",
		mcp.WithDescription(prompts.GetLabelNamesPrompt),
		mcp.WithString("metric",
			mcp.Description("Metric name (from list_metrics) to get label names for. Leave empty for all metrics."),
		),
		mcp.WithString("start",
			mcp.Description("Start time for label discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)"),
		),
		mcp.WithString("end",
			mcp.Description("End time for label discovery as RFC3339 or Unix timestamp (optional, defaults to now)"),
		),
		mcp.WithOutputSchema[handlers.LabelNamesOutput](),
	)
}

func CreateGetLabelValuesTool() mcp.Tool {
	return mcp.NewTool("get_label_values",
		mcp.WithDescription(prompts.GetLabelValuesPrompt),
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
		mcp.WithOutputSchema[handlers.LabelValuesOutput](),
	)
}

func CreateGetSeriesTool() mcp.Tool {
	return mcp.NewTool("get_series",
		mcp.WithDescription(prompts.GetSeriesPrompt),
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
		mcp.WithOutputSchema[handlers.SeriesOutput](),
	)
}

func CreateGetAlertsTool() mcp.Tool {
	return mcp.NewTool("get_alerts",
		mcp.WithDescription(prompts.GetAlertsPrompt),
		mcp.WithBoolean("active",
			mcp.Description("Filter for active alerts only (true/false, optional)"),
		),
		mcp.WithBoolean("silenced",
			mcp.Description("Filter for silenced alerts only (true/false, optional)"),
		),
		mcp.WithBoolean("inhibited",
			mcp.Description("Filter for inhibited alerts only (true/false, optional)"),
		),
		mcp.WithBoolean("unprocessed",
			mcp.Description("Filter for unprocessed alerts only (true/false, optional)"),
		),
		mcp.WithString("filter",
			mcp.Description("Label matchers to filter alerts (e.g., 'alertname=HighCPU', optional)"),
		),
		mcp.WithString("receiver",
			mcp.Description("Receiver name to filter alerts (optional)"),
		),
		mcp.WithOutputSchema[handlers.AlertsOutput](),
	)
}

func CreateGetSilencesTool() mcp.Tool {
	return mcp.NewTool("get_silences",
		mcp.WithDescription(prompts.GetSilencesPrompt),
		mcp.WithString("filter",
			mcp.Description("Label matchers to filter silences (e.g., 'alertname=HighCPU', optional)"),
		),
		mcp.WithOutputSchema[handlers.SilencesOutput](),
	)
}
