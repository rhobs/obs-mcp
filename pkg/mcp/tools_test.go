package mcp

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestListMetricsOutputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input ListMetricsOutput
	}{
		{
			name:  "empty",
			input: ListMetricsOutput{Metrics: []string{}},
		},
		{
			name:  "single metric",
			input: ListMetricsOutput{Metrics: []string{"up"}},
		},
		{
			name:  "multiple metrics",
			input: ListMetricsOutput{Metrics: []string{"up", "node_cpu_seconds_total", "go_goroutines"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var result ListMetricsOutput
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
		})
	}
}

func TestRangeQueryOutputSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input RangeQueryOutput
	}{
		{
			name: "matrix single series",
			input: RangeQueryOutput{
				ResultType: "matrix",
				Result: []SeriesResult{{
					Metric: map[string]string{"__name__": "up"},
					Values: [][]any{{1700000000.0, "1"}},
				}},
			},
		},
		{
			name: "matrix multiple series",
			input: RangeQueryOutput{
				ResultType: "matrix",
				Result: []SeriesResult{
					{Metric: map[string]string{"job": "a"}, Values: [][]any{}},
					{Metric: map[string]string{"job": "b"}, Values: [][]any{}},
					{Metric: map[string]string{"job": "c"}, Values: [][]any{}},
				},
			},
		},
		{
			name: "empty result",
			input: RangeQueryOutput{
				ResultType: "matrix",
				Result:     []SeriesResult{},
			},
		},
		{
			name: "vector result",
			input: RangeQueryOutput{
				ResultType: "vector",
				Result: []SeriesResult{{
					Metric: map[string]string{"__name__": "up"},
					Values: [][]any{{1700000000.0, "1"}},
				}},
			},
		},
		{
			name: "scalar result",
			input: RangeQueryOutput{
				ResultType: "scalar",
				Result: []SeriesResult{{
					Metric: map[string]string{},
					Values: [][]any{{1700000000.0, "42"}},
				}},
			},
		},
		{
			name: "with warnings",
			input: RangeQueryOutput{
				ResultType: "matrix",
				Result:     []SeriesResult{},
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

			var result RangeQueryOutput
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
		})
	}
}

func TestSeriesResultSerialization(t *testing.T) {
	tests := []struct {
		name  string
		input SeriesResult
	}{
		{
			name: "with labels and values",
			input: SeriesResult{
				Metric: map[string]string{"__name__": "up", "job": "prometheus"},
				Values: [][]any{{1700000000.0, "1"}, {1700000060.0, "1"}},
			},
		},
		{
			name: "empty",
			input: SeriesResult{
				Metric: map[string]string{},
				Values: [][]any{},
			},
		},
		{
			name: "many labels",
			input: SeriesResult{
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

			var result SeriesResult
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
		})
	}
}

func TestToolParameters(t *testing.T) {
	tests := []struct {
		tool             mcp.Tool
		expectedRequired []string
		expectedOptional []string
	}{
		{
			tool:             CreateListMetricsTool(),
			expectedRequired: []string{},
			expectedOptional: []string{},
		},
		{
			tool:             CreateExecuteRangeQueryTool(),
			expectedRequired: []string{"query", "step"},
			expectedOptional: []string{"start", "end", "duration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.tool.Name, func(t *testing.T) {
			tool := tt.tool

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
		tool   mcp.Tool
		params []paramPatternTest
	}{
		{
			tool:   CreateListMetricsTool(),
			params: []paramPatternTest{}, // no parameters
		},
		{
			tool: CreateExecuteRangeQueryTool(),
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
		t.Run(tt.tool.Name, func(t *testing.T) {
			for _, pt := range tt.params {
				t.Run(pt.param, func(t *testing.T) {
					prop, exists := tt.tool.InputSchema.Properties[pt.param]
					if !exists {
						t.Fatalf("parameter %q not found", pt.param)
					}

					propMap, ok := prop.(map[string]any)
					if !ok {
						t.Fatalf("parameter %q is not a map", pt.param)
					}

					pattern, hasPattern := propMap["pattern"].(string)

					if hasPattern != pt.hasPattern {
						t.Errorf("expected hasPattern=%v, got %v", pt.hasPattern, hasPattern)
						return
					}

					if !pt.hasPattern {
						return
					}

					re, err := regexp.Compile(pattern)
					if err != nil {
						t.Fatalf("invalid pattern %q: %v", pattern, err)
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
	tools := []mcp.Tool{
		CreateListMetricsTool(),
		CreateExecuteRangeQueryTool(),
	}

	if len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			if tool.OutputSchema.Type == "" && len(tool.RawOutputSchema) == 0 {
				t.Errorf("tool %q missing output schema", tool.Name)
			}

			if tool.Description == "" {
				t.Errorf("tool %q missing description", tool.Name)
			}
		})
	}
}
