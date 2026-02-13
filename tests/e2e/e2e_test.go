//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
)

var (
	testConfig *TestConfig
	mcpClient  *MCPClient
)

func TestMain(m *testing.M) {
	// Set up signal handler for graceful shutdown on Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, cleaning up...")
		cancel()
		if testConfig != nil {
			testConfig.Cleanup()
		}
		os.Exit(130) // Standard exit code for SIGINT
	}()

	testConfig = NewTestConfig()
	if err := testConfig.Setup(ctx); err != nil {
		fmt.Printf("Failed to setup test environment: %v\n", err)
		os.Exit(1)
	}

	mcpClient = NewMCPClient(testConfig.MCPURL)

	// Run tests
	code := m.Run()

	// Cleanup
	testConfig.Cleanup()

	os.Exit(code)
}

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

func TestListMetrics(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 1, "list_metrics", map[string]any{
		"name_regex": ".*",
	})
	if err != nil {
		t.Fatalf("Failed to call list_metrics: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("list_metrics returned successfully")
}

func TestListMetricsReturnsKnownMetrics(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 2, "list_metrics", map[string]any{
		"name_regex": ".*",
	})
	if err != nil {
		t.Fatalf("Failed to call list_metrics: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}

	// Verify known metrics from kube-prometheus are present
	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)

	expectedMetrics := []string{"up", "prometheus_build_info"}
	for _, metric := range expectedMetrics {
		if !strings.Contains(resultStr, metric) {
			t.Errorf("Expected metric %q not found in results", metric)
		}
	}
}

func TestExecuteRangeQuery(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 3, "execute_range_query", map[string]any{
		"query":    `up{job="prometheus"}`,
		"step":     "1m",
		"duration": "5m",
		"end":      "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_range_query: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
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
	resp, err := mcpClient.CallTool(t, 8, "execute_instant_query", map[string]any{
		"query": `up{job="prometheus"}`,
		"time":  "NOW",
	})
	if err != nil {
		t.Fatalf("Failed to call execute_instant_query: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	if resp.Result == nil {
		t.Error("Expected result, got nil")
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

	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	// Verify Watchdog alert structure (kube-prometheus always has Watchdog firing)
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

	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_silences with filter returned successfully")
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

	// Should succeed but may return empty alerts array
	t.Log("Query for non-existent alert handled correctly")
}
