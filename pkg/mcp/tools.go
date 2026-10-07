package mcp

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/api"

	"github.com/rhobs/obs-mcp/pkg/clusterinspector"
	"github.com/rhobs/obs-mcp/pkg/logs"
	"github.com/rhobs/obs-mcp/pkg/metrics"
	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/traces"
)

// ToolGroup holds a named category of tools for documentation generation.
type ToolGroup struct {
	Name  string
	Icon  string
	Tools []api.ServerTool
}

// GroupedTools returns tools organized by category for documentation.
func GroupedTools() []ToolGroup {
	allMetricsTools := (&metrics.Toolset{}).GetTools(context.Background(), api.ToolsetContext{Inspector: clusterinspector.New()})
	var promTools, alertTools []api.ServerTool
	for i := range allMetricsTools {
		switch allMetricsTools[i].Tool.Name {
		case "get_alerts", "get_silences":
			alertTools = append(alertTools, allMetricsTools[i])
		default:
			promTools = append(promTools, allMetricsTools[i])
		}
	}

	return []ToolGroup{
		{Name: "Prometheus / Thanos", Icon: "📈", Tools: promTools},
		{Name: "Alertmanager", Icon: "🔔", Tools: alertTools},
		{Name: "Tempo (Distributed Tracing)", Icon: "🔍", Tools: (&traces.Toolset{}).GetTools(context.Background(), api.ToolsetContext{Inspector: clusterinspector.New()})},
		{Name: "Loki (Log Management)", Icon: "📋", Tools: (&logs.Toolset{}).GetTools(context.Background(), api.ToolsetContext{Inspector: clusterinspector.New()})},
		{Name: "OpenTelemetry Collector", Icon: "⚙️", Tools: (&otelcol.Toolset{}).GetTools(context.Background(), api.ToolsetContext{Inspector: clusterinspector.New()})},
	}
}
