package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rhobs/obs-mcp/pkg/tempo"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

// AllTools returns all available MCP tools.
// When adding a new tool, add it to pkg/tools/definitions.go to keep both MCP and Toolset in sync, as well as docs.
func AllTools() []mcp.Tool {
	toolDefs := append(tools.AllTools(), tempo.AllTools()...)
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
