package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/prometheus/common/model"
	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/perses"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

// errorResult is a helper to log and return an error result.
func errorResult(msg string) (*mcp.CallToolResult, error) {
	slog.Info("Query execution error: " + msg)
	return mcp.NewToolResultError(msg), nil
}

// ListMetricsHandler handles the listing of available Prometheus metrics.
func ListMetricsHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("ListMetricsHandler called")
		slog.Debug("ListMetricsHandler params", "params", req.Params)
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
		}

		metrics, err := promClient.ListMetrics(ctx)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to list metrics: %s", err.Error()))
		}

		slog.Info("ListMetricsHandler executed successfully", "resultLength", len(metrics))
		slog.Debug("ListMetricsHandler results", "results", metrics)

		output := ListMetricsOutput{Metrics: metrics}
		result, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal metrics: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(result)), nil
	}
}

// ExecuteRangeQueryHandler handles the execution of Prometheus range queries.
func ExecuteRangeQueryHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("ExecuteRangeQueryHandler called")
		slog.Debug("ExecuteRangeQueryHandler params", "params", req.Params)

		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		// Get required query parameter
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError("query parameter is required and must be a string"), nil //nolint:nilerr // MCP pattern: error in result, not return
		}

		// Get required step parameter
		step, err := req.RequireString("step")
		if err != nil {
			return mcp.NewToolResultError("step parameter is required and must be a string"), nil //nolint:nilerr // MCP pattern: error in result, not return
		}

		// Parse step duration
		stepDuration, err := model.ParseDuration(step)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid step format: %s", err.Error())), nil
		}

		// Get optional parameters
		startStr := req.GetString("start", "")
		endStr := req.GetString("end", "")
		durationStr := req.GetString("duration", "")

		if endStr == "NOW" {
			endStr = ""
		}

		// Validate parameter combinations
		if startStr != "" && endStr != "" && durationStr != "" {
			return errorResult("cannot specify both start/end and duration parameters")
		}

		if (startStr != "" && endStr == "") || (startStr == "" && endStr != "") {
			return errorResult("both start and end must be provided together")
		}

		var startTime, endTime time.Time

		// Handle duration-based query (default to 1h if nothing specified)
		if durationStr != "" || (startStr == "" && endStr == "") {
			if durationStr == "" {
				durationStr = "1h"
			}

			duration, err := model.ParseDuration(durationStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid duration format: %s", err.Error()))
			}

			endTime = time.Now()
			startTime = endTime.Add(-time.Duration(duration))
		} else {
			// Handle explicit start/end times
			startTime, err = prometheus.ParseTimestamp(startStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid start time format: %s", err.Error()))
			}

			endTime, err = prometheus.ParseTimestamp(endStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid end time format: %s", err.Error()))
			}
		}

		// Execute the range query
		result, err := promClient.ExecuteRangeQuery(ctx, query, startTime, endTime, time.Duration(stepDuration))
		if err != nil {
			return errorResult(fmt.Sprintf("failed to execute range query: %s", err.Error()))
		}

		// Convert to structured output
		output := RangeQueryOutput{
			ResultType: fmt.Sprintf("%v", result["resultType"]),
		}

		resMatrix, ok := result["result"].(model.Matrix)
		if ok {
			slog.Info("ExecuteRangeQueryHandler executed successfully", "resultLength", resMatrix.Len())
			slog.Debug("ExecuteRangeQueryHandler results", "results", resMatrix)

			output.Result = make([]SeriesResult, len(resMatrix))
			for i, series := range resMatrix {
				labels := make(map[string]string)
				for k, v := range series.Metric {
					labels[string(k)] = string(v)
				}
				values := make([][]any, len(series.Values))
				for j, sample := range series.Values {
					values[j] = []any{float64(sample.Timestamp) / 1000, sample.Value.String()}
				}
				output.Result[i] = SeriesResult{
					Metric: labels,
					Values: values,
				}
			}
		} else {
			slog.Info("ExecuteRangeQueryHandler executed successfully (unknown format)", "result", result)
		}

		if warnings, ok := result["warnings"].([]string); ok {
			output.Warnings = warnings
		}

		// Convert to JSON for fallback text
		jsonResult, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal result: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(jsonResult)), nil
	}
}

// DashboardsHandler handles returning all dashboards from the cluster.
// Returns all Dashboard resources to provide maximum context for LLM selection.
func DashboardsHandler(_ ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("DashboardsHandler called")

		// TODO: add a label selectors flag when more dashboards start annotating themselves?
		dashboards, err := k8s.ListDashboards(ctx, "", "")
		if err != nil {
			return errorResult(fmt.Sprintf("failed to list dashboards: %s", err.Error()))
		}

		// Convert to DashboardInfo
		dashboardInfos := make([]perses.DashboardInfo, 0, len(dashboards))
		for _, dashboard := range dashboards {
			info := perses.DashboardInfo{
				Name:      dashboard.Name,
				Namespace: dashboard.Namespace,
				Labels:    dashboard.Labels,
			}

			// Extract description from annotation
			if dashboard.Annotations != nil {
				if desc, ok := dashboard.Annotations[k8s.MCPHelpAnnotation /* TODO: currently no such annotation is curated across openshift */]; ok {
					info.Description = desc
				}
			}

			dashboardInfos = append(dashboardInfos, info)
		}

		slog.Info("DashboardsHandler executed successfully", "dashboardCount", len(dashboardInfos))
		slog.Debug("DashboardsHandler results", "results", dashboardInfos)

		output := DashboardsOutput{Dashboards: dashboardInfos}

		result, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal dashboards: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(result)), nil
	}
}

// GetDashboardHandler handles getting a specific dashboard by name and namespace.
func GetDashboardHandler(_ ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("GetDashboardHandler called")
		slog.Debug("GetDashboardHandler params", "params", req.Params)

		name, err := req.RequireString("name")
		if err != nil {
			return errorResult("name parameter is required and must be a string")
		}

		namespace, err := req.RequireString("namespace")
		if err != nil {
			return errorResult("namespace parameter is required and must be a string")
		}

		dashboardName, dashboardNamespace, spec, err := k8s.GetDashboard(ctx, namespace, name)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get Dashboard: %s", err.Error()))
		}

		slog.Info("GetDashboardHandler executed successfully", "name", dashboardName, "namespace", dashboardNamespace)
		slog.Debug("GetDashboardHandler spec", "spec", spec)

		output := GetDashboardOutput{
			Name:      dashboardName,
			Namespace: dashboardNamespace,
			Spec:      spec,
		}

		result, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal dashboard: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(result)), nil
	}
}

// GetDashboardPanelsHandler handles getting panel metadata from a dashboard for LLM selection.
func GetDashboardPanelsHandler(_ ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("GetDashboardPanelsHandler called")
		slog.Debug("GetDashboardPanelsHandler params", "params", req.Params)

		name, err := req.RequireString("name")
		if err != nil {
			return errorResult("name parameter is required and must be a string")
		}

		namespace, err := req.RequireString("namespace")
		if err != nil {
			return errorResult("namespace parameter is required and must be a string")
		}

		// Optional panel IDs filter
		panelIDsStr := req.GetString("panel_ids", "")
		var panelIDs []string
		if panelIDsStr != "" {
			for _, part := range splitByComma(panelIDsStr) {
				if part != "" {
					panelIDs = append(panelIDs, part)
				}
			}
		}

		dashboardName, dashboardNamespace, spec, err := k8s.GetDashboard(ctx, namespace, name)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get dashboard: %s", err.Error()))
		}

		// Extract panel metadata (with optional filtering)
		panels, err := perses.ExtractPanels(dashboardName, dashboardNamespace, spec, false, panelIDs)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to extract panels: %s", err.Error()))
		}

		duration := "1h"
		if d, ok := spec["duration"].(string); ok {
			duration = d
		}

		slog.Info("GetDashboardPanelsHandler executed successfully",
			"name", dashboardName,
			"namespace", dashboardNamespace,
			"requested", len(panelIDs),
			"returned", len(panels))

		output := GetDashboardPanelsOutput{
			Name:      dashboardName,
			Namespace: dashboardNamespace,
			Duration:  duration,
			Panels:    panels,
		}

		result, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal panels: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(result)), nil
	}
}

// FormatPanelsForUIHandler handles formatting selected panels for UI rendering.
func FormatPanelsForUIHandler(_ ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("FormatPanelsForUIHandler called")
		slog.Debug("FormatPanelsForUIHandler params", "params", req.Params)

		name, err := req.RequireString("name")
		if err != nil {
			return errorResult("name parameter is required and must be a string")
		}

		namespace, err := req.RequireString("namespace")
		if err != nil {
			return errorResult("namespace parameter is required and must be a string")
		}

		panelIDsStr, err := req.RequireString("panel_ids")
		if err != nil {
			return errorResult("panel_ids parameter is required and must be a string")
		}

		// Parse comma-separated panel IDs
		var panelIDs []string
		if panelIDsStr != "" {
			for _, part := range splitByComma(panelIDsStr) {
				if part != "" {
					panelIDs = append(panelIDs, part)
				}
			}
		}

		_, _, spec, err := k8s.GetDashboard(ctx, namespace, name)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get Dashboard: %s", err.Error()))
		}

		// Extract full panel details for UI
		panels, err := perses.ExtractPanels(name, namespace, spec, true, panelIDs)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to extract panels: %s", err.Error()))
		}

		// Convert panels to DashboardWidget format
		widgets := convertPanelsToDashboardWidgets(panels)

		slog.Info("FormatPanelsForUIHandler executed successfully",
			"dashboard", name,
			"namespace", namespace,
			"requestedPanels", len(panelIDs),
			"formattedWidgets", len(widgets))

		output := FormatPanelsForUIOutput{
			Widgets: widgets,
		}

		result, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal panels: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(result)), nil
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
func convertPanelsToDashboardWidgets(panels []perses.DashboardPanel) []perses.DashboardWidget {
	widgets := make([]perses.DashboardWidget, 0, len(panels))

	for _, panel := range panels {
		// Set defaults for required fields
		step := panel.Step
		if step == "" {
			step = "15s" // default step for Prometheus queries
		}
		duration := panel.Duration
		if duration == "" {
			duration = "1h" // default duration
		}

		// Infer breakpoint from panel width if available
		breakpoint := "lg" // default
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

		// Add position if available
		if panel.Position != nil {
			widget.Position = *panel.Position
		}

		widgets = append(widgets, widget)
	}

	return widgets
}

// mapChartTypeToComponent maps Perses chart types to component names
func mapChartTypeToComponent(chartType string) string {
	switch chartType {
	case "TimeSeriesChart":
		return "PersesTimeSeries"
	case "PieChart", "StatChart":
		return "PersesPieChart"
	case "Table":
		return "PersesTable"
	default:
		return "PersesTimeSeries" // default fallback
	}
}

// inferBreakpointFromWidth maps panel width to responsive breakpoint.
// Perses uses a 24-column grid, so we infer breakpoints based on width.
func inferBreakpointFromWidth(width int) string {
	if width >= 18 {
		return "xl"
	} else if width >= 12 {
		return "lg"
	} else if width >= 6 {
		return "md"
	}
	return "sm"
}
