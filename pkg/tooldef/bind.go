package tooldef

import "github.com/rhobs/obs-mcp/pkg/handlers"

// GetString is a helper to extract a string parameter with a default value
func GetString(params map[string]any, key, defaultValue string) string {
	if val, ok := params[key]; ok {
		if str, ok := val.(string); ok && str != "" {
			return str
		}
	}
	return defaultValue
}

// GetBoolPtr is a helper to extract an optional boolean parameter as a pointer
func GetBoolPtr(params map[string]any, key string) *bool {
	if val, ok := params[key]; ok {
		if b, ok := val.(bool); ok {
			return &b
		}
	}
	return nil
}

// These functions eliminate duplication between MCP and Toolset handlers

func BuildInstantQueryInput(args map[string]any) handlers.InstantQueryInput {
	return handlers.InstantQueryInput{
		Query: GetString(args, "query", ""),
		Time:  GetString(args, "time", ""),
	}
}

func BuildRangeQueryInput(args map[string]any) handlers.RangeQueryInput {
	return handlers.RangeQueryInput{
		Query:    GetString(args, "query", ""),
		Step:     GetString(args, "step", ""),
		Start:    GetString(args, "start", ""),
		End:      GetString(args, "end", ""),
		Duration: GetString(args, "duration", ""),
	}
}

func BuildLabelNamesInput(args map[string]any) handlers.LabelNamesInput {
	return handlers.LabelNamesInput{
		Metric: GetString(args, "metric", ""),
		Start:  GetString(args, "start", ""),
		End:    GetString(args, "end", ""),
	}
}

func BuildLabelValuesInput(args map[string]any) handlers.LabelValuesInput {
	return handlers.LabelValuesInput{
		Label:  GetString(args, "label", ""),
		Metric: GetString(args, "metric", ""),
		Start:  GetString(args, "start", ""),
		End:    GetString(args, "end", ""),
	}
}

func BuildSeriesInput(args map[string]any) handlers.SeriesInput {
	return handlers.SeriesInput{
		Matches: GetString(args, "matches", ""),
		Start:   GetString(args, "start", ""),
		End:     GetString(args, "end", ""),
	}
}

func BuildAlertsInput(args map[string]any) handlers.AlertsInput {
	return handlers.AlertsInput{
		Active:      GetBoolPtr(args, "active"),
		Silenced:    GetBoolPtr(args, "silenced"),
		Inhibited:   GetBoolPtr(args, "inhibited"),
		Unprocessed: GetBoolPtr(args, "unprocessed"),
		Filter:      GetString(args, "filter", ""),
		Receiver:    GetString(args, "receiver", ""),
	}
}

func BuildSilencesInput(args map[string]any) handlers.SilencesInput {
	return handlers.SilencesInput{
		Filter: GetString(args, "filter", ""),
	}
}
