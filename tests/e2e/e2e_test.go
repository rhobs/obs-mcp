//go:build e2e && !openshift

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	resp, err := http.Get(testConfig.MCPURL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestBackendNotLocalhost verifies that obs-mcp is connected to a real metrics
// backend and not falling back to http://localhost:9090. A successful list_metrics
// call returning known prometheus metrics is proof of correct URL configuration.
func TestBackendNotLocalhost(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 1, "list_metrics", map[string]any{
		"name_regex": "prometheus_build_info",
	})
	if err != nil {
		t.Fatalf("Failed to call list_metrics: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("MCP error: %s -- is PROMETHEUS_URL set correctly in the deployment?", resp.Error.Message)
	}
	resultJSON, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(resultJSON), "prometheus_build_info") {
		t.Error("prometheus_build_info not found -- server may be pointing at localhost:9090 instead of the configured backend")
	}
}

func TestListMetricsReturnsKnownMetricsWithMatcher(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 2, "list_metrics", map[string]any{
		"name_regex": "prometheus.*",
	})
	if err != nil {
		t.Fatalf("Failed to call list_metrics: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}

	// Verify known metrics from prometheus are present
	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)

	expectedMetrics := []string{"prometheus_build_info"}
	for _, metric := range expectedMetrics {
		if !strings.Contains(resultStr, metric) {
			t.Errorf("Expected metric %q not found in results", metric)
		}
	}
}

func TestExecuteRangeQuery(t *testing.T) {
	skipIfThanosLacksTSDB(t)

	resp, err := mcpClient.CallTool(t, 3, "execute_range_query", map[string]any{
		"query":    `up{job=~"prometheus.*"}`,
		"step":     "1m",
		"duration": "5m",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_range_query: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("execute_range_query returned an error result: %s", resultJSON)
	}
	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}

	t.Logf("execute_range_query returned successfully")
}

func TestRangeQueryWithInvalidPromQL(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 4, "execute_range_query", map[string]any{
		"query":    `up{{{invalid`, // Invalid PromQL syntax
		"step":     "1m",
		"duration": "5m",
		"end":      "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_range_query: %v", err)
	}

	// Should return an error for invalid syntax
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Log("Correctly returned error for invalid PromQL")
		} else {
			t.Error("Expected error for invalid PromQL syntax")
		}
	}
}

func TestRangeQueryMissingRequiredParam(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 5, "execute_range_query", map[string]any{
		// Missing "query" parameter
		"step":     "1m",
		"duration": "5m",
		"end":      "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_range_query: %v", err)
	}

	// Should return an error for missing required param
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Log("Correctly returned error for missing query parameter")
		} else {
			t.Error("Expected error for missing required parameter")
		}
	}
}

func TestRangeQueryEmptyResult(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 6, "execute_range_query", map[string]any{
		"query":    `nonexistent_metric_xyz{job="test"}`,
		"step":     "1m",
		"duration": "5m",
		"end":      "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_range_query: %v", err)
	}

	// Should succeed but return empty result
	if resp.Error != nil {
		t.Errorf("Unexpected error: %s", resp.Error.Message)
	}

	t.Log("Query for non-existent metric handled correctly")
}

func TestGuardrailsBlockDangerousQuery(t *testing.T) {
	// This should be blocked by guardrails (blanket regex without label matcher)
	resp, err := mcpClient.CallTool(t, 7, "execute_range_query", map[string]any{
		"query":    `{__name__=~".+"}`, // Dangerous: selects all metrics
		"step":     "1m",
		"duration": "5m",
		"end":      "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_range_query: %v", err)
	}

	// Check if the result indicates an error (guardrails blocked it)
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Logf("Guardrails correctly blocked query")
		} else {
			t.Error("Expected guardrails to block the dangerous query, but it was allowed")
		}
	} else if resp.Error != nil {
		t.Logf("Guardrails correctly blocked query: %s", resp.Error.Message)
	} else {
		t.Error("Expected guardrails to block the dangerous query")
	}
}

func TestExecuteInstantQuery(t *testing.T) {
	skipIfThanosLacksTSDB(t)

	resp, err := mcpClient.CallTool(t, 8, "execute_instant_query", map[string]any{
		"query": `up{job=~"prometheus.*"}`,
		"time":  "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_instant_query: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("execute_instant_query returned an error result: %s", resultJSON)
	}
	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}

	t.Logf("execute_instant_query returned successfully")
}

func TestInstantQueryWithInvalidPromQL(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 9, "execute_instant_query", map[string]any{
		"query": `up{{{invalid`, // Invalid PromQL syntax
	})
	if err != nil {
		t.Fatalf("Failed to call execute_instant_query: %v", err)
	}

	// Should return an error for invalid syntax
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Log("Correctly returned error for invalid PromQL")
		} else {
			t.Error("Expected error for invalid PromQL syntax")
		}
	}
}

func TestGetLabelNames(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 10, "get_label_names", map[string]any{
		"metric": "up",
	})
	if err != nil {
		t.Fatalf("Failed to call get_label_names: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	// Verify we have common labels
	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)

	expectedLabels := []string{"job", "instance"}
	for _, label := range expectedLabels {
		if !strings.Contains(resultStr, label) {
			t.Errorf("Expected label %q not found in results", label)
		}
	}

	t.Logf("get_label_names returned successfully")
}

func TestGetLabelNamesAllMetrics(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 11, "get_label_names", map[string]any{})
	if err != nil {
		t.Fatalf("Failed to call get_label_names: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_label_names (all metrics) returned successfully")
}

func TestGetLabelValues(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 12, "get_label_values", map[string]any{
		"label":  "job",
		"metric": "up",
	})
	if err != nil {
		t.Fatalf("Failed to call get_label_values: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	// Verify we have the prometheus job
	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)

	if !strings.Contains(resultStr, "prometheus") {
		t.Errorf("Expected 'prometheus' job value not found in results")
	}

	t.Logf("get_label_values returned successfully")
}

func TestGetLabelValuesMissingRequiredParam(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 13, "get_label_values", map[string]any{
		// Missing "label" parameter
		"metric": "up",
	})
	if err != nil {
		t.Fatalf("Failed to call get_label_values: %v", err)
	}

	// Should return an error for missing required param
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Log("Correctly returned error for missing label parameter")
		} else {
			t.Error("Expected error for missing required parameter")
		}
	}
}

func TestGetSeries(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 14, "get_series", map[string]any{
		"matches": `up{job="prometheus"}`,
	})
	if err != nil {
		t.Fatalf("Failed to call get_series: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	// Verify we have cardinality information
	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)

	if !strings.Contains(resultStr, "cardinality") {
		t.Errorf("Expected 'cardinality' field not found in results")
	}

	t.Logf("get_series returned successfully")
}

func TestGetSeriesMissingRequiredParam(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 15, "get_series", map[string]any{
		// Missing "matches" parameter
	})
	if err != nil {
		t.Fatalf("Failed to call get_series: %v", err)
	}

	// Should return an error for missing required param
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Log("Correctly returned error for missing matches parameter")
		} else {
			t.Error("Expected error for missing required parameter")
		}
	}
}

func TestGetAlerts(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 16, "get_alerts", map[string]any{})
	if err != nil {
		t.Fatalf("Failed to call get_alerts: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("get_alerts returned an error result: %s", resultJSON)
	}
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_alerts returned successfully")
}

func TestGetAlertsWithActiveFilter(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 17, "get_alerts", map[string]any{
		"active": true,
	})
	if err != nil {
		t.Fatalf("Failed to call get_alerts with active filter: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("get_alerts (active filter) returned an error result: %s", resultJSON)
	}
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_alerts with active filter returned successfully")
}

func TestGetAlertsWithFilter(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 18, "get_alerts", map[string]any{
		"filter": "alertname=Watchdog",
	})
	if err != nil {
		t.Fatalf("Failed to call get_alerts with filter: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("get_alerts (Watchdog filter) returned an error result: %s", resultJSON)
		return
	}
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	// Verify Watchdog alert structure (prometheus always has Watchdog firing)
	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)

	if !strings.Contains(resultStr, "alerts") {
		t.Errorf("Expected 'alerts' field not found in results")
	}

	t.Logf("get_alerts with filter returned successfully")
}

func TestGetSilences(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 19, "get_silences", map[string]any{})
	if err != nil {
		t.Fatalf("Failed to call get_silences: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("get_silences returned an error result: %s", resultJSON)
		return
	}
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	// Verify silences field exists in response
	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)

	if !strings.Contains(resultStr, "silences") {
		t.Errorf("Expected 'silences' field not found in results")
	}

	t.Logf("get_silences returned successfully")
}

func TestGetSilencesWithFilter(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 20, "get_silences", map[string]any{
		"filter": "alertname=Watchdog",
	})
	if err != nil {
		t.Fatalf("Failed to call get_silences with filter: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("get_silences (filter) returned an error result: %s", resultJSON)
	}
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_silences with filter returned successfully")
}

func TestInstantQueryMissingRequiredParam(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 23, "execute_instant_query", map[string]any{
		// Missing "query" parameter
		"time": "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_instant_query: %v", err)
	}
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Log("Correctly returned error for missing query parameter")
		} else {
			t.Error("Expected error for missing required parameter")
		}
	}
}

func TestRangeQueryWithExplicitStartEnd(t *testing.T) {
	skipIfThanosLacksTSDB(t)

	resp, err := mcpClient.CallTool(t, 24, "execute_range_query", map[string]any{
		"query": `up{job=~"prometheus.*"}`,
		"step":  "1m",
		"start": "NOW-5m",
		"end":   "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_range_query: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("execute_range_query (explicit start/end) returned an error result: %s", resultJSON)
	}
	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}
	t.Logf("execute_range_query (explicit start/end) returned successfully")
}

func TestGetLabelNamesWithTimeRange(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 25, "get_label_names", map[string]any{
		"metric": "up",
		"start":  "NOW-1h",
		"end":    "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call get_label_names: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}
	resultJSON, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(resultJSON), "job") {
		t.Errorf("Expected label 'job' not found in results")
	}
	t.Logf("get_label_names (time range) returned successfully")
}

func TestGetLabelValuesWithTimeRange(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 26, "get_label_values", map[string]any{
		"label":  "job",
		"metric": "up",
		"start":  "NOW-1h",
		"end":    "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call get_label_values: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}
	resultJSON, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(resultJSON), "prometheus") {
		t.Errorf("Expected 'prometheus' job value not found in results")
	}
	t.Logf("get_label_values (time range) returned successfully")
}

func TestGetSeriesWithTimeRange(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 27, "get_series", map[string]any{
		"matches": `up{job="prometheus"}`,
		"start":   "NOW-1h",
		"end":     "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call get_series: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}
	resultJSON, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(resultJSON), "cardinality") {
		t.Errorf("Expected 'cardinality' field not found in results")
	}
	t.Logf("get_series (time range) returned successfully")
}

func TestGetAlertsWithBooleanFilters(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"silenced", map[string]any{"silenced": true}},
		{"inhibited", map[string]any{"inhibited": true}},
		{"unprocessed", map[string]any{"unprocessed": true}},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := mcpClient.CallTool(t, 28+i, "get_alerts", tt.args)
			if err != nil {
				t.Fatalf("Failed to call get_alerts: %v", err)
			}
			if resp.Error != nil {
				t.Errorf("MCP error: %s", resp.Error.Message)
			}
			if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
				resultJSON, _ := json.Marshal(resp.Result)
				t.Errorf("get_alerts (%s) returned an error result: %s", tt.name, resultJSON)
			}
			if resp.Result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestGetAlertsWithReceiver(t *testing.T) {
	// Query by a receiver name unlikely to exist; should return empty alerts, not an error.
	resp, err := mcpClient.CallTool(t, 31, "get_alerts", map[string]any{
		"receiver": "nonexistent-receiver-xyz",
	})
	if err != nil {
		t.Fatalf("Failed to call get_alerts with receiver: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("get_alerts (receiver) returned an error result: %s", resultJSON)
	}
	t.Log("get_alerts (receiver param) handled correctly")
}

func TestGetSilencesEmptyFilter(t *testing.T) {
	// Filter for a silence that doesn't exist — should return empty list, not error.
	resp, err := mcpClient.CallTool(t, 32, "get_silences", map[string]any{
		"filter": "alertname=NonExistentSilence12345",
	})
	if err != nil {
		t.Fatalf("Failed to call get_silences: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("get_silences (empty filter) returned an error result: %s", resultJSON)
	}
	t.Log("get_silences (non-matching filter) handled correctly")
}

func TestGetAlertsEmptyFilter(t *testing.T) {
	// Filter for non-existent alert should return empty
	resp, err := mcpClient.CallTool(t, 21, "get_alerts", map[string]any{
		"filter": "alertname=NonExistentAlert12345",
	})
	if err != nil {
		t.Fatalf("Failed to call get_alerts: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Errorf("get_alerts (empty filter) returned an error result: %s", resultJSON)
	}

	// Should succeed but may return empty alerts array
	t.Log("Query for non-existent alert handled correctly")
}

func TestTempoListInstances(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 22, "tempo_list_instances", map[string]any{})
	if err != nil {
		t.Fatalf("Failed to call tempo_list_instances: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Fatalf("tempo_list_instances returned an error result: %s", resultJSON)
	}

	structured := resp.Result["structuredContent"].(map[string]any)
	instances := structured["instances"].([]any)
	require.ElementsMatch(t, []any{
		map[string]any{"kind": "TempoStack", "tempoNamespace": "tracing", "tempoName": "tempo1", "multitenancy": false, "status": "Ready"},
		map[string]any{"kind": "TempoStack", "tempoNamespace": "tracing", "tempoName": "tempo2", "multitenancy": false, "status": "Ready"},
	}, instances)

	t.Log("tempo_list_instances returned successfully")
}

func TestTempoSearchTags(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 23, "tempo_search_tags", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
	})
	if err != nil {
		t.Fatalf("Failed to call tempo_search_tags: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Fatalf("tempo_search_tags returned an error result: %s", resultJSON)
	}

	structured := resp.Result["structuredContent"].(map[string]any)
	scopes := structured["scopes"].([]any)
	require.NotEmpty(t, scopes, "expected at least one scope in search tags response")

	// Verify that service.name is present in the resource scope
	var found bool
	for _, s := range scopes {
		scope := s.(map[string]any)
		if scope["name"] == "resource" {
			tags := scope["tags"].([]any)
			for _, tag := range tags {
				if tag == "service.name" {
					found = true
					break
				}
			}
			break
		}
	}
	require.True(t, found, "expected service.name tag in resource scope")

	t.Log("tempo_search_tags returned successfully")
}

func TestTempoSearchTraces(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 40, "tempo_search_traces", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"query":          "{}",
	})
	if err != nil {
		t.Fatalf("Failed to call tempo_search_traces: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Fatalf("tempo_search_traces returned an error result: %s", resultJSON)
	}

	structured, ok := resp.Result["structuredContent"].(map[string]any)
	require.True(t, ok, "expected structuredContent in result")
	traces, ok := structured["traces"].([]any)
	require.True(t, ok, "expected traces field in structured content")
	require.NotEmpty(t, traces, "expected at least one trace; was trace data ingested before this test ran?")

	t.Logf("tempo_search_traces returned %d traces", len(traces))
}

func TestTempoSearchTraces_EmptyQuery(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 41, "tempo_search_traces", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"query":          "",
	})
	if err != nil {
		t.Fatalf("Failed to call tempo_search_traces: %v", err)
	}
	if resp.Error != nil {
		t.Logf("Correctly returned MCP error for empty query: %s", resp.Error.Message)
		return
	}
	// Empty query must be rejected with isError:true.
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Log("Correctly returned error for empty query")
			return
		}
	}
	t.Error("Expected isError:true for empty query parameter")
}

func TestTempoGetTraceByID(t *testing.T) {
	// First retrieve a real trace ID from the search tool.
	searchResp, err := mcpClient.CallTool(t, 42, "tempo_search_traces", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"query":          "{}",
		"limit":          1,
	})
	if err != nil {
		t.Fatalf("Failed to call tempo_search_traces: %v", err)
	}
	if searchResp.Error != nil {
		t.Fatalf("MCP error during trace search: %s", searchResp.Error.Message)
	}

	structured, ok := searchResp.Result["structuredContent"].(map[string]any)
	if !ok {
		t.Skip("No structuredContent in search response; skipping tempo_get_trace_by_id test")
	}
	traces, ok := structured["traces"].([]any)
	if !ok || len(traces) == 0 {
		t.Skip("No traces available; skipping tempo_get_trace_by_id test")
	}

	firstTrace := traces[0].(map[string]any)
	traceID, ok := firstTrace["traceID"].(string)
	require.True(t, ok, "expected traceID field in first trace")
	require.NotEmpty(t, traceID, "expected non-empty traceID")

	t.Logf("Fetching trace %s", traceID)

	// Now fetch that trace by ID.
	resp, err := mcpClient.CallTool(t, 43, "tempo_get_trace_by_id", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"traceid":        traceID,
	})
	if err != nil {
		t.Fatalf("Failed to call tempo_get_trace_by_id: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Fatalf("tempo_get_trace_by_id returned an error result: %s", resultJSON)
	}

	traceStructured, ok := resp.Result["structuredContent"].(map[string]any)
	require.True(t, ok, "expected structuredContent in result")
	require.NotNil(t, traceStructured["trace"], "expected trace field in response")

	t.Logf("tempo_get_trace_by_id returned trace %s successfully", traceID)
}

func TestTempoGetTraceByID_InvalidID(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 44, "tempo_get_trace_by_id", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"traceid":        "00000000000000000000000000000000",
	})
	if err != nil {
		t.Fatalf("Failed to call tempo_get_trace_by_id: %v", err)
	}
	// A non-existent trace ID must result in isError:true (404 from Tempo backend).
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Log("Correctly returned error for non-existent trace ID")
			return
		}
	}
	if resp.Error != nil {
		t.Logf("Correctly returned MCP error for non-existent trace ID: %s", resp.Error.Message)
		return
	}
	t.Error("Expected error for non-existent trace ID")
}

func TestTempoSearchTagValues(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 45, "tempo_search_tag_values", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
		"tag":            "resource.service.name",
	})
	if err != nil {
		t.Fatalf("Failed to call tempo_search_tag_values: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}
	if isErr, ok := resp.Result["isError"].(bool); ok && isErr {
		resultJSON, _ := json.Marshal(resp.Result)
		t.Fatalf("tempo_search_tag_values returned an error result: %s", resultJSON)
	}

	structured, ok := resp.Result["structuredContent"].(map[string]any)
	require.True(t, ok, "expected structuredContent in result")
	tagValues, ok := structured["tagValues"].(map[string]any)
	require.True(t, ok, "expected tagValues field in structured content")

	// Should contain at least one string value for service.name.
	stringValues, ok := tagValues["string"].([]any)
	require.True(t, ok, "expected string key in tagValues")
	require.NotEmpty(t, stringValues, "expected at least one service name value")

	t.Logf("tempo_search_tag_values returned %d service name values", len(stringValues))
}

func TestTempoSearchTagValues_MissingTag(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 46, "tempo_search_tag_values", map[string]any{
		"tempoNamespace": "tracing",
		"tempoName":      "tempo1",
	})
	if err != nil {
		t.Fatalf("Failed to call tempo_search_tag_values: %v", err)
	}
	if resp.Error != nil {
		t.Logf("Correctly returned MCP error for missing tag: %s", resp.Error.Message)
		return
	}
	// Missing required 'tag' param must return isError:true.
	if resp.Result != nil {
		if isError, ok := resp.Result["isError"].(bool); ok && isError {
			t.Log("Correctly returned error for missing tag parameter")
			return
		}
	}
	t.Error("Expected isError:true for missing tag parameter")
}

func TestOtelcolToolset(t *testing.T) {
	runOtelcolToolsetTests(t, mcpClient)
}
