package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rhobs/obs-mcp/pkg/perses"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

// AllTools returns all available MCP tools.
// When adding a new tool, add it to pkg/tools/definitions.go to keep both MCP and Toolset in sync, as well as docs.
func AllTools() []mcp.Tool {
	toolDefs := tools.AllTools()
	mcpTools := make([]mcp.Tool, len(toolDefs))

	for i, toolDef := range toolDefs {
		mcpTools[i] = *toolDef.ToMCPTool()
	}

	return mcpTools
}

// Individual tool creation functions for backward compatibility and testing
func CreateListMetricsTool() mcp.Tool {
	return *tools.ListMetrics.ToMCPTool()
}

func CreateExecuteInstantQueryTool() mcp.Tool {
	return *tools.ExecuteInstantQuery.ToMCPTool()
}

func CreateExecuteRangeQueryTool() mcp.Tool {
	return *tools.ExecuteRangeQuery.ToMCPTool()
}

func CreateShowTimeseriesTool() mcp.Tool {
	// For UI purposes only, no additional data to be sent to the LLM context.
	return *tools.ShowTimeseries.ToMCPTool()
}

func CreateGetLabelNamesTool() mcp.Tool {
	return *tools.GetLabelNames.ToMCPTool()
}

func CreateGetLabelValuesTool() mcp.Tool {
	return *tools.GetLabelValues.ToMCPTool()
}

func CreateGetSeriesTool() mcp.Tool {
	return *tools.GetSeries.ToMCPTool()
}

func CreateGetAlertsTool() mcp.Tool {
	return *tools.GetAlerts.ToMCPTool()
}

func CreateGetSilencesTool() mcp.Tool {
	return *tools.GetSilences.ToMCPTool()
}

// ListPersesDashboardsOutput defines the output schema for the list_perses_dashboards tool.
type ListPersesDashboardsOutput struct {
	Dashboards []perses.PersesDashboardInfo `json:"dashboards" jsonschema:"description=List of PersesDashboard objects from the cluster"`
}

// ListPersesDashboardsInput defines the input schema for the list_perses_dashboards tool.
type ListPersesDashboardsInput struct {
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"label_selector"`
}

var ListPersesDashboards = tools.ToolDef[ListPersesDashboardsOutput]{
	Name: "list_perses_dashboards",
	Description: `List all PersesDashboard custom resources from the Kubernetes cluster.

PersesDashboard is a Custom Resource from the Perses operator (https://github.com/perses/perses-operator) that defines
dashboard configurations. This tool returns summary information about all available dashboards in the form of a list of names, namespaces, labels, and descriptions.

IMPORTANT: Before using this tool, first check out_of_the_box_perses_dashboards for curated platform dashboards that are more likely to answer common user questions.
Only use this tool if the out-of-the-box dashboards don't have what the user is looking for.

You can optionally filter by namespace and/or labels.

Once you have found the dashboard you need, you can use the get_perses_dashboard tool to get the dashboard's panels and configuration.

IMPORTANT: If you are looking for a specific dashboard, use this tool first to see if it exists. If it does, use the get_perses_dashboard tool to get the dashboard's panels and configuration.
`,
	Title:    "List Perses Dashboards",
	ReadOnly: true,
	Params: []tools.ParamDef{
		{
			Name:        "namespace",
			Type:        tools.ParamTypeString,
			Description: "Optional namespace to filter dashboards. Leave empty to list from all namespaces.",
		},
		{
			Name:        "label_selector",
			Type:        tools.ParamTypeString,
			Description: "Optional Kubernetes label selector to filter dashboards (e.g., 'app=myapp', 'env=prod,team=platform', 'app in (foo,bar)'). Leave empty to list all dashboards.",
		},
	},
}

func CreateListPersesDashboardsTool() *mcp.Tool {
	return ListPersesDashboards.ToMCPTool()
}

// OOTBPersesDashboardsOutput defines the output schema for the out_of_the_box_perses_dashboards tool.
type OOTBPersesDashboardsOutput struct {
	Dashboards []perses.PersesDashboardInfo `json:"dashboards" jsonschema:"description=List of curated out-of-the-box PersesDashboard definitions"`
}

var OOTBPersesDashboards = tools.ToolDef[OOTBPersesDashboardsOutput]{
	Name: "out_of_the_box_perses_dashboards",
	Description: `List curated out-of-the-box PersesDashboard definitions for the platform.

IMPORTANT: Use this tool FIRST when looking for dashboards. These are pre-configured, curated dashboards that cover common platform observability needs and are
most likely to answer user questions about the platform.

Only fall back to list_perses_dashboards if the dashboards returned here don't have
what the user is looking for.

Returns a list of dashboard summaries with name, namespace, labels, and description explaining what each dashboard contains.
`,
	Title:    "Out of the Box Perses Dashboards",
	ReadOnly: true,
}

func CreateOOTBPersesDashboardsTool() *mcp.Tool {
	return OOTBPersesDashboards.ToMCPTool()
}

// GetPersesDashboardOutput defines the output schema for the get_perses_dashboard tool.
type GetPersesDashboardOutput struct {
	Name      string                 `json:"name" jsonschema:"description=Name of the PersesDashboard"`
	Namespace string                 `json:"namespace" jsonschema:"description=Namespace where the PersesDashboard is located"`
	Spec      map[string]interface{} `json:"spec" jsonschema:"description=The full dashboard specification including panels, layouts, variables, and datasources"`
}

// GetPersesDashboardInput defines the input schema for the get_perses_dashboard tool.
type GetPersesDashboardInput struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

var GetPersesDashboard = tools.ToolDef[GetPersesDashboardOutput]{
	Name: "get_perses_dashboard",
	Description: `Get a specific PersesDashboard by name and namespace. This tool is used to get the dashboard's panels and configuration.

Use the list_perses_dashboards or out_of_the_box_perses_dashboards tool first to find available dashboards, then use this tool to get the full specification of a specific dashboard.

Returns the dashboard's full specification including panels, layouts, variables, and datasources in JSON format.

You can glean PromQL queries from the dashboard's panels and variables, as well as production context to allow you to answer a user's questions better.
`,
	Title:    "Get Perses Dashboard",
	ReadOnly: true,
	Params: []tools.ParamDef{
		{
			Name:        "name",
			Type:        tools.ParamTypeString,
			Description: "Name of the PersesDashboard",
			Required:    true,
		},
		{
			Name:        "namespace",
			Type:        tools.ParamTypeString,
			Description: "Namespace of the PersesDashboard",
			Required:    true,
		},
	},
}
