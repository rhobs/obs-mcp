package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/rhobs/obs-mcp/pkg/tools"
)

// ListMetricsHandler handles the listing of available Prometheus metrics.
func ListMetricsHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return tools.ListMetricsHandler(ctx, promClient, tools.BuildListMetricsInput(req.GetArguments())).ToMCPResult()
	}
}

// ExecuteRangeQueryHandler handles the execution of Prometheus range queries.
func ExecuteRangeQueryHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return tools.ExecuteRangeQueryHandler(ctx, promClient, tools.BuildRangeQueryInput(req.GetArguments())).ToMCPResult()
	}
}

// ExecuteInstantQueryHandler handles the execution of Prometheus instant queries.
func ExecuteInstantQueryHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return tools.ExecuteInstantQueryHandler(ctx, promClient, tools.BuildInstantQueryInput(req.GetArguments())).ToMCPResult()
	}
}

// GetLabelNamesHandler handles the retrieval of label names.
func GetLabelNamesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return tools.GetLabelNamesHandler(ctx, promClient, tools.BuildLabelNamesInput(req.GetArguments())).ToMCPResult()
	}
}

// GetLabelValuesHandler handles the retrieval of label values.
func GetLabelValuesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return tools.GetLabelValuesHandler(ctx, promClient, tools.BuildLabelValuesInput(req.GetArguments())).ToMCPResult()
	}
}

// GetSeriesHandler handles the retrieval of time series.
func GetSeriesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return tools.GetSeriesHandler(ctx, promClient, tools.BuildSeriesInput(req.GetArguments())).ToMCPResult()
	}
}

// GetAlertsHandler handles the retrieval of alerts from Alertmanager.
func GetAlertsHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		amClient, err := getAlertmanagerClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Alertmanager client: %s", err.Error())), nil
		}

		return tools.GetAlertsHandler(ctx, amClient, tools.BuildAlertsInput(req.GetArguments())).ToMCPResult()
	}
}

// GetSilencesHandler handles the retrieval of silences from Alertmanager.
func GetSilencesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		amClient, err := getAlertmanagerClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Alertmanager client: %s", err.Error())), nil
		}

		return tools.GetSilencesHandler(ctx, amClient, tools.BuildSilencesInput(req.GetArguments())).ToMCPResult()
	}
}
