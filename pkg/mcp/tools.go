package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/rhobs/obs-mcp/pkg/perses"
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
	Metric map[string]string `json:"metric" jsonschema:"description=The metric labels as key-value pairs"`
	Values [][]any           `json:"values" jsonschema:"description=Array of [timestamp, value] pairs where timestamp is Unix epoch in seconds and value is the metric value"`
}

// CreateListMetricsTool creates the list_metrics tool definition.
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

// CreateExecuteRangeQueryTool creates the execute_range_query tool definition.
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

// DashboardsOutput defines the output schema for the list_perses_dashboards tool.
type DashboardsOutput struct {
	Dashboards []perses.DashboardInfo `json:"dashboards" jsonschema:"description=List of all PersesDashboard resources from the cluster with their metadata"`
}

// GetDashboardOutput defines the output schema for the get_perses_dashboard tool.
type GetDashboardOutput struct {
	Name      string         `json:"name" jsonschema:"description=Name of the Dashboard"`
	Namespace string         `json:"namespace" jsonschema:"description=Namespace where the Dashboard is located"`
	Spec      map[string]any `json:"spec" jsonschema:"description=The full dashboard specification including panels, layouts, variables, and datasources"`
}

// CreateListDashboardsTool creates the list_perses_dashboards tool definition.
func CreateListDashboardsTool() mcp.Tool {
	// "list_dashboards" conflicts with the same tool in layout-manager, and makes LCS throw duplicate tool name errors
	tool := mcp.NewTool("list_perses_dashboards",
		mcp.WithDescription(`List all PersesDashboard resources from the cluster.

Start here when there is a need to visualize metrics.

Returns dashboard summaries with name, namespace, labels, and descriptions.

Use the descriptions to identify dashboards relevant to the user's question.

In the case that there is insufficient information in the description, use get_perses_dashboard to fetch the full dashboard spec for more context. Doing so is an expensive operation, so only do this when necessary.

Follow up with get_dashboard_panels to see what panels are available in the relevant dashboard(s).
`),
		mcp.WithOutputSchema[DashboardsOutput](),
	)
	// workaround for tool with no parameter
	tool.InputSchema = mcp.ToolInputSchema{}
	tool.RawInputSchema = []byte(`{"type":"object","properties":{}}`)
	return tool
}

// CreateGetDashboardTool creates the get_perses_dashboard tool definition.
func CreateGetDashboardTool() mcp.Tool {
	// "get_dashboard" conflicts with the same tool in layout-manager, and makes LCS throw duplicate tool name errors
	return mcp.NewTool("get_perses_dashboard",
		mcp.WithDescription(`Get a specific Dashboard by name and namespace. This tool is used to get the dashboard's panels and configuration.

Use the list_perses_dashboards tool first to find available dashboards, then use this tool to get the full specification of a specific dashboard, if needed (to gather more context).

The intended use of this tool is only to gather more context on one or more dashboards when the description from list_perses_dashboards is insufficient.

Information about panels themselves should be gathered using get_dashboard_panels instead (e.g., looking at a "kind: Markdown" panel to gather more context).

Returns the dashboard's full specification including panels, layouts, variables, and datasources in JSON format.

For most use cases, you will want to follow up with get_dashboard_panels to extract panel metadata for selection.
`),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Dashboard"),
		),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the Dashboard"),
		),
		mcp.WithOutputSchema[GetDashboardOutput](),
	)
}

// GetDashboardPanelsOutput defines the output schema for the get_dashboard_panels tool.
type GetDashboardPanelsOutput struct {
	Name      string                   `json:"name" jsonschema:"description=Name of the dashboard"`
	Namespace string                   `json:"namespace" jsonschema:"description=Namespace of the dashboard"`
	Duration  string                   `json:"duration,omitempty" jsonschema:"description=Default time duration for queries extracted from dashboard spec (e.g. 1h, 24h)"`
	Panels    []*perses.DashboardPanel `json:"panels" jsonschema:"description=List of panel metadata including IDs, titles, queries, and chart types for LLM selection"`
}

// CreateGetDashboardPanelsTool creates the get_dashboard_panels tool definition.
func CreateGetDashboardPanelsTool() mcp.Tool {
	return mcp.NewTool("get_dashboard_panels",
		mcp.WithDescription(`Get panel(s) information from a specific Dashboard.

After finding a relevant dashboard (using list_perses_dashboards and conditionally, get_perses_dashboard), use this to see what panels it contains.

Returns panel metadata including:
- Panel IDs (format: 'panelName' or 'panelName-N' for multi-query panels)
- Titles and descriptions
- PromQL queries (may contain variables like $namespace)
- Chart types (TimeSeriesChart, PieChart, Table)

You can optionally provide specific panel IDs to fetch only those panels. This is useful when you remember panel IDs from earlier calls and want to re-fetch just their metadata without retrieving the entire dashboard's panels.

Use this information to identify which panels answer the user's question, then use format_panels_for_ui with the selected panel IDs to prepare them for display.
`),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the Dashboard"),
		),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the Dashboard"),
		),
		mcp.WithString("panel_ids",
			mcp.Description("Optional comma-separated list of panel IDs to filter. Panel IDs follow the format 'panelName' or 'panelName-N' where N is the query index (e.g. 'cpuUsage,memoryUsage-0,networkTraffic-1'). Use this to fetch metadata for specific panels you've seen in earlier calls. Leave empty to get all panels."),
		),
		mcp.WithOutputSchema[GetDashboardPanelsOutput](),
	)
}

// FormatPanelsForUIOutput defines the output schema for the format_panels_for_ui tool.
type FormatPanelsForUIOutput struct {
	Widgets []perses.DashboardWidget `json:"widgets" jsonschema:"description=Dashboard widgets in DashboardWidget format ready for direct rendering by genie-plugin UI"`
}

// CreateFormatPanelsForUITool creates the format_panels_for_ui tool definition.
func CreateFormatPanelsForUITool() mcp.Tool {
	return mcp.NewTool("format_panels_for_ui",
		mcp.WithDescription(`Format selected dashboard panels for UI rendering in DashboardWidget format.

After choosing relevant panels, use this to prepare them for display.

Returns an array of DashboardWidget objects ready for direct rendering, with:
- id: Unique panel identifier
- componentType: Perses component name (PersesTimeSeries, PersesPieChart, PersesTable)
- position: Grid layout coordinates (x, y, w, h) in 24-column grid
- breakpoint: Responsive grid breakpoint (xl/lg/md/sm) inferred from panel width
- props: Component properties (query, duration, step, start, end)

Panel IDs (fetched using get_dashboard_panels) must be provided to specify which panels to format.
`),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the dashboard containing the panels"),
		),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the dashboard"),
		),
		mcp.WithString("panel_ids",
			mcp.Required(), // Panel IDs are not required in get_dashboard_panels, but are required here to specify which panels to format
			mcp.Description("Comma-separated list of panel IDs to format (e.g. 'myPanelID-1,0_1-2')"),
		),
		mcp.WithOutputSchema[FormatPanelsForUIOutput](),
	)
}
