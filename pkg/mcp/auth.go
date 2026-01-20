package mcp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	promapi "github.com/prometheus/client_golang/api"
	promcfg "github.com/prometheus/common/config"
	"k8s.io/client-go/rest"

	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

// AuthMode defines the authentication mode for Prometheus client
type AuthMode string

const (
	AuthModeKubeConfig     AuthMode = "kubeconfig"
	AuthModeServiceAccount AuthMode = "serviceaccount"
	AuthModeHeader         AuthMode = "header"
)

const (
	defaultServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	defaultServiceAccountCAPath    = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
)

type ContextKey string

const (
	// AuthHeaderKey is the context key for the Kubernetes authorization header
	AuthHeaderKey ContextKey = "kubernetes-authorization"

	// TestPromClientKey is the context key for injecting a test Prometheus client
	TestPromClientKey ContextKey = "test-prometheus-client"
)

// ParseAuthMode validates and converts a string to AuthMode
func ParseAuthMode(mode string) (AuthMode, error) {
	switch mode {
	case string(AuthModeKubeConfig):
		return AuthModeKubeConfig, nil
	case string(AuthModeServiceAccount):
		return AuthModeServiceAccount, nil
	case string(AuthModeHeader):
		return AuthModeHeader, nil
	default:
		return "", fmt.Errorf("invalid auth mode: %s (valid options: kubeconfig, serviceaccount, header)", mode)
	}
}

func getPromClient(ctx context.Context, opts ObsMCPOptions) (prometheus.Loader, error) {
	// Check if a test client was injected via context
	if testClient := ctx.Value(TestPromClientKey); testClient != nil {
		if client, ok := testClient.(prometheus.Loader); ok {
			return client, nil
		}
	}

	// Normal production path

	apiConfig, err := createAPIConfig(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %v", err)
	}

	promClient, err := prometheus.NewPrometheusClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %v", err)
	}

	promClient.WithGuardrails(opts.Guardrails)

	return promClient, nil
}

func createAPIConfig(ctx context.Context, opts ObsMCPOptions) (promapi.Config, error) {
	switch opts.AuthMode {
	case AuthModeKubeConfig:
		return createKubeconfigAPIConfig(opts)
	case AuthModeServiceAccount:
		return createServiceAccountAPIConfig(opts)
	case AuthModeHeader:
		return createHeaderAPIConfig(ctx, opts)
	default:
		return promapi.Config{}, fmt.Errorf("unsupported auth mode: %s", opts.AuthMode)
	}
}

func createKubeconfigAPIConfig(opts ObsMCPOptions) (promapi.Config, error) {
	// Get kubeconfig-based transport
	restConfig, err := k8s.GetClientConfig()
	if err != nil {
		return promapi.Config{}, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	if restConfig.BearerToken == "" {
		return promapi.Config{}, fmt.Errorf("kubeconfig doesn't contain a bearer token for Prometheus authentication")
	}

	// For routes/ingresses, we need to configure TLS to skip verification
	// or use the system's CA pool, since routes typically use different
	// certificates than the Kubernetes API
	tlsConfig := rest.TLSClientConfig{Insecure: opts.Insecure}
	restConfig.TLSClientConfig = tlsConfig

	// Create HTTP client with kubeconfig authentication
	rt, err := rest.TransportFor(restConfig)
	if err != nil {
		return promapi.Config{}, fmt.Errorf("failed to create transport from kubeconfig: %w", err)
	}

	return promapi.Config{
		Address:      opts.PromURL,
		RoundTripper: rt,
	}, nil
}

func createServiceAccountAPIConfig(opts ObsMCPOptions) (promapi.Config, error) {
	slog.Info("Using service account token for authentication")
	tokenBytes, err := readTokenFromFile()
	if err != nil {
		slog.Error("Failed to read the service account token", "err", err)
		return promapi.Config{}, err
	}
	token := string(tokenBytes)

	return createAPIConfigWithToken(opts.PromURL, token, opts.Insecure)
}

func createHeaderAPIConfig(ctx context.Context, opts ObsMCPOptions) (promapi.Config, error) {
	token := getTokenFromCtx(ctx)
	if token == "" {
		slog.Warn("No token provided in context for header auth mode")
	}

	return createAPIConfigWithToken(opts.PromURL, token, opts.Insecure)
}

func createAPIConfigWithToken(prometheusURL, token string, insecure bool) (promapi.Config, error) {
	apiConfig := promapi.Config{
		Address: prometheusURL,
	}

	useTLS := strings.HasPrefix(prometheusURL, "https://")
	if useTLS {
		defaultRt := promapi.DefaultRoundTripper.(*http.Transport)

		if insecure {
			defaultRt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			certs, err := createCertPool()
			if err != nil {
				return promapi.Config{}, err
			}
			defaultRt.TLSClientConfig = &tls.Config{RootCAs: certs}
		}

		if token != "" {
			apiConfig.RoundTripper = promcfg.NewAuthorizationCredentialsRoundTripper(
				"Bearer", promcfg.NewInlineSecret(token), defaultRt)
		} else {
			apiConfig.RoundTripper = defaultRt
		}
	} else {
		slog.Warn("Connecting to Prometheus without TLS")
	}

	return apiConfig, nil
}

func getTokenFromCtx(ctx context.Context) string {
	k8sToken := ctx.Value(AuthHeaderKey)
	if k8sToken == nil {
		slog.Warn("No token provided in context.")
		return ""
	}
	k8TokenStr, ok := k8sToken.(string)
	if !ok {
		slog.Warn("Couldn't parse token... ignoring.")
		return ""
	}
	return k8TokenStr
}

func createCertPool() (*x509.CertPool, error) {
	certs := x509.NewCertPool()

	pemData, err := os.ReadFile(defaultServiceAccountCAPath)
	if err != nil {
		slog.Error("Failed to read the CA certificate", "err", err)
		return nil, err
	}
	certs.AppendCertsFromPEM(pemData)
	return certs, nil
}

func readTokenFromFile() ([]byte, error) {
	return os.ReadFile(defaultServiceAccountTokenPath)
}
