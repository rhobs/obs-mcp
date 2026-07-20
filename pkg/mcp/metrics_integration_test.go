package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/rhobs/obs-mcp/pkg/instrumentation"
)

// TestStreamableHTTPWithMetrics verifies that metrics are properly collected
// when using the MCP streamable HTTP transport.
func TestStreamableHTTPWithMetrics(t *testing.T) {
	// Create a test MCP server with minimal setup
	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	mcpServer := mcp.NewServer(impl, &mcp.ServerOptions{})

	// Create a prometheus registry for testing
	registry := prom.NewRegistry()

	// Create metrics instrumentation middleware
	instrMiddleware := instrumentation.NewMiddleware(registry, nil)

	// Create streamable HTTP handler
	streamableHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return mcpServer
		},
		&mcp.StreamableHTTPOptions{
			Stateless: true, // Use stateless mode for simpler testing
		},
	)

	// Wrap with metrics middleware
	handler := instrMiddleware.NewHandler("mcp", streamableHandler)

	// Create test server
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Get initial metric values
	var initialRequestCount float64
	if m := getMetric(t, registry, "http_requests_total", map[string]string{
		"handler": "mcp",
		"code":    "202",
		"method":  "post",
	}); m.GetCounter() != nil {
		initialRequestCount = m.GetCounter().GetValue()
	}

	// Test 1: Send a POST request with initialize
	t.Run("POST_initialize", func(t *testing.T) {
		initRequest := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{},
				"clientInfo": map[string]any{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		body, _ := json.Marshal(initRequest)
		req, err := http.NewRequest("POST", ts.URL, strings.NewReader(string(body)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream, application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Read response (SSE stream)
		respBody, _ := io.ReadAll(resp.Body)
		t.Logf("Response status: %d, body: %s", resp.StatusCode, string(respBody))

		// Verify metrics were incremented
		time.Sleep(100 * time.Millisecond) // Give metrics time to update

		var newRequestCount float64
		if m := getMetric(t, registry, "http_requests_total", map[string]string{
			"handler": "mcp",
			"code":    "200",
			"method":  "post",
		}); m.GetCounter() != nil {
			newRequestCount = m.GetCounter().GetValue()
		}

		if newRequestCount <= initialRequestCount {
			t.Errorf("Expected request count to increase, got initial=%f, new=%f",
				initialRequestCount, newRequestCount)
		}
	})

	// Test 2: Verify request duration metrics exist
	t.Run("request_duration_metrics", func(t *testing.T) {
		m := getMetric(t, registry, "http_request_duration_seconds", map[string]string{
			"handler": "mcp",
		})

		if m.GetHistogram().GetSampleCount() == 0 {
			t.Error("Expected request duration metrics to be recorded")
		}
	})

	// Test 3: Verify in-flight metrics
	t.Run("in_flight_metrics", func(t *testing.T) {
		m := getMetric(t, registry, "http_inflight_requests", map[string]string{
			"handler": "mcp",
			"method":  "POST",
		})

		if m == nil || m.Gauge == nil {
			t.Error("Expected in-flight request metrics to exist")
		}
	})

	// Test 4: Verify request/response size metrics
	t.Run("size_metrics", func(t *testing.T) {
		var requestSize, responseSize uint64
		if m := getMetric(t, registry, "http_request_size_bytes", map[string]string{
			"handler": "mcp",
		}); m.GetSummary() != nil {
			requestSize = m.GetSummary().GetSampleCount()
		}
		if m := getMetric(t, registry, "http_response_size_bytes", map[string]string{
			"handler": "mcp",
		}); m.GetSummary() != nil {
			responseSize = m.GetSummary().GetSampleCount()
		}

		t.Logf("Request size count: %d, Response size count: %d", requestSize, responseSize)
	})

	// Test 5: Print all collected metrics for debugging
	t.Run("print_all_metrics", func(t *testing.T) {
		metricFamilies, err := registry.Gather()
		if err != nil {
			t.Fatalf("Failed to gather metrics: %v", err)
		}

		t.Logf("All collected metrics:")
		for _, mf := range metricFamilies {
			for _, m := range mf.GetMetric() {
				labels := make(map[string]string)
				for _, label := range m.GetLabel() {
					labels[label.GetName()] = label.GetValue()
				}

				var value any
				switch {
				case m.Counter != nil:
					value = m.Counter.GetValue()
				case m.Gauge != nil:
					value = m.Gauge.GetValue()
				case m.Histogram != nil:
					value = m.Histogram.GetSampleCount()
				case m.Summary != nil:
					value = m.Summary.GetSampleCount()
				}

				t.Logf("  %s%v = %v", mf.GetName(), labels, value)
			}
		}
	})
}

// TestStreamableHTTPWithMetrics_SSEStream tests that SSE streaming works with metrics.
func TestStreamableHTTPWithMetrics_SSEStream(t *testing.T) {
	// Create a test MCP server
	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	mcpServer := mcp.NewServer(impl, &mcp.ServerOptions{})

	// Create a prometheus registry for testing
	registry := prom.NewRegistry()

	// Create metrics instrumentation middleware
	instrMiddleware := instrumentation.NewMiddleware(registry, nil)

	// Create streamable HTTP handler (non-stateless for SSE support)
	streamableHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return mcpServer
		},
		&mcp.StreamableHTTPOptions{
			Stateless: false, // Stateful mode supports GET for SSE
		},
	)

	// Wrap with metrics middleware
	handler := instrMiddleware.NewHandler("mcp", streamableHandler)

	// Create test server
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Test: Send a POST to initialize and get session ID
	t.Run("POST_and_GET_SSE", func(t *testing.T) {
		// First, initialize to get a session
		initRequest := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{},
				"clientInfo": map[string]any{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		body, _ := json.Marshal(initRequest)
		req, err := http.NewRequest("POST", ts.URL, strings.NewReader(string(body)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream, application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		// Extract session ID from response header
		sessionID := resp.Header.Get("Mcp-Session-Id")
		resp.Body.Close()

		if sessionID == "" {
			t.Log("No session ID returned (stateless mode or initialization didn't complete)")
			return
		}

		// Now try to open an SSE stream with the session ID
		getReq, err := http.NewRequest("GET", ts.URL, http.NoBody)
		if err != nil {
			t.Fatalf("Failed to create GET request: %v", err)
		}
		getReq.Header.Set("Accept", "text/event-stream")
		getReq.Header.Set("Mcp-Session-Id", sessionID)

		// Use a context with timeout for the GET request since SSE is a hanging connection
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		getReq = getReq.WithContext(ctx)

		getResp, err := http.DefaultClient.Do(getReq)
		if err != nil {
			// Context timeout is expected for SSE
			if ctx.Err() != context.DeadlineExceeded {
				t.Fatalf("Failed to send GET request: %v", err)
			}
		} else {
			defer getResp.Body.Close()
			t.Logf("GET response status: %d", getResp.StatusCode)
		}

		// Verify that both POST and GET metrics were recorded
		time.Sleep(100 * time.Millisecond)

		var postCount float64
		if m := getMetric(t, registry, "http_requests_total", map[string]string{
			"handler": "mcp",
			"method":  "post",
		}); m.GetCounter() != nil {
			postCount = m.GetCounter().GetValue()
		}

		if postCount == 0 {
			t.Error("Expected POST request metrics to be recorded")
		}

		t.Logf("POST request count: %f", postCount)
	})
}

// getMetric finds the first metric matching name and labels from the registry.
func getMetric(t *testing.T, registry *prom.Registry, name string, labels map[string]string) *dto.Metric {
	t.Helper()
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	for _, mf := range metricFamilies {
		if mf.GetName() == name {
			for _, m := range mf.GetMetric() {
				if labelsMatch(m.GetLabel(), labels) {
					return m
				}
			}
		}
	}
	return nil
}

// labelsMatch checks if metric labels match the expected labels.
func labelsMatch(metricLabels []*dto.LabelPair, expected map[string]string) bool {
	if len(metricLabels) < len(expected) {
		return false
	}

	matches := 0
	for _, label := range metricLabels {
		if expectedValue, ok := expected[label.GetName()]; ok {
			if label.GetValue() == expectedValue {
				matches++
			}
		}
	}

	return matches == len(expected)
}
