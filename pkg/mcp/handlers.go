package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
