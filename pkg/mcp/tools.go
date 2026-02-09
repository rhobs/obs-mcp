package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/rhobs/obs-mcp/pkg/tools"
)

// AllTools returns all available MCP tools.
// When adding a new tool, add it to pkg/tooldef/definitions.go to keep both MCP and Toolset in sync, as well as docs.
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
	tool := tools.ListMetrics.ToMCPTool()
	// The syntax looks a bit odd, but essentially, WithOutputSchema is a generic function,
	// which returns a ToolOption with signature func (*Tool). We are essentially calling the returned
	// ToolOption on this tool.
	mcp.WithOutputSchema[tools.ListMetricsOutput]()(&tool)
	return tool
}

func CreateExecuteInstantQueryTool() mcp.Tool {
	tool := tools.ExecuteInstantQuery.ToMCPTool()
	mcp.WithOutputSchema[tools.InstantQueryOutput]()(&tool)
	return tool
}

func CreateExecuteRangeQueryTool() mcp.Tool {
	tool := tools.ExecuteRangeQuery.ToMCPTool()
	mcp.WithOutputSchema[tools.RangeQueryOutput]()(&tool)
	return tool
}

func CreateGetLabelNamesTool() mcp.Tool {
	tool := tools.GetLabelNames.ToMCPTool()
	mcp.WithOutputSchema[tools.LabelNamesOutput]()(&tool)
	return tool
}

func CreateGetLabelValuesTool() mcp.Tool {
	tool := tools.GetLabelValues.ToMCPTool()
	mcp.WithOutputSchema[tools.LabelValuesOutput]()(&tool)
	return tool
}

func CreateGetSeriesTool() mcp.Tool {
	tool := tools.GetSeries.ToMCPTool()
	mcp.WithOutputSchema[tools.SeriesOutput]()(&tool)
	return tool
}

func CreateGetAlertsTool() mcp.Tool {
	tool := tools.GetAlerts.ToMCPTool()
	mcp.WithOutputSchema[tools.AlertsOutput]()(&tool)
	return tool
}

func CreateGetSilencesTool() mcp.Tool {
	tool := tools.GetSilences.ToMCPTool()
	mcp.WithOutputSchema[tools.SilencesOutput]()(&tool)
	return tool
}
