package mcp

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rhobs/obs-mcp/pkg/logs"
	"github.com/rhobs/obs-mcp/pkg/metrics"
	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/traces"
)

// ToolGroup holds a named category of tools for documentation generation.
type ToolGroup struct {
	Name  string
	Icon  string
	Tools []mcp.Tool
}

// AllTools returns all available MCP tools.
// When adding a new tool, add it to the respective toolset's GetTools() method.
func AllTools() []mcp.Tool {
	var all []mcp.Tool
	for _, g := range GroupedTools() {
		all = append(all, g.Tools...)
	}
	return all
}

// GroupedTools returns tools organized by category for documentation.
func GroupedTools() []ToolGroup {
	allMetricsTools := toolsetToMCPTools(&metrics.Toolset{})
	var promTools, alertTools []mcp.Tool
	for _, t := range allMetricsTools {
		switch t.Name {
		case "get_alerts", "get_silences":
			alertTools = append(alertTools, t)
		default:
			promTools = append(promTools, t)
		}
	}

	return []ToolGroup{
		{Name: "Prometheus / Thanos", Icon: "📈", Tools: promTools},
		{Name: "Alertmanager", Icon: "🔔", Tools: alertTools},
		{Name: "Tempo (Distributed Tracing)", Icon: "🔍", Tools: toolsetToMCPTools(&traces.Toolset{})},
		{Name: "Loki (Log Management)", Icon: "📋", Tools: toolsetToMCPTools(&logs.Toolset{})},
		{Name: "OpenTelemetry Collector", Icon: "⚙️", Tools: toolsetToMCPTools(&otelcol.Toolset{})},
	}
}

// toolsetToMCPTools converts a Toolset's tools to mcp.Tool for documentation generation.
func toolsetToMCPTools(ts api.Toolset) []mcp.Tool {
	serverTools := ts.GetTools(nil)
	out := make([]mcp.Tool, len(serverTools))
	for i := range serverTools {
		st := serverTools[i]
		out[i] = apiToolToMCPTool(st.Tool)
	}
	return out
}

func apiToolToMCPTool(t api.Tool) mcp.Tool {
	inputSchema := make(map[string]any)
	if t.InputSchema != nil {
		inputSchema["type"] = t.InputSchema.Type
		if len(t.InputSchema.Properties) > 0 {
			props := make(map[string]any, len(t.InputSchema.Properties))
			for name, schema := range t.InputSchema.Properties {
				prop := map[string]any{
					"description": schema.Description,
				}
				if schema.Type != "" {
					prop["type"] = schema.Type
				}
				if schema.Pattern != "" {
					prop["pattern"] = schema.Pattern
				}
				props[name] = prop
			}
			inputSchema["properties"] = props
		}
		if len(t.InputSchema.Required) > 0 {
			required := make([]any, len(t.InputSchema.Required))
			for i, r := range t.InputSchema.Required {
				required[i] = r
			}
			inputSchema["required"] = required
		}
	}

	tool := mcp.Tool{
		Name:         t.Name,
		Description:  t.Description,
		InputSchema:  inputSchema,
		OutputSchema: t.OutputSchema,
	}
	if t.Annotations.Title != "" {
		tool.Title = t.Annotations.Title
	}
	if t.Meta != nil {
		tool.Meta = mcp.Meta(t.Meta)
	}
	return tool
}
