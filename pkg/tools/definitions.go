package tools

import "slices"

// All tool definitions as a single source of truth
var (
	ListMetrics = ToolDef[ListMetricsOutput]{
		Name:        "list_metrics",
		Description: ListMetricsPrompt,
		Title:       "List Available Metrics",
		Params: []ParamDef{
			{
				Name:        "name_regex",
				Type:        ParamTypeString,
				Description: "Regex pattern to filter metric names. IMPORTANT: Metric names are typically prefixed (e.g., 'prometheus_tsdb_head_series'). Use wildcards to match substrings: '.*tsdb.*' matches any metric containing 'tsdb', while 'tsdb' only matches the exact string 'tsdb'. Examples: 'http_.*' (starts with http_), '.*memory.*' (contains memory), 'node_.*' (starts with node_). This parameter is required. Don't pass in blanket regex like '.*' or '.+'.",
				Required:    true,
			},
		},
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
	}

	ExecuteInstantQuery = ToolDef[InstantQueryOutput]{
		Name:        "execute_instant_query",
		Description: ExecuteInstantQueryPrompt,
		Title:       "Execute Instant Query",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{
				Name:        "query",
				Type:        ParamTypeString,
				Description: "PromQL query string using metric names verified via list_metrics",
				Required:    true,
			},
			{
				Name:        "time",
				Type:        ParamTypeString,
				Description: "Evaluation time as RFC3339 or Unix timestamp. Omit or use 'NOW' for current time.",
				Required:    false,
			},
		},
	}

	ExecuteRangeQuery = ToolDef[RangeQueryOutput]{
		Name:        "execute_range_query",
		Description: ExecuteRangeQueryPrompt,
		Title:       "Execute Range Query",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{
				Name:        "query",
				Type:        ParamTypeString,
				Description: "PromQL query string using metric names verified via list_metrics",
				Required:    true,
			},
			{
				Name:        "step",
				Type:        ParamTypeString,
				Description: "Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps.",
				Required:    true,
				Pattern:     `^\d+[smhdwy]$`,
			},
			{
				Name:        "start",
				Type:        ParamTypeString,
				Description: "Start time as RFC3339 or Unix timestamp (optional)",
				Required:    false,
			},
			{
				Name:        "end",
				Type:        ParamTypeString,
				Description: "End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time.",
				Required:    false,
			},
			{
				Name:        "duration",
				Type:        ParamTypeString,
				Description: "Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional)",
				Required:    false,
				Pattern:     `^\d+[smhdwy]$`,
			},
		},
	}

	ShowTimeseries = ToolDef[struct{}]{
		Name:        "show_timeseries",
		Description: ShowTimeseriesPrompt,
		Title:       "Show Timeseries Chart",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: slices.Concat(ExecuteRangeQuery.Params, []ParamDef{
			{
				Name:        "title",
				Type:        ParamTypeString,
				Description: "Human-readable chart title describing what the query shows (e.g., 'API Error Rate Over Last Hour'). Displayed above the chart when provided.",
				Required:    false,
			},
			{
				Name:        "description",
				Type:        ParamTypeString,
				Description: "Explanation of the chart's meaning or context (e.g., 'Shows the rate of HTTP 5xx errors per second, broken down by pod'). Displayed below the title when provided.",
				Required:    false,
			},
		}),
		AdditionalFields: map[string]any{
			"olsUi": map[string]any{
				"id": "mcp-obs/show-timeseries",
			},
		},
	}

	GetLabelNames = ToolDef[LabelNamesOutput]{
		Name:        "get_label_names",
		Description: GetLabelNamesPrompt,
		Title:       "Get Label Names",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{
				Name:        "metric",
				Type:        ParamTypeString,
				Description: "Metric name (from list_metrics) to get label names for. Leave empty for all metrics.",
				Required:    false,
			},
			{
				Name:        "start",
				Type:        ParamTypeString,
				Description: "Start time for label discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)",
				Required:    false,
			},
			{
				Name:        "end",
				Type:        ParamTypeString,
				Description: "End time for label discovery as RFC3339 or Unix timestamp (optional, defaults to now)",
				Required:    false,
			},
		},
	}

	GetLabelValues = ToolDef[LabelValuesOutput]{
		Name:        "get_label_values",
		Description: GetLabelValuesPrompt,
		Title:       "Get Label Values",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{
				Name:        "label",
				Type:        ParamTypeString,
				Description: "Label name (from get_label_names) to get values for",
				Required:    true,
			},
			{
				Name:        "metric",
				Type:        ParamTypeString,
				Description: "Metric name (from list_metrics) to scope the label values to. Leave empty for all metrics.",
				Required:    false,
			},
			{
				Name:        "start",
				Type:        ParamTypeString,
				Description: "Start time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)",
				Required:    false,
			},
			{
				Name:        "end",
				Type:        ParamTypeString,
				Description: "End time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to now)",
				Required:    false,
			},
		},
	}

	GetSeries = ToolDef[SeriesOutput]{
		Name:        "get_series",
		Description: GetSeriesPrompt,
		Title:       "Get Series",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{
				Name:        "matches",
				Type:        ParamTypeString,
				Description: "PromQL series selector using metric names from list_metrics",
				Required:    true,
			},
			{
				Name:        "start",
				Type:        ParamTypeString,
				Description: "Start time for series discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)",
				Required:    false,
			},
			{
				Name:        "end",
				Type:        ParamTypeString,
				Description: "End time for series discovery as RFC3339 or Unix timestamp (optional, defaults to now)",
				Required:    false,
			},
		},
	}

	GetAlerts = ToolDef[AlertsOutput]{
		Name:        "get_alerts",
		Description: GetAlertsPrompt,
		Title:       "Get Alerts",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{
				Name:        "active",
				Type:        ParamTypeBoolean,
				Description: "Filter for active alerts only (true/false, optional)",
				Required:    false,
			},
			{
				Name:        "silenced",
				Type:        ParamTypeBoolean,
				Description: "Filter for silenced alerts only (true/false, optional)",
				Required:    false,
			},
			{
				Name:        "inhibited",
				Type:        ParamTypeBoolean,
				Description: "Filter for inhibited alerts only (true/false, optional)",
				Required:    false,
			},
			{
				Name:        "unprocessed",
				Type:        ParamTypeBoolean,
				Description: "Filter for unprocessed alerts only (true/false, optional)",
				Required:    false,
			},
			{
				Name:        "filter",
				Type:        ParamTypeString,
				Description: "Label matchers to filter alerts (e.g., 'alertname=HighCPU', optional)",
				Required:    false,
			},
			{
				Name:        "receiver",
				Type:        ParamTypeString,
				Description: "Receiver name to filter alerts (optional)",
				Required:    false,
			},
		},
	}

	GetSilences = ToolDef[SilencesOutput]{
		Name:        "get_silences",
		Description: GetSilencesPrompt,
		Title:       "Get Silences",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{
				Name:        "filter",
				Type:        ParamTypeString,
				Description: "Label matchers to filter silences (e.g., 'alertname=HighCPU', optional)",
				Required:    false,
			},
		},
	}

	GetCurrentTime = ToolDef[CurrentTimeOutput]{
		Name:        "get_current_time",
		Description: "Get the current date and time in RFC3339 format (UTC). Useful for time windows in Tempo queries.",
		Title:       "Get Current Time",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params:      []ParamDef{},
	}

	TempoListInstances = ToolDef[TempoListInstancesOutput]{
		Name:        "tempo_list_instances",
		Description: "List all Tempo instances. The assistant should display the instances in a table.",
		Title:       "List Tempo Instances",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params:      []ParamDef{},
	}

	TempoGetTraceByID = ToolDef[TempoTextOutput]{
		Name:        "tempo_get_trace_by_id",
		Description: "Get a trace by trace ID",
		Title:       "Tempo Get Trace By ID",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{Name: "tempoNamespace", Type: ParamTypeString, Description: "The namespace of the Tempo instance to query", Required: true},
			{Name: "tempoName", Type: ParamTypeString, Description: "The name of the Tempo instance to query", Required: true},
			{Name: "tenant", Type: ParamTypeString, Description: "The tenant to query", Required: true},
			{Name: "traceid", Type: ParamTypeString, Description: "Trace ID of the trace", Required: true},
			{Name: "start", Type: ParamTypeString, Description: "Start time in RFC 3339 format", Required: false},
			{Name: "end", Type: ParamTypeString, Description: "End time in RFC 3339 format", Required: false},
		},
	}

	TempoSearchTraces = ToolDef[TempoTextOutput]{
		Name:        "tempo_search_traces",
		Description: "Search for traces in Tempo",
		Title:       "Tempo Search Traces",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{Name: "tempoNamespace", Type: ParamTypeString, Description: "The namespace of the Tempo instance to query", Required: true},
			{Name: "tempoName", Type: ParamTypeString, Description: "The name of the Tempo instance to query", Required: true},
			{Name: "tenant", Type: ParamTypeString, Description: "The tenant to query", Required: true},
			{Name: "query", Type: ParamTypeString, Description: "Search query in the TraceQL query language", Required: true},
			{Name: "limit", Type: ParamTypeString, Description: "Maximum search results (integer as string; omit for server default)", Required: false},
			{Name: "start", Type: ParamTypeString, Description: "Start time in RFC 3339 format", Required: false},
			{Name: "end", Type: ParamTypeString, Description: "End time in RFC 3339 format", Required: false},
			{Name: "spss", Type: ParamTypeString, Description: "Spans per span-set limit (integer as string; omit for server default)", Required: false},
		},
	}

	TempoSearchTags = ToolDef[TempoTextOutput]{
		Name:        "tempo_search_tags",
		Description: "Search for tag names in Tempo",
		Title:       "Tempo Search Tags",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{Name: "tempoNamespace", Type: ParamTypeString, Description: "The namespace of the Tempo instance to query", Required: true},
			{Name: "tempoName", Type: ParamTypeString, Description: "The name of the Tempo instance to query", Required: true},
			{Name: "tenant", Type: ParamTypeString, Description: "The tenant to query", Required: true},
			{Name: "scope", Type: ParamTypeString, Description: "Scope to filter tags: resource, span, intrinsic, event, link, or instrumentation", Required: false},
			{Name: "query", Type: ParamTypeString, Description: "TraceQL query for filtering tag names", Required: false},
			{Name: "start", Type: ParamTypeString, Description: "Start time in RFC 3339 format", Required: false},
			{Name: "end", Type: ParamTypeString, Description: "End time in RFC 3339 format", Required: false},
			{Name: "limit", Type: ParamTypeString, Description: "Maximum number of tag names per scope (integer as string)", Required: false},
			{Name: "maxStaleValues", Type: ParamTypeString, Description: "Search termination threshold for stale values (integer as string)", Required: false},
		},
	}

	TempoSearchTagValues = ToolDef[TempoTextOutput]{
		Name:        "tempo_search_tag_values",
		Description: "Search for tag values in Tempo",
		Title:       "Tempo Search Tag Values",
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
		Params: []ParamDef{
			{Name: "tempoNamespace", Type: ParamTypeString, Description: "The namespace of the Tempo instance to query", Required: true},
			{Name: "tempoName", Type: ParamTypeString, Description: "The name of the Tempo instance to query", Required: true},
			{Name: "tenant", Type: ParamTypeString, Description: "The tenant to query", Required: true},
			{Name: "tag", Type: ParamTypeString, Description: "The tag name to get values for", Required: true},
			{Name: "query", Type: ParamTypeString, Description: "TraceQL query for filtering tag values", Required: false},
			{Name: "start", Type: ParamTypeString, Description: "Start time in RFC 3339 format", Required: false},
			{Name: "end", Type: ParamTypeString, Description: "End time in RFC 3339 format", Required: false},
			{Name: "limit", Type: ParamTypeString, Description: "Maximum number of tag values to return (integer as string)", Required: false},
			{Name: "maxStaleValues", Type: ParamTypeString, Description: "Search termination threshold for stale values (integer as string)", Required: false},
		},
	}
)

// AllTools returns all tool definitions
func AllTools() []ToolDefInterface {
	return []ToolDefInterface{
		ListMetrics,
		ExecuteInstantQuery,
		ExecuteRangeQuery,
		ShowTimeseries,
		GetLabelNames,
		GetLabelValues,
		GetSeries,
		GetAlerts,
		GetSilences,
		GetCurrentTime,
		TempoListInstances,
		TempoGetTraceByID,
		TempoSearchTraces,
		TempoSearchTags,
		TempoSearchTagValues,
	}
}
