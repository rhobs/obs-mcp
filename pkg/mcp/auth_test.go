package mcp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"

	promapi "github.com/prometheus/client_golang/api"
)

func TestCreateHeaderAPIConfig(t *testing.T) {
	// This test validates the complete flow: context -> token extraction -> RoundTripper adds Authorization header
	token := "test-bearer-token-12345"

	// Step 1: Create a mock transport to capture the request
	var capturedRequest *http.Request
	mockTransport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			// Capture the request before it's sent
			capturedRequest = req.Clone(req.Context())
			return nil, nil
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Prevent actual network call
			return nil, fmt.Errorf("network call prevented in test")
		},
	}

	// Step 2: Temporarily replace the default RoundTripper
	originalTransport := promapi.DefaultRoundTripper
	promapi.DefaultRoundTripper = mockTransport
	defer func() {
		promapi.DefaultRoundTripper = originalTransport
	}()

	// Step 3: Create an HTTP request with Bearer token and extract token into context
	req, err := http.NewRequest("GET", "/test", http.NoBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set(string(AuthHeaderKey), "Bearer "+token)
	ctx := context.Background()
	ctx = authFromRequest(ctx, req)

	// Step 4: Create API config using the complete production code path
	opts := ObsMCPOptions{
		MetricsBackendURL: "https://prometheus.example.com",
		Insecure:          true,
	}
	apiConfig, err := createHeaderAPIConfig(ctx, opts)
	if err != nil {
		t.Fatalf("failed to create API config: %v", err)
	}

	// Step 5: Create a test request
	testReq, err := http.NewRequest("GET", "https://prometheus.example.com/api/v1/query", http.NoBody)
	if err != nil {
		t.Fatalf("failed to create test request: %v", err)
	}
	testReq.Header.Set("X-Test", "test-value")

	// Step 6: Make the request using the RoundTripper
	// The Proxy function captures the request before DialContext prevents the actual network call
	resp, _ := apiConfig.RoundTripper.RoundTrip(testReq)
	// We ignore the error from DialContext since we only care about the captured request
	if resp != nil && resp.Body != nil {
		resp.Body.Close() // Mainly to make the linter happy.
	}

	// Step 7: Verify the Authorization header was added to the captured request
	authHeader := capturedRequest.Header.Get("Authorization")
	expectedAuthHeader := "Bearer " + token
	if authHeader != expectedAuthHeader {
		t.Errorf("expected Authorization header %q, got %q", expectedAuthHeader, authHeader)
	}

	// Verify the test header is still present
	if capturedRequest.Header.Get("X-Test") != "test-value" {
		t.Error("expected X-Test header to be preserved")
	}
}
