package mcp

import (
	"context"
	"fmt"
	"strings"

	promapi "github.com/prometheus/client_golang/api"

	"github.com/rhobs/obs-mcp/pkg/alertmanager"
	"github.com/rhobs/obs-mcp/pkg/auth"
	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

type ContextKey string

const (
	// TestPromClientKey is the context key for injecting a test Prometheus client
	TestPromClientKey ContextKey = "test-prometheus-client"

	// TestAlertmanagerClientKey is the context key for injecting a test Alertmanager client
	TestAlertmanagerClientKey ContextKey = "test-alertmanager-client"
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

	amClient, err := alertmanager.NewAlertmanagerClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alertmanager client: %w", err)
	}

	return amClient, nil
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
