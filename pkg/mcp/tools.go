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

// DashboardsOutput defines the output schema for the list_perses_dashboards tool.
type DashboardsOutput struct {
	Dashboards []perses.DashboardInfo `json:"dashboards" jsonschema:"List of all PersesDashboard resources from the cluster with their metadata"`
}

var ListDashboards = tools.ToolDef[DashboardsOutput]{
	Name: "list_perses_dashboards",
	Description: `List all PersesDashboard resources from the cluster.

Start here when there is a need to visualize metrics.

Returns dashboard summaries with name, namespace, labels, and descriptions.

Use the descriptions to identify dashboards relevant to the user's question.

In the case that there is insufficient information in the description, use get_perses_dashboard to fetch the full dashboard spec for more context. Doing so is an expensive operation, so only do this when necessary.

Follow up with get_dashboard_panels to see what panels are available in the relevant dashboard(s).
`,
	Title:    "List Dashboards",
	ReadOnly: true,
}

func CreateListDashboardsTool() *mcp.Tool {
	return ListDashboards.ToMCPTool()
}

// GetDashboardOutput defines the output schema for the get_perses_dashboard tool.
type GetDashboardOutput struct {
	Name      string         `json:"name" jsonschema:"Name of the Dashboard"`
	Namespace string         `json:"namespace" jsonschema:"Namespace where the Dashboard is located"`
	Spec      map[string]any `json:"spec" jsonschema:"The full dashboard specification including panels, layouts, variables, and datasources"`
}

// GetDashboardInput defines the input schema for the get_perses_dashboard tool.
type GetDashboardInput struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

var GetDashboard = tools.ToolDef[GetDashboardOutput]{
	Name: "get_perses_dashboard",
	Description: `Get a specific Dashboard by name and namespace. This tool is used to get the dashboard's panels and configuration.

Use the list_perses_dashboards tool first to find available dashboards, then use this tool to get the full specification of a specific dashboard, if needed (to gather more context).

The intended use of this tool is only to gather more context on one or more dashboards when the description from list_perses_dashboards is insufficient.

Information about panels themselves should be gathered using get_dashboard_panels instead (e.g., looking at a "kind: Markdown" panel to gather more context).

Returns the dashboard's full specification including panels, layouts, variables, and datasources in JSON format.

For most use cases, you will want to follow up with get_dashboard_panels to extract panel metadata for selection.
`,
	Title:    "Get Dashboard",
	ReadOnly: true,
	Params: []tools.ParamDef{
		{
			Name:        "name",
			Type:        tools.ParamTypeString,
			Description: "Name of the Dashboard",
			Required:    true,
		},
		{
			Name:        "namespace",
			Type:        tools.ParamTypeString,
			Description: "Namespace of the Dashboard",
			Required:    true,
		},
	},
}

// GetDashboardPanelsOutput defines the output schema for the get_dashboard_panels tool.
type GetDashboardPanelsOutput struct {
	Name      string                   `json:"name" jsonschema:"Name of the dashboard"`
	Namespace string                   `json:"namespace" jsonschema:"Namespace of the dashboard"`
	Duration  string                   `json:"duration,omitempty" jsonschema:"Default time duration for queries extracted from dashboard spec (e.g. 1h, 24h)"`
	Panels    []*perses.DashboardPanel `json:"panels" jsonschema:"List of panel metadata including IDs, titles, queries, and chart types for LLM selection"`
}

// GetDashboardPanelsInput defines the input schema for the get_dashboard_panels tool.
type GetDashboardPanelsInput struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	PanelIDs  string `json:"panel_ids"`
}

var GetDashboardPanels = tools.ToolDef[GetDashboardPanelsOutput]{
	Name: "get_dashboard_panels",
	Description: `Get panel(s) information from a specific Dashboard.

After finding a relevant dashboard (using list_perses_dashboards and conditionally, get_perses_dashboard), use this to see what panels it contains.

Returns panel metadata including:
- Panel IDs (format: 'panelName' or 'panelName-N' for multi-query panels)
- Titles and descriptions
- PromQL queries (may contain variables like $namespace)
- Chart types (TimeSeriesChart, PieChart, Table)

You can optionally provide specific panel IDs to fetch only those panels. This is useful when you remember panel IDs from earlier calls and want to re-fetch just their metadata without retrieving the entire dashboard's panels.

Use this information to identify which panels answer the user's question, then use format_panels_for_ui with the selected panel IDs to prepare them for display.
`,
	Title:    "Get Dashboard Panels",
	ReadOnly: true,
	Params: []tools.ParamDef{
		{
			Name:        "name",
			Type:        tools.ParamTypeString,
			Description: "Name of the Dashboard",
			Required:    true,
		},
		{
			Name:        "namespace",
			Type:        tools.ParamTypeString,
			Description: "Namespace of the Dashboard",
			Required:    true,
		},
		{
			Name:        "panel_ids",
			Type:        tools.ParamTypeString,
			Description: "Optional comma-separated list of panel IDs to filter. Panel IDs follow the format 'panelName' or 'panelName-N' where N is the query index (e.g. 'cpuUsage,memoryUsage-0,networkTraffic-1'). Use this to fetch metadata for specific panels you've seen in earlier calls. Leave empty to get all panels.",
		},
	},
}

// FormatPanelsForUIOutput defines the output schema for the format_panels_for_ui tool.
type FormatPanelsForUIOutput struct {
	Widgets []perses.DashboardWidget `json:"widgets" jsonschema:"Dashboard widgets in DashboardWidget format ready for direct rendering by genie-plugin UI"`
}

// FormatPanelsForUIInput defines the input schema for the format_panels_for_ui tool.
type FormatPanelsForUIInput struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	PanelIDs  string `json:"panel_ids"`
}

var FormatPanelsForUI = tools.ToolDef[FormatPanelsForUIOutput]{
	Name: "format_panels_for_ui",
	Description: `Format selected dashboard panels for UI rendering in DashboardWidget format.

After choosing relevant panels, use this to prepare them for display.

Returns an array of DashboardWidget objects ready for direct rendering, with:
- id: Unique panel identifier
- componentType: Perses component name (PersesTimeSeries, PersesPieChart, PersesTable)
- position: Grid layout coordinates (x, y, w, h) in 24-column grid
- breakpoint: Responsive grid breakpoint (xl/lg/md/sm) inferred from panel width
- props: Component properties (query, duration, step, start, end)

Panel IDs (fetched using get_dashboard_panels) must be provided to specify which panels to format.
`,
	Title:    "Format Panels for UI",
	ReadOnly: true,
	Params: []tools.ParamDef{
		{
			Name:        "name",
			Type:        tools.ParamTypeString,
			Description: "Name of the dashboard containing the panels",
			Required:    true,
		},
		{
			Name:        "namespace",
			Type:        tools.ParamTypeString,
			Description: "Namespace of the dashboard",
			Required:    true,
		},
		{
			Name:        "panel_ids",
			Type:        tools.ParamTypeString,
			Description: "Comma-separated list of panel IDs to format (e.g. 'myPanelID-1,0_1-2')",
			Required:    true,
		},
	},
}
