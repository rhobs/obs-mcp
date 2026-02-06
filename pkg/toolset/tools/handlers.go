package tools

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"

	"github.com/rhobs/obs-mcp/pkg/handlers"
)

// Helper function to get string argument with default
func getStringArg(params api.ToolHandlerParams, key, defaultValue string) string {
	if val, ok := params.GetArguments()[key].(string); ok && val != "" {
		return val
	}
	return defaultValue
}

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

	input := handlers.InstantQueryInput{
		Query: getStringArg(params, "query", ""),
		Time:  getStringArg(params, "time", ""),
	}

	return handlers.ExecuteInstantQueryHandler(params.Context, promClient, input).ToToolsetResult()
}

// ExecuteRangeQueryHandler handles the execution of Prometheus range queries.
func ExecuteRangeQueryHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	input := handlers.RangeQueryInput{
		Query:    getStringArg(params, "query", ""),
		Step:     getStringArg(params, "step", ""),
		Start:    getStringArg(params, "start", ""),
		End:      getStringArg(params, "end", ""),
		Duration: getStringArg(params, "duration", ""),
	}

	return handlers.ExecuteRangeQueryHandler(params.Context, promClient, input).ToToolsetResult()
}

// GetLabelNamesHandler handles the retrieval of label names.
func GetLabelNamesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	input := handlers.LabelNamesInput{
		Metric: getStringArg(params, "metric", ""),
		Start:  getStringArg(params, "start", ""),
		End:    getStringArg(params, "end", ""),
	}

	return handlers.GetLabelNamesHandler(params.Context, promClient, input).ToToolsetResult()
}

// GetLabelValuesHandler handles the retrieval of label values.
func GetLabelValuesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	input := handlers.LabelValuesInput{
		Label:  getStringArg(params, "label", ""),
		Metric: getStringArg(params, "metric", ""),
		Start:  getStringArg(params, "start", ""),
		End:    getStringArg(params, "end", ""),
	}

	return handlers.GetLabelValuesHandler(params.Context, promClient, input).ToToolsetResult()
}

// GetSeriesHandler handles the retrieval of time series.
func GetSeriesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	promClient, err := getPromClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Prometheus client: %w", err)), nil
	}

	input := handlers.SeriesInput{
		Matches: getStringArg(params, "matches", ""),
		Start:   getStringArg(params, "start", ""),
		End:     getStringArg(params, "end", ""),
	}

	return handlers.GetSeriesHandler(params.Context, promClient, input).ToToolsetResult()
}

// GetAlertsHandler handles the retrieval of alerts from Alertmanager.
func GetAlertsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	amClient, err := getAlertmanagerClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Alertmanager client: %w", err)), nil
	}

	// Parse boolean parameters
	var input handlers.AlertsInput
	if activeVal, ok := params.GetArguments()["active"].(bool); ok {
		input.Active = &activeVal
	}
	if silencedVal, ok := params.GetArguments()["silenced"].(bool); ok {
		input.Silenced = &silencedVal
	}
	if inhibitedVal, ok := params.GetArguments()["inhibited"].(bool); ok {
		input.Inhibited = &inhibitedVal
	}
	if unprocessedVal, ok := params.GetArguments()["unprocessed"].(bool); ok {
		input.Unprocessed = &unprocessedVal
	}
	input.Filter = getStringArg(params, "filter", "")
	input.Receiver = getStringArg(params, "receiver", "")

	return handlers.GetAlertsHandler(params.Context, amClient, input).ToToolsetResult()
}

// GetSilencesHandler handles the retrieval of silences from Alertmanager.
func GetSilencesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	amClient, err := getAlertmanagerClient(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create Alertmanager client: %w", err)), nil
	}

	input := handlers.SilencesInput{
		Filter: getStringArg(params, "filter", ""),
	}

	return handlers.GetSilencesHandler(params.Context, amClient, input).ToToolsetResult()
}
