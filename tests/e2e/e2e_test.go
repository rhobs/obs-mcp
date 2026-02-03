//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	obsMCPURL   = "http://localhost:9100"
	mcpEndpoint = "/mcp"
	testTimeout = 30 * time.Second
)

var obsMCPPortForwardCmd *exec.Cmd

func TestMain(m *testing.M) {
	// Setup: start port-forward for obs-mcp
	var err error
	obsMCPPortForwardCmd, err = startPortForward("obs-mcp", "svc/obs-mcp", 9100, 9100)
	if err != nil {
		fmt.Printf("Failed to start port-forward: %v\n", err)
		os.Exit(1)
	}

	if err := waitForReady(obsMCPURL+"/health", 30*time.Second); err != nil {
		fmt.Printf("Failed waiting for obs-mcp: %v\n", err)
		stopPortForward(obsMCPPortForwardCmd)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown: stop port-forward
	stopPortForward(obsMCPPortForwardCmd)

	os.Exit(code)
}

func startPortForward(namespace, resource string, localPort, remotePort int) (*exec.Cmd, error) {
	cmd := exec.Command("kubectl", "port-forward",
		"-n", namespace,
		resource,
		fmt.Sprintf("%d:%d", localPort, remotePort),
	)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start port-forward for %s/%s: %w", namespace, resource, err)
	}
	return cmd, nil
}

func waitForReady(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s to be ready", url)
		case <-ticker.C:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

func stopPortForward(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait() // Reap the process to avoid zombies
	}
}

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int            `json:"id"`
	Result  map[string]any `json:"result,omitempty"`
	Error   *MCPError      `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func sendMCPRequest(t *testing.T, req MCPRequest) (*MCPResponse, error) {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, obsMCPURL+mcpEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var mcpResp MCPResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w (body: %s)", err, string(respBody))
	}

	return &mcpResp, nil
}

func TestHealthEndpoint(t *testing.T) {
	resp, err := http.Get(obsMCPURL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestListMetrics(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "list_metrics",
			"arguments": map[string]any{},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call list_metrics: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some metrics back
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("list_metrics returned successfully")
}

func TestListMetricsReturnsKnownMetrics(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "list_metrics",
			"arguments": map[string]any{},
		},
	}

	resp, err := sendMCPRequest(t, req)
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "execute_range_query",
			"arguments": map[string]any{
				"query":    `up{job="prometheus"}`,
				"step":     "1m",
				"duration": "5m",
				"end":      "NOW",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call execute_range_query: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	t.Logf("execute_range_query returned successfully")
}

func TestRangeQueryWithInvalidPromQL(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "execute_range_query",
			"arguments": map[string]any{
				"query":    `up{{{invalid`, // Invalid PromQL syntax
				"step":     "1m",
				"duration": "5m",
				"end":      "NOW",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "execute_range_query",
			"arguments": map[string]any{
				// Missing "query" parameter
				"step":     "1m",
				"duration": "5m",
				"end":      "NOW",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "execute_range_query",
			"arguments": map[string]any{
				"query":    `nonexistent_metric_xyz{job="test"}`,
				"step":     "1m",
				"duration": "5m",
				"end":      "NOW",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "execute_range_query",
			"arguments": map[string]any{
				"query":    `{__name__=~".+"}`, // Dangerous: selects all metrics
				"step":     "1m",
				"duration": "5m",
				"end":      "NOW",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      8,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "execute_instant_query",
			"arguments": map[string]any{
				"query": `up{job="prometheus"}`,
				"time":  "NOW",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call execute_instant_query: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got result
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("execute_instant_query returned successfully")
}

func TestInstantQueryWithInvalidPromQL(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      9,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "execute_instant_query",
			"arguments": map[string]any{
				"query": `up{{{invalid`, // Invalid PromQL syntax
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "get_label_names",
			"arguments": map[string]any{
				"metric": "up",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call get_label_names: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some labels back
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      11,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "get_label_names",
			"arguments": map[string]any{},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call get_label_names: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some labels back
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_label_names (all metrics) returned successfully")
}

func TestGetLabelValues(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      12,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "get_label_values",
			"arguments": map[string]any{
				"label":  "job",
				"metric": "up",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call get_label_values: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some values back
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      13,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "get_label_values",
			"arguments": map[string]any{
				// Missing "label" parameter
				"metric": "up",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      14,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "get_series",
			"arguments": map[string]any{
				"matches": `up{job="prometheus"}`,
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call get_series: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some series back
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      15,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "get_series",
			"arguments": map[string]any{
				// Missing "matches" parameter
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
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
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      16,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "get_alerts",
			"arguments": map[string]any{},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call get_alerts: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some result back
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_alerts returned successfully")
}

func TestGetAlertsWithActiveFilter(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      17,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "get_alerts",
			"arguments": map[string]any{
				"active": true,
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call get_alerts with active filter: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some result back
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_alerts with active filter returned successfully")
}

func TestGetAlertsWithFilter(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      18,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "get_alerts",
			"arguments": map[string]any{
				"filter": "alertname=Watchdog",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call get_alerts with filter: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some result back
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_alerts with filter returned successfully")
}

func TestGetSilences(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      19,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "get_silences",
			"arguments": map[string]any{},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call get_silences: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some result back
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_silences returned successfully")
}

func TestGetSilencesWithFilter(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      20,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "get_silences",
			"arguments": map[string]any{
				"filter": "alertname=Watchdog",
			},
		},
	}

	resp, err := sendMCPRequest(t, req)
	if err != nil {
		t.Fatalf("Failed to call get_silences with filter: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("MCP error: %s", resp.Error.Message)
	}

	// Verify we got some result back
	if resp.Result == nil {
		t.Error("Expected result, got nil")
	}

	t.Logf("get_silences with filter returned successfully")
}
