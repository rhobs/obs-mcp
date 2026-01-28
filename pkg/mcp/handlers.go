package mcp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rhobs/obs-mcp/pkg/resultutil"
	"github.com/rhobs/obs-mcp/pkg/tempo"
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

// GetCurrentTimeHandler returns the current UTC time in RFC3339 format.
func GetCurrentTimeHandler() mcp.ToolHandlerFor[any, tools.CurrentTimeOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, tools.CurrentTimeOutput, error) {
		return nil, tools.CurrentTimeOutput{Time: time.Now().UTC().Format(time.RFC3339)}, nil
	}
}

// TempoListInstancesHandler lists Tempo instances visible in the cluster.
func TempoListInstancesHandler(ts *tempo.TempoToolset) mcp.ToolHandlerFor[any, tools.TempoListInstancesOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, tools.TempoListInstancesOutput, error) {
		instances, err := ts.ListInstances(ctx)
		if err != nil {
			return nil, tools.TempoListInstancesOutput{}, err
		}
		return nil, tools.TempoListInstancesOutput{Instances: instances}, nil
	}
}

// TempoGetTraceByIDHandler fetches a single trace from Tempo.
func TempoGetTraceByIDHandler(ts *tempo.TempoToolset) mcp.ToolHandlerFor[tools.TempoGetTraceByIDInput, tools.TempoTextOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.TempoGetTraceByIDInput) (*mcp.CallToolResult, tools.TempoTextOutput, error) {
		if input.Traceid == "" {
			return nil, tools.TempoTextOutput{}, fmt.Errorf("traceid parameter must not be empty")
		}
		body, err := ts.GetTraceByID(ctx, input.TempoNamespace, input.TempoName, input.Tenant, input.Traceid, input.Start, input.End)
		if err != nil {
			return nil, tools.TempoTextOutput{}, err
		}
		return nil, tools.TempoTextOutput{Result: body}, nil
	}
}

// TempoSearchTracesHandler runs a TraceQL search against Tempo.
func TempoSearchTracesHandler(ts *tempo.TempoToolset) mcp.ToolHandlerFor[tools.TempoSearchTracesInput, tools.TempoTextOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.TempoSearchTracesInput) (*mcp.CallToolResult, tools.TempoTextOutput, error) {
		limit, err := optionalAtoi(input.Limit)
		if err != nil {
			return nil, tools.TempoTextOutput{}, fmt.Errorf("limit: %w", err)
		}
		spss, err := optionalAtoi(input.Spss)
		if err != nil {
			return nil, tools.TempoTextOutput{}, fmt.Errorf("spss: %w", err)
		}
		body, err := ts.SearchTraces(ctx, input.TempoNamespace, input.TempoName, input.Tenant, input.Query, limit, input.Start, input.End, spss)
		if err != nil {
			return nil, tools.TempoTextOutput{}, err
		}
		return nil, tools.TempoTextOutput{Result: body}, nil
	}
}

// TempoSearchTagsHandler lists tag names from Tempo.
func TempoSearchTagsHandler(ts *tempo.TempoToolset) mcp.ToolHandlerFor[tools.TempoSearchTagsInput, tools.TempoTextOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.TempoSearchTagsInput) (*mcp.CallToolResult, tools.TempoTextOutput, error) {
		limit, err := optionalAtoi(input.Limit)
		if err != nil {
			return nil, tools.TempoTextOutput{}, fmt.Errorf("limit: %w", err)
		}
		maxStale, err := optionalAtoi(input.MaxStaleValues)
		if err != nil {
			return nil, tools.TempoTextOutput{}, fmt.Errorf("maxStaleValues: %w", err)
		}
		body, err := ts.SearchTags(ctx, input.TempoNamespace, input.TempoName, input.Tenant, input.Scope, input.Query, input.Start, input.End, limit, maxStale)
		if err != nil {
			return nil, tools.TempoTextOutput{}, err
		}
		return nil, tools.TempoTextOutput{Result: body}, nil
	}
}

// TempoSearchTagValuesHandler lists values for a tag in Tempo.
func TempoSearchTagValuesHandler(ts *tempo.TempoToolset) mcp.ToolHandlerFor[tools.TempoSearchTagValuesInput, tools.TempoTextOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input tools.TempoSearchTagValuesInput) (*mcp.CallToolResult, tools.TempoTextOutput, error) {
		if input.Tag == "" {
			return nil, tools.TempoTextOutput{}, fmt.Errorf("tag parameter must not be empty")
		}
		limit, err := optionalAtoi(input.Limit)
		if err != nil {
			return nil, tools.TempoTextOutput{}, fmt.Errorf("limit: %w", err)
		}
		maxStale, err := optionalAtoi(input.MaxStaleValues)
		if err != nil {
			return nil, tools.TempoTextOutput{}, fmt.Errorf("maxStaleValues: %w", err)
		}
		body, err := ts.SearchTagValues(ctx, input.TempoNamespace, input.TempoName, input.Tenant, input.Tag, input.Query, input.Start, input.End, limit, maxStale)
		if err != nil {
			return nil, tools.TempoTextOutput{}, err
		}
		return nil, tools.TempoTextOutput{Result: body}, nil
	}
}

func optionalAtoi(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}
