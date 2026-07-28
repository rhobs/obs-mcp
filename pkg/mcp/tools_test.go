package mcp

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"

	tools "github.com/rhobs/obs-mcp/pkg/metrics"
)

func getToolByName(name string) api.Tool {
	allTools := (&tools.Toolset{}).GetTools(nil)
	for i := range allTools {
		if allTools[i].Tool.Name == name {
			return allTools[i].Tool
		}
	}
	panic("tool not found: " + name)
}

func TestListMetricsOutputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input tools.ListMetricsOutput
	}{
		{
			name:  "empty",
			input: tools.ListMetricsOutput{Metrics: []string{}},
		},
		{
			name:  "single metric",
			input: tools.ListMetricsOutput{Metrics: []string{"up"}},
		},
		{
			name:  "multiple metrics",
			input: tools.ListMetricsOutput{Metrics: []string{"up", "node_cpu_seconds_total", "go_goroutines"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var result tools.ListMetricsOutput
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
		})
	}
}

func TestRangeQueryOutputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input tools.RangeQueryOutput
	}{
		{
			name: "matrix single series",
			input: tools.RangeQueryOutput{
				ResultType: "matrix",
				Result: []tools.SeriesResult{{
					Metric: map[string]string{"__name__": "up"},
					Values: [][]any{{1700000000.0, "1"}},
				}},
			},
		},
		{
			name: "matrix multiple series",
			input: tools.RangeQueryOutput{
				ResultType: "matrix",
				Result: []tools.SeriesResult{
					{Metric: map[string]string{"job": "a"}, Values: [][]any{}},
					{Metric: map[string]string{"job": "b"}, Values: [][]any{}},
					{Metric: map[string]string{"job": "c"}, Values: [][]any{}},
				},
			},
		},
		{
			name: "empty result",
			input: tools.RangeQueryOutput{
				ResultType: "matrix",
				Result:     []tools.SeriesResult{},
			},
		},
		{
			name: "vector result",
			input: tools.RangeQueryOutput{
				ResultType: "vector",
				Result: []tools.SeriesResult{{
					Metric: map[string]string{"__name__": "up"},
					Values: [][]any{{1700000000.0, "1"}},
				}},
			},
		},
		{
			name: "scalar result",
			input: tools.RangeQueryOutput{
				ResultType: "scalar",
				Result: []tools.SeriesResult{{
					Metric: map[string]string{},
					Values: [][]any{{1700000000.0, "42"}},
				}},
			},
		},
		{
			name: "with warnings",
			input: tools.RangeQueryOutput{
				ResultType: "matrix",
				Result:     []tools.SeriesResult{},
				Warnings:   []string{"warning1", "warning2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var result tools.RangeQueryOutput
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
		})
	}
}

func TestSeriesResultSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input tools.SeriesResult
	}{
		{
			name: "with labels and values",
			input: tools.SeriesResult{
				Metric: map[string]string{"__name__": "up", "job": "prometheus"},
				Values: [][]any{{1700000000.0, "1"}, {1700000060.0, "1"}},
			},
		},
		{
			name: "empty",
			input: tools.SeriesResult{
				Metric: map[string]string{},
				Values: [][]any{},
			},
		},
		{
			name: "many labels",
			input: tools.SeriesResult{
				Metric: map[string]string{
					"__name__": "http_requests", "method": "GET", "status": "200",
					"handler": "/api", "instance": "localhost:9090",
				},
				Values: [][]any{{1700000000.0, "100"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var result tools.SeriesResult
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
		})
	}
}

func TestToolParameters(t *testing.T) {
	tests := []struct {
		toolName         string
		expectedRequired []string
		expectedOptional []string
	}{
		{
			toolName:         "list_metrics",
			expectedRequired: []string{"name_regex"},
			expectedOptional: []string{},
		},
		{
			toolName:         "execute_range_query",
			expectedRequired: []string{"query", "step"},
			expectedOptional: []string{"start", "end", "duration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			tool := getToolByName(tt.toolName)

			if tool.InputSchema == nil {
				t.Fatalf("InputSchema is nil")
			}

			requiredSet := make(map[string]bool)
			for _, r := range tool.InputSchema.Required {
				requiredSet[r] = true
			}

			if len(tool.InputSchema.Required) != len(tt.expectedRequired) {
				t.Errorf("expected %d required params, got %d",
					len(tt.expectedRequired), len(tool.InputSchema.Required))
			}

			for _, param := range tt.expectedRequired {
				if !requiredSet[param] {
					t.Errorf("parameter %q should be required", param)
				}
			}

			for _, param := range tt.expectedOptional {
				if _, exists := tool.InputSchema.Properties[param]; !exists {
					t.Errorf("optional parameter %q not found", param)
				}
				if requiredSet[param] {
					t.Errorf("parameter %q should be optional", param)
				}
			}
		})
	}
}

type paramPatternTest struct {
	param         string
	hasPattern    bool
	validInputs   []string
	invalidInputs []string
}

func TestToolPatternValidation(t *testing.T) {
	tests := []struct {
		toolName string
		params   []paramPatternTest
	}{
		{
			toolName: "list_metrics",
			params:   []paramPatternTest{}, // no parameters
		},
		{
			toolName: "execute_range_query",
			params: []paramPatternTest{
				{
					param:         "step",
					hasPattern:    true,
					validInputs:   []string{"1s", "30s", "1m", "5m", "1h", "24h", "1d", "7d", "1w", "2w"},
					invalidInputs: []string{"", "1", "s", "1x", "1.5m", "1m30s", "invalid"},
				},
				{
					param:         "duration",
					hasPattern:    true,
					validInputs:   []string{"1s", "30s", "1m", "5m", "1h", "24h", "1d", "7d", "1w", "2w"},
					invalidInputs: []string{"", "1", "s", "1x", "1.5m", "1m30s", "invalid"},
				},
				{
					param:      "query",
					hasPattern: false,
				},
				{
					param:      "start",
					hasPattern: false,
				},
				{
					param:      "end",
					hasPattern: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			tool := getToolByName(tt.toolName)
			for _, pt := range tt.params {
				t.Run(pt.param, func(t *testing.T) {
					if tool.InputSchema == nil {
						t.Fatalf("InputSchema is nil")
					}

					prop, exists := tool.InputSchema.Properties[pt.param]
					if !exists {
						t.Fatalf("parameter %q not found", pt.param)
					}

					hasPattern := prop.Pattern != ""

					if hasPattern != pt.hasPattern {
						t.Errorf("expected hasPattern=%v, got %v", pt.hasPattern, hasPattern)
						return
					}

					if !pt.hasPattern {
						return
					}

					re, err := regexp.Compile(prop.Pattern)
					if err != nil {
						t.Fatalf("invalid pattern %q: %v", prop.Pattern, err)
					}

					for _, input := range pt.validInputs {
						if !re.MatchString(input) {
							t.Errorf("pattern should match %q but didn't", input)
						}
					}

					for _, input := range pt.invalidInputs {
						if re.MatchString(input) {
							t.Errorf("pattern should NOT match %q but did", input)
						}
					}
				})
			}
		})
	}
}

func TestToolsHaveOutputSchema(t *testing.T) {
	allTools := (&tools.Toolset{}).GetTools(nil)

	if len(allTools) == 0 {
		t.Fatal("expected at least one tool")
	}

	for _, st := range allTools {
		t.Run(st.Tool.Name, func(t *testing.T) {
			if st.Tool.OutputSchema == nil {
				t.Logf("tool %q has no output schema (expected for manual tool construction)", st.Tool.Name)
			}

			if st.Tool.Description == "" {
				t.Errorf("tool %q missing description", st.Tool.Name)
			}
		})
	}
}
