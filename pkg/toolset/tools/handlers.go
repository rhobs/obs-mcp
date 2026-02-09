package tools

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"

	"github.com/rhobs/obs-mcp/pkg/handlers"
	"github.com/rhobs/obs-mcp/pkg/tooldef"
)

// ListMetricsHandler handles the listing of available Prometheus metrics.
func ListMetricsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	return handlers.ListMetricsHandler(params.Context, promClient).ToToolsetResult()
}

// ExecuteInstantQueryHandler handles the execution of Prometheus instant queries.
func ExecuteInstantQueryHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	return handlers.ExecuteInstantQueryHandler(params.Context, promClient, tooldef.BuildInstantQueryInput(params.GetArguments())).ToToolsetResult()
}

// ExecuteRangeQueryHandler handles the execution of Prometheus range queries.
func ExecuteRangeQueryHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	return handlers.ExecuteRangeQueryHandler(params.Context, promClient, tooldef.BuildRangeQueryInput(params.GetArguments())).ToToolsetResult()
}

// GetLabelNamesHandler handles the retrieval of label names.
func GetLabelNamesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	return handlers.GetLabelNamesHandler(params.Context, promClient, tooldef.BuildLabelNamesInput(params.GetArguments())).ToToolsetResult()
}

// GetLabelValuesHandler handles the retrieval of label values.
func GetLabelValuesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	return handlers.GetLabelValuesHandler(params.Context, promClient, tooldef.BuildLabelValuesInput(params.GetArguments())).ToToolsetResult()
}

// GetSeriesHandler handles the retrieval of time series.
func GetSeriesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	return handlers.GetSeriesHandler(params.Context, promClient, tooldef.BuildSeriesInput(params.GetArguments())).ToToolsetResult()
}

// GetAlertsHandler handles the retrieval of alerts from Alertmanager.
func GetAlertsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	amClient, err := getAlertmanagerClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Alertmanager client: %w", err)), nil
	}

	return handlers.GetAlertsHandler(params.Context, amClient, tooldef.BuildAlertsInput(params.GetArguments())).ToToolsetResult()
}

// GetSilencesHandler handles the retrieval of silences from Alertmanager.
func GetSilencesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	amClient, err := getAlertmanagerClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Alertmanager client: %w", err)), nil
	}

	return handlers.GetSilencesHandler(params.Context, amClient, tooldef.BuildSilencesInput(params.GetArguments())).ToToolsetResult()
}
