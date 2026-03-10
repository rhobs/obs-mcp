//go:build e2e && openshift

package e2e

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"k8s.io/client-go/rest"

	"github.com/rhobs/obs-mcp/pkg/k8s"
)

// TestMain for the OpenShift test suite. Route discovery tests call pkg/k8s
// directly. TestOpenShiftMetricsPresent connects to obs-mcp only when
// OBS_MCP_URL is set; it skips otherwise.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// authenticatedHTTPClient returns an HTTP client that sends a bearer token on
// every request and skips TLS verification for OpenShift ingress certs.
//
// It prefers OPENSHIFT_TOKEN (set by CI step scripts via
// `oc create token prometheus-k8s -n openshift-monitoring`)
// because the CI kubeconfig uses client certificates which the OAuth proxy
// on monitoring routes does not accept (returns 401).
// Falls back to the kubeconfig bearer token for local development.
func authenticatedHTTPClient(t *testing.T) *http.Client {
	t.Helper()

	if token := os.Getenv("OPENSHIFT_TOKEN"); token != "" {
		t.Log("Using OPENSHIFT_TOKEN for route authentication")
		return &http.Client{
			Transport: &tokenRoundTripper{
				token: token,
				base: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
				},
			},
		}
	}

	restConfig, err := k8s.GetClientConfig()
	if err != nil {
		t.Fatalf("Failed to get kubeconfig: %v", err)
	}
	restConfig.TLSClientConfig = rest.TLSClientConfig{Insecure: true} //nolint:gosec
	rt, err := rest.TransportFor(restConfig)
	if err != nil {
		t.Fatalf("Failed to create authenticated transport: %v", err)
	}
	return &http.Client{Transport: rt}
}

// tokenRoundTripper injects a bearer token into every outgoing request.
type tokenRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (t *tokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

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
