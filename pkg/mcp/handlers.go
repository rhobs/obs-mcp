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

// ListPersesDashboardsHandler handles listing PersesDashboard CRD objects from the cluster.
func ListPersesDashboardsHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[ListPersesDashboardsInput, ListPersesDashboardsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ListPersesDashboardsInput) (*mcp.CallToolResult, ListPersesDashboardsOutput, error) {
		slog.Info("ListPersesDashboardsHandler called")

		dashboards, err := k8s.ListPersesDashboards(ctx, input.Namespace, input.LabelSelector)
		if err != nil {
			return nil, ListPersesDashboardsOutput{}, fmt.Errorf("failed to list PersesDashboards: %w", err)
		}

		slog.Info("ListPersesDashboardsHandler executed successfully", "resultLength", len(dashboards))

		dashboardInfos := make([]perses.PersesDashboardInfo, len(dashboards))
		for i, db := range dashboards {
			dashboardInfo := perses.PersesDashboardInfo{
				Name:      db.Name,
				Namespace: db.Namespace,
				Labels:    db.GetLabels(),
			}

			// Extract MCP help description from annotation if present
			if annotations := db.GetAnnotations(); annotations != nil {
				if description, ok := annotations[k8s.PersesMCPHelpAnnotation]; ok {
					dashboardInfo.Description = description
				}
			}

			dashboardInfos[i] = dashboardInfo
		}

		return nil, ListPersesDashboardsOutput{Dashboards: dashboardInfos}, nil
	}
}

// OOTBPersesDashboardsHandler handles returning pre-configured out-of-the-box dashboards.
func OOTBPersesDashboardsHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[struct{}, OOTBPersesDashboardsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, OOTBPersesDashboardsOutput, error) {
		slog.Info("OOTBPersesDashboardsHandler called")
		slog.Info("OOTBPersesDashboardsHandler executed successfully", "resultLength", len(opts.OOTBDashboards))
		return nil, OOTBPersesDashboardsOutput{Dashboards: opts.OOTBDashboards}, nil
	}
}

// GetPersesDashboardHandler handles getting a specific PersesDashboard by name and namespace.
func GetPersesDashboardHandler(opts ObsMCPOptions) mcp.ToolHandlerFor[GetPersesDashboardInput, GetPersesDashboardOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input GetPersesDashboardInput) (*mcp.CallToolResult, GetPersesDashboardOutput, error) {
		slog.Info("GetPersesDashboardHandler called")

		dashboardName, dashboardNamespace, spec, err := k8s.GetPersesDashboard(ctx, input.Namespace, input.Name)
		if err != nil {
			return nil, GetPersesDashboardOutput{}, fmt.Errorf("failed to get PersesDashboard: %w", err)
		}

		slog.Info("GetPersesDashboardHandler executed successfully", "name", dashboardName, "namespace", dashboardNamespace)

		return nil, GetPersesDashboardOutput{
			Name:      dashboardName,
			Namespace: dashboardNamespace,
			Spec:      spec,
		}, nil
	}
}
