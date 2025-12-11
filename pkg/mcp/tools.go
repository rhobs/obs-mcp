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

// ListPersesDashboardsOutput defines the output schema for the list_perses_dashboards tool.
type ListPersesDashboardsOutput struct {
	Dashboards []perses.PersesDashboardInfo `json:"dashboards" jsonschema:"description=List of PersesDashboard objects from the cluster"`
}

func CreateListPersesDashboardsTool() mcp.Tool {
	return mcp.NewTool("list_perses_dashboards",
		mcp.WithDescription(`List all PersesDashboard custom resources from the Kubernetes cluster.

PersesDashboard is a Custom Resource from the Perses operator (https://github.com/perses/perses-operator) that defines
dashboard configurations. This tool returns summary information about all available dashboards in the form of a list of names, namespaces, labels, and descriptions.

IMPORTANT: Before using this tool, first check out_of_the_box_perses_dashboards for curated platform dashboards that are more likely to answer common user questions. 
Only use this tool if the out-of-the-box dashboards don't have what the user is looking for.

You can optionally filter by namespace and/or labels.

Once you have found the dashboard you need, you can use the get_perses_dashboard tool to get the dashboard's panels and configuration.

IMPORTANT: If you are looking for a specific dashboard, use this tool first to see if it exists. If it does, use the get_perses_dashboard tool to get the dashboard's panels and configuration.
`),
		mcp.WithString("namespace",
			mcp.Description("Optional namespace to filter dashboards. Leave empty to list from all namespaces."),
		),
		mcp.WithString("label_selector",
			mcp.Description("Optional Kubernetes label selector to filter dashboards (e.g., 'app=myapp', 'env=prod,team=platform', 'app in (foo,bar)'). Leave empty to list all dashboards."),
		),
		mcp.WithOutputSchema[ListPersesDashboardsOutput](),
	)
}

// OOTBPersesDashboardsOutput defines the output schema for the out_of_the_box_perses_dashboards tool.
type OOTBPersesDashboardsOutput struct {
	Dashboards []perses.PersesDashboardInfo `json:"dashboards" jsonschema:"description=List of curated out-of-the-box PersesDashboard definitions"`
}

// GetPersesDashboardOutput defines the output schema for the get_perses_dashboard tool.
type GetPersesDashboardOutput struct {
	Name      string                 `json:"name" jsonschema:"description=Name of the PersesDashboard"`
	Namespace string                 `json:"namespace" jsonschema:"description=Namespace where the PersesDashboard is located"`
	Spec      map[string]interface{} `json:"spec" jsonschema:"description=The full dashboard specification including panels, layouts, variables, and datasources"`
}

func CreateOOTBPersesDashboardsTool() mcp.Tool {
	tool := mcp.NewTool("out_of_the_box_perses_dashboards",
		mcp.WithDescription(`List curated out-of-the-box PersesDashboard definitions for the platform.

IMPORTANT: Use this tool FIRST when looking for dashboards. These are pre-configured, curated dashboards that cover common platform observability needs and are 
most likely to answer user questions about the platform.

Only fall back to list_perses_dashboards if the dashboards returned here don't have
what the user is looking for.

Returns a list of dashboard summaries with name, namespace, labels, and description explaining what each dashboard contains.
`),
		mcp.WithOutputSchema[OOTBPersesDashboardsOutput](),
	)
	// workaround for tool with no parameter
	tool.InputSchema = mcp.ToolInputSchema{}
	tool.RawInputSchema = []byte(`{"type":"object","properties":{}}`)
	return tool
}

func CreateGetPersesDashboardTool() mcp.Tool {
	return mcp.NewTool("get_perses_dashboard",
		mcp.WithDescription(`Get a specific PersesDashboard by name and namespace. This tool is used to get the dashboard's panels and configuration.

Use the list_perses_dashboards or out_of_the_box_perses_dashboards tool first to find available dashboards, then use this tool to get the full specification of a specific dashboard.

Returns the dashboard's full specification including panels, layouts, variables, and datasources in JSON format.

You can glean PromQL queries from the dashboard's panels and variables, as well as production context to allow you to answer a user's questions better.
`),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the PersesDashboard"),
		),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the PersesDashboard"),
		),
		mcp.WithOutputSchema[GetPersesDashboardOutput](),
	)
}
