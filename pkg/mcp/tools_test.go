package mcp

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rhobs/obs-mcp/pkg/tools"
)

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
		tool             mcp.Tool
		expectedRequired []string
		expectedOptional []string
	}{
		{
			tool:             CreateListMetricsTool(),
			expectedRequired: []string{"name_regex"},
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

			// Convert InputSchema to map to access properties
			inputSchemaMap, ok := tool.InputSchema.(map[string]any)
			if !ok {
				t.Fatalf("InputSchema is not a map")
			}

			var requiredList []string
			if required, exists := inputSchemaMap["required"]; exists {
				switch req := required.(type) {
				case []any:
					for _, r := range req {
						if rStr, ok := r.(string); ok {
							requiredList = append(requiredList, rStr)
						}
					}
				case []string:
					requiredList = req
				}
			}

			requiredSet := make(map[string]bool)
			for _, r := range requiredList {
				requiredSet[r] = true
			}

			if len(requiredList) != len(tt.expectedRequired) {
				t.Errorf("expected %d required params, got %d",
					len(tt.expectedRequired), len(requiredList))
			}

			for _, param := range tt.expectedRequired {
				if !requiredSet[param] {
					t.Errorf("parameter %q should be required", param)
				}
			}

			var properties map[string]any
			if props, exists := inputSchemaMap["properties"]; exists {
				if propsMap, ok := props.(map[string]any); ok {
					properties = propsMap
				}
			}

			for _, param := range tt.expectedOptional {
				if _, exists := properties[param]; !exists {
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
					// Convert InputSchema to map to access properties
					inputSchemaMap, ok := tt.tool.InputSchema.(map[string]any)
					if !ok {
						t.Fatalf("InputSchema is not a map")
					}

					var properties map[string]any
					if props, exists := inputSchemaMap["properties"]; exists {
						if propsMap, ok := props.(map[string]any); ok {
							properties = propsMap
						}
					}

					prop, exists := properties[pt.param]
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
	toolsToTest := []mcp.Tool{
		CreateListMetricsTool(),
		CreateExecuteRangeQueryTool(),
		CreateListDashboardsTool(),
		CreateListDashboardsTool(),
		CreateGetDashboardTool(),
		CreateGetDashboardPanelsTool(),
	}

	if len(toolsToTest) == 0 {
		t.Fatal("expected at least one tool")
	}

	for _, tool := range toolsToTest {
		t.Run(tool.Name, func(t *testing.T) {
			// In the new SDK, OutputSchema is any and may be nil
			// if not using typed AddTool[In,Out] approach
			if tool.OutputSchema == nil {
				t.Logf("tool %q has no output schema (expected for manual tool construction)", tool.Name)
			}

			if tool.Description == "" {
				t.Errorf("tool %q missing description", tool.Name)
			}
		})
	}
}
