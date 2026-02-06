package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/rhobs/obs-mcp/pkg/handlers"
)

// ListMetricsHandler handles the listing of available Prometheus metrics.
func ListMetricsHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return handlers.ListMetricsHandler(ctx, promClient).ToMCPResult()
	}
}

// ExecuteRangeQueryHandler handles the execution of Prometheus range queries.
func ExecuteRangeQueryHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return handlers.ExecuteRangeQueryHandler(ctx, promClient, handlers.RangeQueryInput{
			Query:    req.GetString("query", ""),
			Step:     req.GetString("step", ""),
			Start:    req.GetString("start", ""),
			End:      req.GetString("end", ""),
			Duration: req.GetString("duration", ""),
		}).ToMCPResult()
	}
}

// ExecuteInstantQueryHandler handles the execution of Prometheus instant queries.
func ExecuteInstantQueryHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return handlers.ExecuteInstantQueryHandler(ctx, promClient, handlers.InstantQueryInput{
			Query: req.GetString("query", ""),
			Time:  req.GetString("time", ""),
		}).ToMCPResult()
	}
}

// GetLabelNamesHandler handles the retrieval of label names.
func GetLabelNamesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return handlers.GetLabelNamesHandler(ctx, promClient, handlers.LabelNamesInput{
			Metric: req.GetString("metric", ""),
			Start:  req.GetString("start", ""),
			End:    req.GetString("end", ""),
		}).ToMCPResult()
	}
}

// GetLabelValuesHandler handles the retrieval of label values.
func GetLabelValuesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return handlers.GetLabelValuesHandler(ctx, promClient, handlers.LabelValuesInput{
			Label:  req.GetString("label", ""),
			Metric: req.GetString("metric", ""),
			Start:  req.GetString("start", ""),
			End:    req.GetString("end", ""),
		}).ToMCPResult()
	}
}

// GetSeriesHandler handles the retrieval of time series.
func GetSeriesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Prometheus client: %s", err.Error())), nil
		}

		return handlers.GetSeriesHandler(ctx, promClient, handlers.SeriesInput{
			Matches: req.GetString("matches", ""),
			Start:   req.GetString("start", ""),
			End:     req.GetString("end", ""),
		}).ToMCPResult()
	}
}

// GetAlertsHandler handles the retrieval of alerts from Alertmanager.
func GetAlertsHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		amClient, err := getAlertmanagerClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Alertmanager client: %s", err.Error())), nil
		}

		// Parse MCP parameters into input struct
		var input handlers.AlertsInput
		if req.Params.Arguments != nil {
			if args, ok := req.Params.Arguments.(map[string]any); ok {
				if activeVal, ok := args["active"].(bool); ok {
					input.Active = &activeVal
				}
				if silencedVal, ok := args["silenced"].(bool); ok {
					input.Silenced = &silencedVal
				}
				if inhibitedVal, ok := args["inhibited"].(bool); ok {
					input.Inhibited = &inhibitedVal
				}
				if unprocessedVal, ok := args["unprocessed"].(bool); ok {
					input.Unprocessed = &unprocessedVal
				}
			}
		}
		input.Filter = req.GetString("filter", "")
		input.Receiver = req.GetString("receiver", "")

		return handlers.GetAlertsHandler(ctx, amClient, input).ToMCPResult()
	}
}

// GetSilencesHandler handles the retrieval of silences from Alertmanager.
func GetSilencesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		amClient, err := getAlertmanagerClient(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create Alertmanager client: %s", err.Error())), nil
		}

		return handlers.GetSilencesHandler(ctx, amClient, handlers.SilencesInput{
			Filter: req.GetString("filter", ""),
		}).ToMCPResult()
	}
}
