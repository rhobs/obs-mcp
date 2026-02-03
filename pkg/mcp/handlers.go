package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/prometheus/common/model"

	"github.com/rhobs/obs-mcp/pkg/alertmanager"
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

// ExecuteInstantQueryHandler handles the execution of Prometheus instant queries.
func ExecuteInstantQueryHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("ExecuteInstantQueryHandler called")
		slog.Debug("ExecuteInstantQueryHandler params", "params", req.Params)

		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
		}

		// Get required query parameter
		query, err := req.RequireString("query")
		if err != nil {
			return errorResult("query parameter is required and must be a string")
		}

		// Get optional time parameter
		timeStr := req.GetString("time", "")

		var queryTime time.Time
		if timeStr == "" {
			queryTime = time.Now()
		} else {
			queryTime, err = prometheus.ParseTimestamp(timeStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid time format: %s", err.Error()))
			}
		}

		// Execute the instant query
		result, err := promClient.ExecuteInstantQuery(ctx, query, queryTime)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to execute instant query: %s", err.Error()))
		}

		// Convert to structured output
		output := InstantQueryOutput{
			ResultType: fmt.Sprintf("%v", result["resultType"]),
		}

		resVector, ok := result["result"].(model.Vector)
		if ok {
			slog.Info("ExecuteInstantQueryHandler executed successfully", "resultLength", len(resVector))
			slog.Debug("ExecuteInstantQueryHandler results", "results", resVector)

			output.Result = make([]InstantResult, len(resVector))
			for i, sample := range resVector {
				labels := make(map[string]string)
				for k, v := range sample.Metric {
					labels[string(k)] = string(v)
				}
				output.Result[i] = InstantResult{
					Metric: labels,
					Value:  []any{float64(sample.Timestamp) / 1000, sample.Value.String()},
				}
			}
		} else {
			slog.Info("ExecuteInstantQueryHandler executed successfully (unknown format)", "result", result)
		}

		if warnings, ok := result["warnings"].([]string); ok {
			output.Warnings = warnings
		}

		jsonResult, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal result: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(jsonResult)), nil
	}
}

// GetLabelNamesHandler handles the retrieval of label names.
func GetLabelNamesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("GetLabelNamesHandler called")
		slog.Debug("GetLabelNamesHandler params", "params", req.Params)

		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
		}

		// Get optional parameters
		metric := req.GetString("metric", "")
		startStr := req.GetString("start", "")
		endStr := req.GetString("end", "")

		// Default to last hour if not specified
		var startTime, endTime time.Time
		if startStr == "" && endStr == "" {
			endTime = time.Now()
			startTime = endTime.Add(-prometheus.ListMetricsTimeRange)
		} else {
			if startStr != "" {
				startTime, err = prometheus.ParseTimestamp(startStr)
				if err != nil {
					return errorResult(fmt.Sprintf("invalid start time format: %s", err.Error()))
				}
			}
			if endStr != "" {
				endTime, err = prometheus.ParseTimestamp(endStr)
				if err != nil {
					return errorResult(fmt.Sprintf("invalid end time format: %s", err.Error()))
				}
			}
		}

		// Get label names
		labels, err := promClient.GetLabelNames(ctx, metric, startTime, endTime)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get label names: %s", err.Error()))
		}

		output := LabelNamesOutput{Labels: labels}

		slog.Info("GetLabelNamesHandler executed successfully", "labelCount", len(labels))
		slog.Debug("GetLabelNamesHandler results", "results", labels)

		jsonResult, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal label names: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(jsonResult)), nil
	}
}

// GetLabelValuesHandler handles the retrieval of label values.
func GetLabelValuesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("GetLabelValuesHandler called")
		slog.Debug("GetLabelValuesHandler params", "params", req.Params)

		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
		}

		// Get required label parameter
		label, err := req.RequireString("label")
		if err != nil {
			return errorResult("label parameter is required and must be a string")
		}

		// Get optional parameters
		metric := req.GetString("metric", "")
		startStr := req.GetString("start", "")
		endStr := req.GetString("end", "")

		// Default to last hour if not specified
		var startTime, endTime time.Time
		if startStr == "" && endStr == "" {
			endTime = time.Now()
			startTime = endTime.Add(-prometheus.ListMetricsTimeRange)
		} else {
			if startStr != "" {
				startTime, err = prometheus.ParseTimestamp(startStr)
				if err != nil {
					return errorResult(fmt.Sprintf("invalid start time format: %s", err.Error()))
				}
			}
			if endStr != "" {
				endTime, err = prometheus.ParseTimestamp(endStr)
				if err != nil {
					return errorResult(fmt.Sprintf("invalid end time format: %s", err.Error()))
				}
			}
		}

		// Get label values
		values, err := promClient.GetLabelValues(ctx, label, metric, startTime, endTime)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get label values: %s", err.Error()))
		}

		output := LabelValuesOutput{Values: values}

		slog.Info("GetLabelValuesHandler executed successfully", "valueCount", len(values))
		slog.Debug("GetLabelValuesHandler results", "results", values)

		jsonResult, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal label values: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(jsonResult)), nil
	}
}

// GetSeriesHandler handles the retrieval of time series.
func GetSeriesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("GetSeriesHandler called")
		slog.Debug("GetSeriesHandler params", "params", req.Params)

		promClient, err := getPromClient(ctx, opts)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
		}

		// Get required matches parameter
		matchesStr, err := req.RequireString("matches")
		if err != nil {
			return errorResult("matches parameter is required and must be a string")
		}

		// Parse matches - could be comma-separated
		matches := []string{matchesStr}
		// If it contains comma outside of braces, split it
		// For simplicity, treat the entire string as one match for now
		// Users can make multiple calls if needed

		// Get optional parameters
		startStr := req.GetString("start", "")
		endStr := req.GetString("end", "")

		// Default to last hour if not specified
		var startTime, endTime time.Time
		if startStr == "" && endStr == "" {
			endTime = time.Now()
			startTime = endTime.Add(-prometheus.ListMetricsTimeRange)
		} else {
			if startStr != "" {
				startTime, err = prometheus.ParseTimestamp(startStr)
				if err != nil {
					return errorResult(fmt.Sprintf("invalid start time format: %s", err.Error()))
				}
			}
			if endStr != "" {
				endTime, err = prometheus.ParseTimestamp(endStr)
				if err != nil {
					return errorResult(fmt.Sprintf("invalid end time format: %s", err.Error()))
				}
			}
		}

		// Get series
		series, err := promClient.GetSeries(ctx, matches, startTime, endTime)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get series: %s", err.Error()))
		}

		output := SeriesOutput{
			Series:      series,
			Cardinality: len(series),
		}

		slog.Info("GetSeriesHandler executed successfully", "cardinality", len(series))
		slog.Debug("GetSeriesHandler results", "results", series)

		jsonResult, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal series: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(jsonResult)), nil
	}
}

// GetAlertsHandler handles the retrieval of alerts from Alertmanager.
func GetAlertsHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("GetAlertsHandler called")
		slog.Debug("GetAlertsHandler params", "params", req.Params)

		amClient, err := getAlertmanagerClient(ctx, opts)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to create Alertmanager client: %s", err.Error()))
		}

		var active, silenced, inhibited, unprocessed *bool
		if req.Params.Arguments != nil {
			if args, ok := req.Params.Arguments.(map[string]any); ok {
				if activeVal, ok := args["active"].(bool); ok {
					active = &activeVal
				}
				if silencedVal, ok := args["silenced"].(bool); ok {
					silenced = &silencedVal
				}
				if inhibitedVal, ok := args["inhibited"].(bool); ok {
					inhibited = &inhibitedVal
				}
				if unprocessedVal, ok := args["unprocessed"].(bool); ok {
					unprocessed = &unprocessedVal
				}
			}
		}

		// Get optional string parameters
		filterStr := req.GetString("filter", "")
		receiver := req.GetString("receiver", "")
		var filter []string
		if filterStr != "" {
			// Split by comma if multiple filters are provided
			filter = strings.Split(filterStr, ",")
			for i := range filter {
				filter[i] = strings.TrimSpace(filter[i])
			}
		}

		alerts, err := amClient.GetAlerts(ctx, active, silenced, inhibited, unprocessed, filter, receiver)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get alerts: %s", err.Error()))
		}

		// Convert to output format
		output := AlertsOutput{
			Alerts: make([]Alert, len(alerts)),
		}

		for i, alert := range alerts {
			labels := make(map[string]string)
			maps.Copy(labels, alert.Labels)

			annotations := make(map[string]string)
			maps.Copy(annotations, alert.Annotations)

			var silencedBy, inhibitedBy []string
			var state string
			if alert.Status != nil {
				if alert.Status.SilencedBy != nil {
					silencedBy = alert.Status.SilencedBy
				}
				if alert.Status.InhibitedBy != nil {
					inhibitedBy = alert.Status.InhibitedBy
				}
				if alert.Status.State != nil {
					state = *alert.Status.State
				}
			}
			if silencedBy == nil {
				silencedBy = []string{}
			}
			if inhibitedBy == nil {
				inhibitedBy = []string{}
			}

			var startsAt, endsAt string
			if alert.StartsAt != nil {
				startsAt = alert.StartsAt.String()
			}
			if alert.EndsAt != nil {
				endsAt = alert.EndsAt.String()
			}

			output.Alerts[i] = Alert{
				Labels:      labels,
				Annotations: annotations,
				StartsAt:    startsAt,
				EndsAt:      endsAt,
				Status: AlertStatus{
					State:       state,
					SilencedBy:  silencedBy,
					InhibitedBy: inhibitedBy,
				},
			}
		}

		slog.Info("GetAlertsHandler executed successfully", "alertCount", len(alerts))
		slog.Debug("GetAlertsHandler results", "results", output.Alerts)

		jsonResult, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal alerts: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(jsonResult)), nil
	}
}

// GetSilencesHandler handles the retrieval of silences from Alertmanager.
func GetSilencesHandler(opts ObsMCPOptions) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("GetSilencesHandler called")
		slog.Debug("GetSilencesHandler params", "params", req.Params)

		amClient, err := getAlertmanagerClient(ctx, opts)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to create Alertmanager client: %s", err.Error()))
		}

		filterStr := req.GetString("filter", "")
		var filter []string
		if filterStr != "" {
			// Split by comma if multiple filters are provided
			filter = strings.Split(filterStr, ",")
			for i := range filter {
				filter[i] = strings.TrimSpace(filter[i])
			}
		}

		silences, err := amClient.GetSilences(ctx, filter)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to get silences: %s", err.Error()))
		}

		output := SilencesOutput{
			Silences: make([]Silence, len(silences)),
		}

		for i, silence := range silences {
			matchers := make([]Matcher, len(silence.Matchers))
			for j, m := range silence.Matchers {
				isEqual := true
				if m.IsEqual != nil {
					isEqual = *m.IsEqual
				}
				var name, value string
				var isRegex bool
				if m.Name != nil {
					name = *m.Name
				}
				if m.Value != nil {
					value = *m.Value
				}
				if m.IsRegex != nil {
					isRegex = *m.IsRegex
				}
				matchers[j] = Matcher{
					Name:    name,
					Value:   value,
					IsRegex: isRegex,
					IsEqual: isEqual,
				}
			}

			var id, state, createdBy, comment, startsAt, endsAt string
			if silence.ID != nil {
				id = *silence.ID
			}
			if silence.Status != nil && silence.Status.State != nil {
				state = *silence.Status.State
			}
			if silence.StartsAt != nil {
				startsAt = silence.StartsAt.String()
			}
			if silence.EndsAt != nil {
				endsAt = silence.EndsAt.String()
			}
			if silence.CreatedBy != nil {
				createdBy = *silence.CreatedBy
			}
			if silence.Comment != nil {
				comment = *silence.Comment
			}

			output.Silences[i] = Silence{
				ID: id,
				Status: SilenceStatus{
					State: state,
				},
				Matchers:  matchers,
				StartsAt:  startsAt,
				EndsAt:    endsAt,
				CreatedBy: createdBy,
				Comment:   comment,
			}
		}

		slog.Info("GetSilencesHandler executed successfully", "silenceCount", len(silences))
		slog.Debug("GetSilencesHandler results", "results", output.Silences)

		jsonResult, err := json.Marshal(output)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal silences: %s", err.Error()))
		}

		return mcp.NewToolResultStructured(output, string(jsonResult)), nil
	}
}

func getAlertmanagerClient(ctx context.Context, opts ObsMCPOptions) (alertmanager.Loader, error) {
	// Check if a test client was injected via context
	if testClient := ctx.Value(TestAlertmanagerClientKey); testClient != nil {
		if client, ok := testClient.(alertmanager.Loader); ok {
			return client, nil
		}
	}

	apiConfig, err := createAPIConfig(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %v", err)
	}

	// Update the address to use AlertmanagerURL instead of MetricsBackendURL
	apiConfig.Address = opts.AlertmanagerURL

	amClient, err := alertmanager.NewAlertmanagerClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alertmanager client: %v", err)
	}

	return amClient, nil
}
