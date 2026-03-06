//go:build e2e && openshift

package e2e

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/rhobs/obs-mcp/pkg/k8s"
)

// assertValidRouteURL checks that a discovered route URL is well-formed:
// - parseable by net/url
// - scheme is "https"
// - host is non-empty and contains a dot (i.e. not just a bare word)
func assertValidRouteURL(t *testing.T, raw string) {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Errorf("URL is not parseable: %s (%v)", raw, err)
		return
	}
	if parsed.Scheme != "https" {
		t.Errorf("Expected scheme 'https', got %q in URL: %s", parsed.Scheme, raw)
	}
	if parsed.Host == "" {
		t.Errorf("URL has no host: %s", raw)
	}
	if !strings.Contains(parsed.Host, ".") {
		t.Errorf("URL host looks invalid (no dot): %s", parsed.Host)
	}
}

// Route discovery tests below exercise pkg/k8s directly using the kubeconfig
// available to the test runner. They validate the auto-discovery path used when
// obs-mcp runs locally with --auth-mode kubeconfig. The deployed server in CI
// uses --auth-mode serviceaccount with URLs hardcoded in the configmap instead.

// TestRouteDiscovery_ThanosQuerier verifies that the thanos-querier route in
// openshift-monitoring can be discovered and returns a valid https:// URL.
func TestRouteDiscovery_ThanosQuerier(t *testing.T) {
	discoveredURL, err := k8s.GetMetricsBackendURL(k8s.MetricsBackendThanos)
	if err != nil {
		t.Fatalf("Failed to discover thanos-querier route: %v", err)
	}
	assertValidRouteURL(t, discoveredURL)
	t.Logf("Discovered Thanos URL: %s", discoveredURL)
}

// TestRouteDiscovery_PrometheusK8s verifies that the prometheus-k8s route in
// openshift-monitoring can be discovered when using the prometheus backend.
func TestRouteDiscovery_PrometheusK8s(t *testing.T) {
	discoveredURL, err := k8s.GetMetricsBackendURL(k8s.MetricsBackendPrometheus)
	if err != nil {
		t.Fatalf("Failed to discover prometheus-k8s route: %v", err)
	}
	assertValidRouteURL(t, discoveredURL)
	t.Logf("Discovered Prometheus URL: %s", discoveredURL)
}

// TestRouteDiscovery_Alertmanager verifies that the alertmanager-main route in
// openshift-monitoring can be discovered and returns a valid https:// URL.
func TestRouteDiscovery_Alertmanager(t *testing.T) {
	discoveredURL, err := k8s.GetAlertmanagerURL()
	if err != nil {
		t.Fatalf("Failed to discover alertmanager-main route: %v", err)
	}
	assertValidRouteURL(t, discoveredURL)
	t.Logf("Discovered Alertmanager URL: %s", discoveredURL)
}

// TestRouteDiscovery_URLsAreReachable verifies that the discovered route URLs
// respond to HTTP requests. A 401/403 is acceptable -- it means the endpoint
// exists and auth is enforced. Only connection failures are treated as errors.
func TestRouteDiscovery_URLsAreReachable(t *testing.T) {
	tests := []struct {
		name   string
		getURL func() (string, error)
	}{
		{
			name:   "thanos-querier",
			getURL: func() (string, error) { return k8s.GetMetricsBackendURL(k8s.MetricsBackendThanos) },
		},
		{
			name:   "alertmanager-main",
			getURL: k8s.GetAlertmanagerURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawURL, err := tt.getURL()
			if err != nil {
				t.Fatalf("Route discovery failed: %v", err)
			}
			assertValidRouteURL(t, rawURL)

			resp, err := http.Get(rawURL) //nolint:noctx
			if err != nil {
				t.Fatalf("Route %s (%s) is not reachable: %v", tt.name, rawURL, err)
			}
			defer resp.Body.Close()

			t.Logf("Route %s responded with HTTP %d", tt.name, resp.StatusCode)
		})
	}
}

// TestOpenShiftMetricsPresent is a sanity test that confirms obs-mcp is wired
// to OpenShift in-cluster monitoring and not an empty or misconfigured backend.
// It checks for cluster_version, a metric that only exists in OpenShift monitoring
// and is absent from Kind/kube-prometheus environments.
func TestOpenShiftMetricsPresent(t *testing.T) {
	resp, err := mcpClient.CallTool(t, 100, "list_metrics", map[string]any{
		"name_regex": "cluster_version",
	})
	if err != nil {
		t.Fatalf("Failed to call list_metrics: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("MCP error: %s", resp.Error.Message)
	}

	resultJSON, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(resultJSON), "cluster_version") {
		t.Error("Expected OpenShift-specific metric 'cluster_version' not found -- is obs-mcp pointing at OpenShift monitoring?")
	}
	t.Logf("OpenShift metric 'cluster_version' confirmed present")
}
