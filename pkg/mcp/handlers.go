package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	promModel "github.com/prometheus/common/model"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/prometheus/common/model"
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
			return mcp.NewToolResultError("query parameter is required and must be a string"), nil
		}

		// Get required step parameter
		step, err := req.RequireString("step")
		if err != nil {
			return mcp.NewToolResultError("step parameter is required and must be a string"), nil
		}

		// Parse step duration
		stepDuration, err := promModel.ParseDuration(step)
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

			duration, err := promModel.ParseDuration(durationStr)
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
				values := make([][]interface{}, len(series.Values))
				for j, sample := range series.Values {
					values[j] = []interface{}{float64(sample.Timestamp) / 1000, sample.Value.String()}
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
