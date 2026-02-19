package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/rhobs/obs-mcp/pkg/alertmanager"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	"github.com/rhobs/obs-mcp/pkg/resultutil"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

// createPrometheusToolHandler creates an MCP handler that wraps a Prometheus tool,
// handling client creation and error formatting.
func createPrometheusToolHandler[TInput any](
	opts ObsMCPOptions,
	buildInput func(map[string]any) TInput,
	handler func(context.Context, prometheus.Loader, TInput) *resultutil.Result,
) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %v", err)), nil
		}
		return handler(ctx, promClient, buildInput(req.GetArguments())).ToMCPResult()
	}
}

// createAlertmanagerToolHandler creates an MCP handler that wraps an Alertmanager tool,
// handling client creation and error formatting.
func createAlertmanagerToolHandler[TInput any](
	opts ObsMCPOptions,
	buildInput func(map[string]any) TInput,
	handler func(context.Context, alertmanager.Loader, TInput) *resultutil.Result,
) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		amClient, err := getAlertmanagerClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Alertmanager client: %v", err)), nil
		}
		return handler(ctx, amClient, buildInput(req.GetArguments())).ToMCPResult()
	}
}

// ListMetricsHandler handles the listing of available Prometheus metrics.
func ListMetricsHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return createPrometheusToolHandler(opts, tools.BuildListMetricsInput, tools.ListMetricsHandler)
}

// ExecuteRangeQueryHandler handles the execution of Prometheus range queries.
func ExecuteRangeQueryHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return createPrometheusToolHandler(opts, tools.BuildRangeQueryInput, tools.ExecuteRangeQueryHandler)
}

// ExecuteInstantQueryHandler handles the execution of Prometheus instant queries.
func ExecuteInstantQueryHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return createPrometheusToolHandler(opts, tools.BuildInstantQueryInput, tools.ExecuteInstantQueryHandler)
}

// GetLabelNamesHandler handles the retrieval of label names.
func GetLabelNamesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return createPrometheusToolHandler(opts, tools.BuildLabelNamesInput, tools.GetLabelNamesHandler)
}

// GetLabelValuesHandler handles the retrieval of label values.
func GetLabelValuesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return createPrometheusToolHandler(opts, tools.BuildLabelValuesInput, tools.GetLabelValuesHandler)
}

// GetSeriesHandler handles the retrieval of time series.
func GetSeriesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return createPrometheusToolHandler(opts, tools.BuildSeriesInput, tools.GetSeriesHandler)
}

// GetAlertsHandler handles the retrieval of alerts from Alertmanager.
func GetAlertsHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return createAlertmanagerToolHandler(opts, tools.BuildAlertsInput, tools.GetAlertsHandler)
}

// GetSilencesHandler handles the retrieval of silences from Alertmanager.
func GetSilencesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return createAlertmanagerToolHandler(opts, tools.BuildSilencesInput, tools.GetSilencesHandler)
}
