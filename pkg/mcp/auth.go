package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	promapi "github.com/prometheus/client_golang/api"

	"github.com/rhobs/obs-mcp/pkg/alertmanager"
	"github.com/rhobs/obs-mcp/pkg/auth"
	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/logs/loki"
	"github.com/rhobs/obs-mcp/pkg/metrics"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	tempoclient "github.com/rhobs/obs-mcp/pkg/traces/tempo"
)

type ContextKey string

const (
	// TestPromClientKey is the context key for injecting a test Prometheus client
	TestPromClientKey ContextKey = "test-prometheus-client"

	// TestAlertmanagerClientKey is the context key for injecting a test Alertmanager client
	TestAlertmanagerClientKey ContextKey = "test-alertmanager-client"

	// TestLokiClientKey is the context key for injecting a test Loki client
	TestLokiClientKey ContextKey = "test-loki-client"
)

func getPromClient(ctx context.Context, opts ObsMCPOptions) (prometheus.Loader, error) {
	// Check if a test client was injected via context
	if testClient := ctx.Value(TestPromClientKey); testClient != nil {
		if client, ok := testClient.(prometheus.Loader); ok {
			return client, nil
		}
	}

	// Normal production path

	apiConfig, err := createAPIConfig(ctx, opts, opts.MetricsBackendURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	// Instrument the RoundTripper for Prometheus client
	if opts.clientMetrics != nil && apiConfig.RoundTripper != nil {
		apiConfig.RoundTripper = metrics.InstrumentedRoundTripper(apiConfig.RoundTripper, opts.clientMetrics, "prometheus")
	}

	promClient, err := prometheus.NewPrometheusLoader(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	promClient.WithGuardrails(opts.Guardrails)

	return promClient, nil
}

func getAlertmanagerClient(ctx context.Context, opts ObsMCPOptions) (alertmanager.Loader, error) {
	// Check if a test client was injected via context
	if testClient := ctx.Value(TestAlertmanagerClientKey); testClient != nil {
		if client, ok := testClient.(alertmanager.Loader); ok {
			return client, nil
		}
	}

	apiConfig, err := createAPIConfig(ctx, opts, opts.AlertmanagerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	// Update the address to use AlertmanagerURL instead of MetricsBackendURL
	apiConfig.Address = opts.AlertmanagerURL

	// Instrument the RoundTripper for Alertmanager client
	if opts.clientMetrics != nil && apiConfig.RoundTripper != nil {
		apiConfig.RoundTripper = metrics.InstrumentedRoundTripper(apiConfig.RoundTripper, opts.clientMetrics, "alertmanager")
	}

	amClient, err := alertmanager.NewAlertmanagerClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alertmanager client: %w", err)
	}

	return amClient, nil
}

func getTempoHTTPClient(ctx context.Context, opts ObsMCPOptions, url string) (tempoclient.Loader, error) {
	tempoOpts := ObsMCPOptions{
		AuthMode:          opts.AuthMode,
		MetricsBackendURL: url,
		Insecure:          opts.Insecure,
		clientMetrics:     opts.clientMetrics,
	}

	apiConfig, err := createAPIConfig(ctx, tempoOpts, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	// Instrument the RoundTripper for Tempo client
	rt := apiConfig.RoundTripper
	if opts.clientMetrics != nil && rt != nil {
		rt = metrics.InstrumentedRoundTripper(rt, opts.clientMetrics, "tempo")
	}

	httpClient := &http.Client{
		Timeout:   tempoclient.RequestTimeout,
		Transport: rt,
	}
	return tempoclient.NewTempoLoader(httpClient, url), nil
}

func getLokiClient(ctx context.Context, opts ObsMCPOptions, url, tenant string) (loki.Loader, error) {
	if testClient := ctx.Value(TestLokiClientKey); testClient != nil {
		if client, ok := testClient.(loki.Loader); ok {
			return client, nil
		}
	}

	if url == "" {
		url = opts.LokiURL
	}

	lokiOpts := ObsMCPOptions{
		AuthMode:      opts.AuthMode,
		MetricsBackendURL: url,
		Insecure:      opts.Insecure,
		clientMetrics: opts.clientMetrics,
	}
	apiConfig, err := createAPIConfig(ctx, lokiOpts, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create API config: %w", err)
	}

	// Instrument the RoundTripper for Loki client
	if opts.clientMetrics != nil && apiConfig.RoundTripper != nil {
		apiConfig.RoundTripper = metrics.InstrumentedRoundTripper(apiConfig.RoundTripper, opts.clientMetrics, "loki")
	}

	lokiClient, err := loki.NewLoader(apiConfig, tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to create Loki client: %w", err)
	}
	return lokiClient, nil
}

func createAPIConfig(ctx context.Context, opts ObsMCPOptions, url string) (promapi.Config, error) {
	restConfig, err := k8s.GetClientConfig()
	if err != nil {
		return promapi.Config{}, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	tls := strings.HasPrefix(url, "https://")
	rt, err := auth.BuildRoundTripper(ctx, restConfig, opts.AuthMode, tls, opts.Insecure)
	if err != nil {
		return promapi.Config{}, fmt.Errorf("failed to create round tripper: %w", err)
	}

	return promapi.Config{
		Address:      url,
		RoundTripper: rt,
	}, nil
}
