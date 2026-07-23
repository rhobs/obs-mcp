package korrel8r

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	korrel8rmcp "github.com/korrel8r/korrel8r/pkg/mcp"

	"github.com/rhobs/obs-mcp/pkg/auth"
	"github.com/rhobs/obs-mcp/pkg/k8s"
)

const (
	ToolsetName  = "observability/korrel8r"
	Instructions = korrel8rmcp.Instructions
)

var AddTools = korrel8rmcp.AddTools

type Config struct {
	AuthMode    auth.AuthMode
	Insecure    bool
	Korrel8rURL string
}

// NewClient creates a korrel8r REST API client with authentication configured from cfg.
func NewClient(cfg *Config) (*korrel8rmcp.Client, error) {
	if cfg.Korrel8rURL == "" {
		return nil, fmt.Errorf("korrel8r URL is required; set --korrel8r-url or KORREL8R_URL")
	}
	useTLS := strings.HasPrefix(cfg.Korrel8rURL, "https://")
	rt, err := newAuthTransport(cfg, useTLS)
	if err != nil {
		return nil, err
	}
	return korrel8rmcp.NewClient(cfg.Korrel8rURL, &http.Client{Transport: rt}), nil
}

// newAuthTransport creates an http.RoundTripper that handles auth for all modes.
// For kubeconfig/serviceaccount the token is static and baked into the transport.
// For header mode the token varies per request, so we wrap with contextAuthTransport.
func newAuthTransport(cfg *Config, useTLS bool) (http.RoundTripper, error) {
	if cfg.AuthMode == auth.AuthModeHeader {
		base, err := baseTLSTransport(cfg.Insecure, useTLS)
		if err != nil {
			return nil, err
		}
		return &contextAuthTransport{base: base}, nil
	}
	// Static token modes (kubeconfig, serviceaccount): BuildRoundTripper reads
	// the token once and handles TLS/insecure.
	restConfig, err := k8s.GetClientConfig()
	if err != nil {
		return nil, err
	}
	return auth.BuildRoundTripper(context.Background(), restConfig, cfg.AuthMode, useTLS, cfg.Insecure)
}

// baseTLSTransport returns a transport with TLS configured but no auth token.
func baseTLSTransport(insecure, useTLS bool) (http.RoundTripper, error) {
	base := http.DefaultTransport.(*http.Transport).Clone()
	if useTLS && insecure {
		base.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}
	} else if useTLS {
		restConfig, err := k8s.GetClientConfig()
		if err != nil {
			return nil, err
		}
		certs, err := auth.CertPoolFromRESTConfig(restConfig)
		if err != nil {
			return nil, err
		}
		base.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    certs,
		}
	}
	return base, nil
}

// contextAuthTransport reads the bearer token from the request context on each call,
// bridging the obs-mcp auth middleware to outgoing korrel8r HTTP requests.
type contextAuthTransport struct {
	base http.RoundTripper
}

func (t *contextAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token := auth.ReadTokenFromContext(req.Context())
	if token != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return t.base.RoundTrip(req)
}
