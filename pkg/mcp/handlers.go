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
// Returns all Dashboard resources to provide maximum context for LLM selection.
func DashboardsHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[struct{}, tools.DashboardsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, tools.DashboardsOutput, error) {
		slog.Info("DashboardsHandler called")

		// TODO: add a label selectors flag when more dashboards start annotating themselves?
		dashboards, err := k8s.ListDashboards(ctx, "", "")
		if err != nil {
			return nil, tools.DashboardsOutput{}, fmt.Errorf("failed to list dashboards: %w", err)
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

		return nil, tools.DashboardsOutput{Dashboards: dashboardInfos}, nil
	}
}

// GetDashboardHandler handles getting a specific dashboard by name and namespace.
func GetDashboardHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[tools.DashboardInput, tools.GetDashboardOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.DashboardInput) (*mcp.CallToolResult, tools.GetDashboardOutput, error) {
		slog.Info("GetDashboardHandler called")
		slog.Debug("GetDashboardHandler params", "input", input)

		spec, err := k8s.GetDashboard(ctx, input.Namespace, input.Name)
		if err != nil {
			return nil, tools.GetDashboardOutput{}, fmt.Errorf("failed to get Dashboard: %w", err)
		}

		slog.Info("GetDashboardHandler executed successfully", "name", input.Name, "namespace", input.Namespace)
		slog.Debug("GetDashboardHandler spec", "spec", spec)

		return nil, tools.GetDashboardOutput{
			Name:      input.Name,
			Namespace: input.Namespace,
			Spec:      spec,
		}, nil
	}
}

// GetDashboardPanelsHandler handles getting panel metadata from a dashboard for LLM selection.
func GetDashboardPanelsHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[tools.DashboardPanelsInput, tools.GetDashboardPanelsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.DashboardPanelsInput) (*mcp.CallToolResult, tools.GetDashboardPanelsOutput, error) {
		slog.Info("GetDashboardPanelsHandler called")
		slog.Debug("GetDashboardPanelsHandler params", "input", input)

		// Optional panel IDs filter
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
			return nil, tools.GetDashboardPanelsOutput{}, fmt.Errorf("failed to get dashboard: %w", err)
		}

		// Extract panel metadata (with optional filtering)
		panels := perses.ExtractPanels(input.Name, input.Namespace, spec, panelIDs)

		duration := "1h"
		if d, ok := spec["duration"].(string); ok {
			duration = d
		}

		slog.Info("GetDashboardPanelsHandler executed successfully",
			"name", input.Name,
			"namespace", input.Namespace,
			"requested", len(panelIDs),
			"returned", len(panels))

		return nil, tools.GetDashboardPanelsOutput{
			Name:      input.Name,
			Namespace: input.Namespace,
			Duration:  duration,
			Panels:    panels,
		}, nil
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
