package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/rhobs/obs-mcp/pkg/handlers"
	"github.com/rhobs/obs-mcp/pkg/tooldef"
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
	tool := tooldef.ListMetrics.ToMCPTool()
	// The syntax looks a bit odd, but essentially, WithOutputSchema is a generic function,
	// which returns a ToolOption with signature func (*Tool). We are essentially calling the returned
	// ToolOption on this tool.
	mcp.WithOutputSchema[handlers.ListMetricsOutput]()(&tool)
	return tool
}

func CreateExecuteInstantQueryTool() mcp.Tool {
	tool := tooldef.ExecuteInstantQuery.ToMCPTool()
	mcp.WithOutputSchema[handlers.InstantQueryOutput]()(&tool)
	return tool
}

func CreateExecuteRangeQueryTool() mcp.Tool {
	tool := tooldef.ExecuteRangeQuery.ToMCPTool()
	mcp.WithOutputSchema[handlers.RangeQueryOutput]()(&tool)
	return tool
}

func CreateGetLabelNamesTool() mcp.Tool {
	tool := tooldef.GetLabelNames.ToMCPTool()
	mcp.WithOutputSchema[handlers.LabelNamesOutput]()(&tool)
	return tool
}

func CreateGetLabelValuesTool() mcp.Tool {
	tool := tooldef.GetLabelValues.ToMCPTool()
	mcp.WithOutputSchema[handlers.LabelValuesOutput]()(&tool)
	return tool
}

func CreateGetSeriesTool() mcp.Tool {
	tool := tooldef.GetSeries.ToMCPTool()
	mcp.WithOutputSchema[handlers.SeriesOutput]()(&tool)
	return tool
}

func CreateGetAlertsTool() mcp.Tool {
	tool := tooldef.GetAlerts.ToMCPTool()
	mcp.WithOutputSchema[handlers.AlertsOutput]()(&tool)
	return tool
}

func CreateGetSilencesTool() mcp.Tool {
	tool := tooldef.GetSilences.ToMCPTool()
	mcp.WithOutputSchema[handlers.SilencesOutput]()(&tool)
	return tool
}
