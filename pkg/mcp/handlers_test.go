package mcp

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

// MockedLoader is a mock implementation of prometheus.PromClient for testing
type MockedLoader struct {
	ListMetricsFunc         func(ctx context.Context) ([]string, error)
	ExecuteRangeQueryFunc   func(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]interface{}, error)
	ExecuteInstantQueryFunc func(ctx context.Context, query string, time time.Time) (map[string]interface{}, error)
}

func (m *MockedLoader) ListMetrics(ctx context.Context) ([]string, error) {
	if m.ListMetricsFunc != nil {
		return m.ListMetricsFunc(ctx)
	}
	return []string{}, nil
}

func (m *MockedLoader) ExecuteRangeQuery(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]interface{}, error) {
	if m.ExecuteRangeQueryFunc != nil {
		return m.ExecuteRangeQueryFunc(ctx, query, start, end, step)
	}
	return map[string]interface{}{
		"resultType": "matrix",
		"result":     []interface{}{},
	}, nil
}

func (m *MockedLoader) ExecuteInstantQuery(ctx context.Context, query string, time time.Time) (map[string]interface{}, error) {
	if m.ExecuteInstantQueryFunc != nil {
		return m.ExecuteInstantQueryFunc(ctx, query, time)
	}
	return map[string]interface{}{
		"resultType": "vector",
		"result":     []interface{}{},
	}, nil
}

// Ensure MockPromClient implements prometheus.PromClient at compile time
var _ prometheus.Loader = (*MockedLoader)(nil)

// newMockRequest creates a CallToolRequest with the given parameters
func newMockRequest(params map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "execute_range_query",
			Arguments: params,
		},
	}
}

// withMockClient returns a context with the mock client injected
func withMockClient(ctx context.Context, client prometheus.Loader) context.Context {
	return context.WithValue(ctx, TestPromClientKey, client)
}

func TestExecuteRangeQueryHandler_ExplicitTimeRange_RFC3339(t *testing.T) {
	expectedStart, _ := prometheus.ParseTimestamp("2024-01-01T00:00:00Z")
	expectedEnd, _ := prometheus.ParseTimestamp("2024-01-01T01:00:00Z")

	mockClient := &MockedLoader{
		ExecuteRangeQueryFunc: func(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]interface{}, error) {
			if query != "up{job=\"api\"}" {
				t.Errorf("expected query 'up{job=\"api\"}', got %q", query)
			}
			if step != time.Minute {
				t.Errorf("expected step 1m, got %v", step)
			}
			if !start.Equal(expectedStart) {
				t.Errorf("expected start %v, got %v", expectedStart, start)
			}
			if !end.Equal(expectedEnd) {
				t.Errorf("expected end %v, got %v", expectedEnd, end)
			}
			return map[string]interface{}{"resultType": "matrix", "result": []interface{}{}}, nil
		},
	}

	ctx := withMockClient(context.Background(), mockClient)
	handler := ExecuteRangeQueryHandler(ObsMCPOptions{})
	req := newMockRequest(map[string]interface{}{
		"query": "up{job=\"api\"}",
		"step":  "1m",
		"start": "2024-01-01T00:00:00Z",
		"end":   "2024-01-01T01:00:00Z",
	})

	result, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %v", getErrorMessage(t, result))
	}
}

func TestExecuteRangeQueryHandler_StepParsing_ValidSteps(t *testing.T) {
	mockClient := &MockedLoader{
		ExecuteRangeQueryFunc: func(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]interface{}, error) {
			return map[string]interface{}{"resultType": "matrix", "result": []interface{}{}}, nil
		},
	}

	ctx := withMockClient(context.Background(), mockClient)
	handler := ExecuteRangeQueryHandler(ObsMCPOptions{})

	req := newMockRequest(map[string]interface{}{
		"query": "up{job=\"api\"}",
		"step":  "30s",
	})
	result, err := handler(ctx, req)
	if err != nil || result.IsError {
		t.Fatalf("30s step failed: %v", err)
	}

	req = newMockRequest(map[string]interface{}{
		"query": "up{job=\"api\"}",
		"step":  "5m",
	})
	result, err = handler(ctx, req)
	if err != nil || result.IsError {
		t.Fatalf("5m step failed: %v", err)
	}

	req = newMockRequest(map[string]interface{}{
		"query": "up{job=\"api\"}",
		"step":  "1h",
	})
	result, err = handler(ctx, req)
	if err != nil || result.IsError {
		t.Fatalf("1h step failed: %v", err)
	}
}

func TestExecuteRangeQueryHandler_RequiredParameters(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]interface{}
		expectedError string
	}{
		{
			name:          "missing query parameter",
			params:        map[string]interface{}{"step": "1m"},
			expectedError: "query parameter is required and must be a string",
		},
		{
			name:          "missing step parameter",
			params:        map[string]interface{}{"query": "up{job=\"api\"}"},
			expectedError: "step parameter is required and must be a string",
		},
		{
			name: "invalid step format",
			params: map[string]interface{}{
				"query": "up{job=\"api\"}",
				"step":  "invalid",
			},
			expectedError: "invalid step format: not a valid duration string: \"invalid\"",
		},
		{
			name: "start without end",
			params: map[string]interface{}{
				"query": "up{job=\"api\"}",
				"step":  "1m",
				"start": "2024-01-01T00:00:00Z",
			},
			expectedError: "both start and end must be provided together",
		},
		{
			name: "end without start",
			params: map[string]interface{}{
				"query": "up{job=\"api\"}",
				"step":  "1m",
				"end":   "2024-01-01T00:00:00Z",
			},
			expectedError: "both start and end must be provided together",
		},
		{
			name: "start, end, and duration",
			params: map[string]interface{}{
				"query":    "up{job=\"api\"}",
				"step":     "1m",
				"start":    "2024-01-01T00:00:00Z",
				"end":      "2024-01-01T01:00:00Z",
				"duration": "1h",
			},
			expectedError: "cannot specify both start/end and duration parameters",
		},
		{
			name: "invalid start time",
			params: map[string]interface{}{
				"query": "up{job=\"api\"}",
				"step":  "1m",
				"start": "invalid",
				"end":   "2024-01-01T01:00:00Z",
			},
			expectedError: "invalid start time format: timestamp must be RFC3339 format or Unix timestamp",
		},
		{
			name: "invalid end time",
			params: map[string]interface{}{
				"query": "up{job=\"api\"}",
				"step":  "1m",
				"start": "2024-01-01T00:00:00Z",
				"end":   "invalid",
			},
			expectedError: "invalid end time format: timestamp must be RFC3339 format or Unix timestamp",
		},
		{
			name: "invalid duration",
			params: map[string]interface{}{
				"query":    "up{job=\"api\"}",
				"step":     "1m",
				"duration": "invalid",
			},
			expectedError: "invalid duration format: not a valid duration string: \"invalid\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock client that should never be called for parameter validation errors
			mockClient := &MockedLoader{}

			ctx := withMockClient(context.Background(), mockClient)
			handler := ExecuteRangeQueryHandler(ObsMCPOptions{})
			req := newMockRequest(tt.params)
			result, _ := handler(ctx, req)

			// Validate specific error message
			errorMsg := getErrorMessage(t, result)
			if errorMsg != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, errorMsg)
			}
		})
	}
}

func TestExecuteRangeQueryHandler_DurationMode_DefaultOneHour(t *testing.T) {
	mockClient := &MockedLoader{
		ExecuteRangeQueryFunc: func(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]interface{}, error) {
			if query != "up{job=\"api\"}" {
				t.Errorf("expected query 'up{job=\"api\"}', got %q", query)
			}
			if step != time.Minute {
				t.Errorf("expected step 1m, got %v", step)
			}
			duration := end.Sub(start)
			if duration < 59*time.Minute || duration > 61*time.Minute {
				t.Errorf("expected duration ~1h, got %v", duration)
			}
			if time.Since(end) > 2*time.Second {
				t.Errorf("expected end to be approximately now, got %v ago", time.Since(end))
			}
			return map[string]interface{}{"resultType": "matrix", "result": []interface{}{}}, nil
		},
	}

	ctx := withMockClient(context.Background(), mockClient)
	handler := ExecuteRangeQueryHandler(ObsMCPOptions{})
	req := newMockRequest(map[string]interface{}{
		"query": "up{job=\"api\"}",
		"step":  "1m",
	})

	result, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %v", getErrorMessage(t, result))
	}
}

func TestExecuteRangeQueryHandler_DurationMode_CustomDuration(t *testing.T) {
	mockClient := &MockedLoader{
		ExecuteRangeQueryFunc: func(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]interface{}, error) {
			if query != "rate(http_requests_total{job=\"api\"}[5m])" {
				t.Errorf("expected query 'rate(http_requests_total{job=\"api\"}[5m])', got %q", query)
			}
			if step != 30*time.Second {
				t.Errorf("expected step 30s, got %v", step)
			}
			duration := end.Sub(start)
			if duration < 29*time.Minute || duration > 31*time.Minute {
				t.Errorf("expected duration ~30m, got %v", duration)
			}
			return map[string]interface{}{"resultType": "matrix", "result": []interface{}{}}, nil
		},
	}

	ctx := withMockClient(context.Background(), mockClient)
	handler := ExecuteRangeQueryHandler(ObsMCPOptions{})
	req := newMockRequest(map[string]interface{}{
		"query":    "rate(http_requests_total{job=\"api\"}[5m])",
		"step":     "30s",
		"duration": "30m",
	})

	result, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %v", getErrorMessage(t, result))
	}
}

func TestExecuteRangeQueryHandler_DurationMode_NOWKeyword(t *testing.T) {
	mockClient := &MockedLoader{
		ExecuteRangeQueryFunc: func(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]interface{}, error) {
			duration := end.Sub(start)
			if duration < 59*time.Minute || duration > 61*time.Minute {
				t.Errorf("expected duration ~1h when NOW is used, got %v", duration)
			}
			return map[string]interface{}{"resultType": "matrix", "result": []interface{}{}}, nil
		},
	}

	ctx := withMockClient(context.Background(), mockClient)
	handler := ExecuteRangeQueryHandler(ObsMCPOptions{})

	// Test with NOW in end
	req := newMockRequest(map[string]interface{}{
		"query": "up{job=\"api\"}",
		"step":  "1m",
		"end":   "NOW",
	})
	result, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %v", getErrorMessage(t, result))
	}
}

// Helper to extract error message from result
func getErrorMessage(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("expected error result, got success")
		return ""
	}
	switch content := result.Content[0].(type) {
	case mcp.TextContent:
		return content.Text
	default:
		return fmt.Sprintf("%v", content)
	}
}
