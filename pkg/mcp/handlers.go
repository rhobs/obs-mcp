package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/perses"
	"github.com/rhobs/obs-mcp/pkg/resultutil"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

// ListMetricsHandler handles the listing of available Prometheus metrics.
func ListMetricsHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[tools.ListMetricsInput, tools.ListMetricsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.ListMetricsInput) (*mcp.CallToolResult, tools.ListMetricsOutput, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return nil, tools.ListMetricsOutput{}, fmt.Errorf("failed to create Prometheus client: %w", err)
		}

		result := tools.ListMetricsHandler(ctx, promClient, input)
		output, err := resultutil.Unwrap[tools.ListMetricsOutput](result)
		if err != nil {
			return nil, tools.ListMetricsOutput{}, err
		}
		return nil, output, nil
	}
}

// ShowTimeseriesHandler handles the show_timeseries tool.
func ShowTimeseriesHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[tools.ShowTimeseriesInput, struct{}] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.ShowTimeseriesInput) (*mcp.CallToolResult, struct{}, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return nil, struct{}{}, fmt.Errorf("failed to create Prometheus client: %w", err)
		}

		result := tools.ShowTimeseriesHandler(ctx, promClient, input)
		_, err = resultutil.Unwrap[struct{}](result)
		if err != nil {
			return nil, struct{}{}, err
		}
		// We return empty result to not overwhelm the LLM context. The purpose
		// of the tool is to validate the query. The visualization is taking the
		// required data from the tool inputs. An UI-only tool could be introduced
		// to load the data for the visualization, if needed (MCP-apps case).
		return nil, struct{}{}, nil
	}
}

// ExecuteInstantQueryHandler handles the execution of Prometheus instant queries.
func ExecuteInstantQueryHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[tools.InstantQueryInput, tools.InstantQueryOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.InstantQueryInput) (*mcp.CallToolResult, tools.InstantQueryOutput, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return nil, tools.InstantQueryOutput{}, fmt.Errorf("failed to create Prometheus client: %w", err)
		}

		result := tools.ExecuteInstantQueryHandler(ctx, promClient, input)
		output, err := resultutil.Unwrap[tools.InstantQueryOutput](result)
		if err != nil {
			return nil, tools.InstantQueryOutput{}, err
		}
		return nil, output, nil
	}
}

// ExecuteRangeQueryHandler handles the execution of Prometheus range queries.
func ExecuteRangeQueryHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[tools.RangeQueryInput, tools.RangeQueryOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.RangeQueryInput) (*mcp.CallToolResult, tools.RangeQueryOutput, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return nil, tools.RangeQueryOutput{}, fmt.Errorf("failed to create Prometheus client: %w", err)
		}

		result := tools.ExecuteRangeQueryHandler(ctx, promClient, input, opts.FullRangeQueryResponse)
		output, err := resultutil.Unwrap[tools.RangeQueryOutput](result)
		if err != nil {
			return nil, tools.RangeQueryOutput{}, err
		}
		return nil, output, nil
	}
}

// GetLabelNamesHandler handles the retrieval of label names.
func GetLabelNamesHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[tools.LabelNamesInput, tools.LabelNamesOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.LabelNamesInput) (*mcp.CallToolResult, tools.LabelNamesOutput, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return nil, tools.LabelNamesOutput{}, fmt.Errorf("failed to create Prometheus client: %w", err)
		}

		result := tools.GetLabelNamesHandler(ctx, promClient, input)
		output, err := resultutil.Unwrap[tools.LabelNamesOutput](result)
		if err != nil {
			return nil, tools.LabelNamesOutput{}, err
		}
		return nil, output, nil
	}
}

// GetLabelValuesHandler handles the retrieval of label values.
func GetLabelValuesHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[tools.LabelValuesInput, tools.LabelValuesOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.LabelValuesInput) (*mcp.CallToolResult, tools.LabelValuesOutput, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return nil, tools.LabelValuesOutput{}, fmt.Errorf("failed to create Prometheus client: %w", err)
		}

		result := tools.GetLabelValuesHandler(ctx, promClient, input)
		output, err := resultutil.Unwrap[tools.LabelValuesOutput](result)
		if err != nil {
			return nil, tools.LabelValuesOutput{}, err
		}
		return nil, output, nil
	}
}

// GetSeriesHandler handles the retrieval of time series.
func GetSeriesHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[tools.SeriesInput, tools.SeriesOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.SeriesInput) (*mcp.CallToolResult, tools.SeriesOutput, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return nil, tools.SeriesOutput{}, fmt.Errorf("failed to create Prometheus client: %w", err)
		}

		result := tools.GetSeriesHandler(ctx, promClient, input)
		output, err := resultutil.Unwrap[tools.SeriesOutput](result)
		if err != nil {
			return nil, tools.SeriesOutput{}, err
		}
		return nil, output, nil
	}
}

// GetAlertsHandler handles the retrieval of alerts from Alertmanager.
func GetAlertsHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[tools.AlertsInput, tools.AlertsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.AlertsInput) (*mcp.CallToolResult, tools.AlertsOutput, error) {
		amClient, err := getAlertmanagerClient(ctx, opts)
		if err != nil {
			return nil, tools.AlertsOutput{}, fmt.Errorf("failed to create Alertmanager client: %w", err)
		}

		result := tools.GetAlertsHandler(ctx, amClient, input)
		output, err := resultutil.Unwrap[tools.AlertsOutput](result)
		if err != nil {
			return nil, tools.AlertsOutput{}, err
		}
		return nil, output, nil
	}
}

// GetSilencesHandler handles the retrieval of silences from Alertmanager.
func GetSilencesHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[tools.SilencesInput, tools.SilencesOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.SilencesInput) (*mcp.CallToolResult, tools.SilencesOutput, error) {
		amClient, err := getAlertmanagerClient(ctx, opts)
		if err != nil {
			return nil, tools.SilencesOutput{}, fmt.Errorf("failed to create Alertmanager client: %w", err)
		}

		result := tools.GetSilencesHandler(ctx, amClient, input)
		output, err := resultutil.Unwrap[tools.SilencesOutput](result)
		if err != nil {
			return nil, tools.SilencesOutput{}, err
		}
		return nil, output, nil
	}
}

// DashboardsHandler handles returning all dashboards from the cluster.
func DashboardsHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[struct{}, DashboardsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, DashboardsOutput, error) {
		slog.Info("DashboardsHandler called")

		dashboards, err := k8s.ListDashboards(ctx, "", "")
		if err != nil {
			return nil, DashboardsOutput{}, fmt.Errorf("failed to list dashboards: %w", err)
		}

		dashboardInfos := make([]perses.DashboardInfo, 0, len(dashboards))
		for _, dashboard := range dashboards {
			info := perses.DashboardInfo{
				Name:      dashboard.Name,
				Namespace: dashboard.Namespace,
				Labels:    dashboard.Labels,
			}

			if dashboard.Annotations != nil {
				if desc, ok := dashboard.Annotations[k8s.MCPHelpAnnotation]; ok {
					info.Description = desc
				}
			}

			dashboardInfos = append(dashboardInfos, info)
		}

		slog.Info("DashboardsHandler executed successfully", "dashboardCount", len(dashboardInfos))

		return nil, DashboardsOutput{Dashboards: dashboardInfos}, nil
	}
}

// GetDashboardHandler handles getting a specific dashboard by name and namespace.
func GetDashboardHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[GetDashboardInput, GetDashboardOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input GetDashboardInput) (*mcp.CallToolResult, GetDashboardOutput, error) {
		slog.Info("GetDashboardHandler called")

		spec, err := k8s.GetDashboard(ctx, input.Namespace, input.Name)
		if err != nil {
			return nil, GetDashboardOutput{}, fmt.Errorf("failed to get Dashboard: %w", err)
		}

		slog.Info("GetDashboardHandler executed successfully", "name", input.Name, "namespace", input.Namespace)

		return nil, GetDashboardOutput{
			Name:      input.Name,
			Namespace: input.Namespace,
			Spec:      spec,
		}, nil
	}
}

// GetDashboardPanelsHandler handles getting panel metadata from a dashboard for LLM selection.
func GetDashboardPanelsHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[GetDashboardPanelsInput, GetDashboardPanelsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input GetDashboardPanelsInput) (*mcp.CallToolResult, GetDashboardPanelsOutput, error) {
		slog.Info("GetDashboardPanelsHandler called")

		var panelIDs []string
		if input.PanelIDs != "" {
			for _, part := range splitByComma(input.PanelIDs) {
				if part != "" {
					panelIDs = append(panelIDs, part)
				}
			}
		}

		spec, err := k8s.GetDashboard(ctx, input.Namespace, input.Name)
		if err != nil {
			return nil, GetDashboardPanelsOutput{}, fmt.Errorf("failed to get dashboard: %w", err)
		}

		panels := perses.ExtractPanels(input.Name, input.Namespace, spec, false, panelIDs)

		duration := "1h"
		if d, ok := spec["duration"].(string); ok {
			duration = d
		}

		slog.Info("GetDashboardPanelsHandler executed successfully",
			"name", input.Name,
			"namespace", input.Namespace,
			"requested", len(panelIDs),
			"returned", len(panels))

		return nil, GetDashboardPanelsOutput{
			Name:      input.Name,
			Namespace: input.Namespace,
			Duration:  duration,
			Panels:    panels,
		}, nil
	}
}

// FormatPanelsForUIHandler handles formatting selected panels for UI rendering.
func FormatPanelsForUIHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[FormatPanelsForUIInput, FormatPanelsForUIOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input FormatPanelsForUIInput) (*mcp.CallToolResult, FormatPanelsForUIOutput, error) {
		slog.Info("FormatPanelsForUIHandler called")

		var panelIDs []string
		if input.PanelIDs != "" {
			for _, part := range splitByComma(input.PanelIDs) {
				if part != "" {
					panelIDs = append(panelIDs, part)
				}
			}
		}

		spec, err := k8s.GetDashboard(ctx, input.Namespace, input.Name)
		if err != nil {
			return nil, FormatPanelsForUIOutput{}, fmt.Errorf("failed to get Dashboard: %w", err)
		}

		panels := perses.ExtractPanels(input.Name, input.Namespace, spec, true, panelIDs)

		widgets := convertPanelsToDashboardWidgets(panels)

		slog.Info("FormatPanelsForUIHandler executed successfully",
			"dashboard", input.Name,
			"namespace", input.Namespace,
			"requestedPanels", len(panelIDs),
			"formattedWidgets", len(widgets))

		return nil, FormatPanelsForUIOutput{Widgets: widgets}, nil
	}
}

func splitByComma(s string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			trimmed := trimWhitespace(current)
			if trimmed != "" {
				parts = append(parts, trimmed)
			}
			current = ""
		} else {
			current += string(s[i])
		}
	}
	trimmed := trimWhitespace(current)
	if trimmed != "" {
		parts = append(parts, trimmed)
	}
	return parts
}

func trimWhitespace(s string) string {
	start := 0
	end := len(s)

	for start < end && isWhitespace(s[start]) {
		start++
	}

	for end > start && isWhitespace(s[end-1]) {
		end--
	}

	return s[start:end]
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// convertPanelsToDashboardWidgets converts DashboardPanel objects to DashboardWidget format expected by UI.
func convertPanelsToDashboardWidgets(panels []*perses.DashboardPanel) []perses.DashboardWidget {
	widgets := make([]perses.DashboardWidget, 0, len(panels))

	for _, panel := range panels {
		step := panel.Step
		if step == "" {
			step = "15s"
		}
		duration := panel.Duration
		if duration == "" {
			duration = "1h"
		}

		breakpoint := "lg"
		if panel.Position != nil {
			breakpoint = inferBreakpointFromWidth(panel.Position.W)
		}

		widget := perses.DashboardWidget{
			ID:            panel.ID,
			ComponentType: mapChartTypeToComponent(panel.ChartType),
			Breakpoint:    breakpoint,
			Props: perses.DashboardWidgetProps{
				Query:    panel.Query,
				Duration: duration,
				Start:    panel.Start,
				End:      panel.End,
				Step:     step,
			},
		}

		if panel.Position != nil {
			widget.Position = *panel.Position
		}

		widgets = append(widgets, widget)
	}

	return widgets
}

func mapChartTypeToComponent(chartType string) string {
	switch chartType {
	case "TimeSeriesChart":
		return "PersesTimeSeries"
	case "PieChart", "StatChart":
		return "PersesPieChart"
	case "Table":
		return "PersesTable"
	default:
		return "PersesTimeSeries"
	}
}

func inferBreakpointFromWidth(width int) string {
	switch {
	case width >= 18:
		return "xl"
	case width >= 12:
		return "lg"
	case width >= 6:
		return "md"
	default:
		return "sm"
	}
}
