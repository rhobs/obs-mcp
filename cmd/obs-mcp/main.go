package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/prometheus/common/promslog"

	"github.com/mark3labs/mcp-go/server"

	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/mcp"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

const (
	defaultPrometheusURL   = "http://localhost:9090"
	defaultAlertmanagerURL = "http://localhost:9093"
)

func main() {
	// Parse command line flags
	var listen = flag.String("listen", "", "Listen address for HTTP mode (e.g., :9100, 127.0.0.1:8080)")
	var authMode = flag.String("auth-mode", "", "Authentication mode: kubeconfig, serviceaccount, or header")
	var insecure = flag.Bool("insecure", false, "Skip TLS certificate verification")
	var logLevel = flag.String("log-level", "info", "Log level: debug, info, warn, error")
	var metricsBackend = flag.String("metrics-backend", "thanos", "Metrics backend: thanos (default, with prometheus fallback) or prometheus (strict, no fallback)")
	var guardrails = flag.String("guardrails", "all", "Guardrails configuration: 'all' (default), 'none', or comma-separated list of guardrails to enable (disallow-explicit-name-label, require-label-matcher, disallow-blanket-regex)")
	var maxMetricCardinality = flag.Uint64("guardrails.max-metric-cardinality", 20000, "Maximum allowed series count per metric (0 = disabled)")
	var maxLabelCardinality = flag.Uint64("guardrails.max-label-cardinality", 500, "Maximum allowed label value count for blanket regex (0 = always disallow blanket regex). Only takes effect if disallow-blanket-regex is enabled.")
	var summarizeRangeQuery = flag.Bool("summarize-range-query", false, "Return summary statistics (max, min, avg, count) instead of full data points for range queries")
	flag.Parse()

	// Configure slog with specified log level
	configureLogging(*logLevel)

	// Parse and validate auth mode
	parsedAuthMode, err := mcp.ParseAuthMode(*authMode)
	if err != nil {
		log.Fatalf("Invalid auth mode: %v", err)
	}

	// Parse and validate metrics backend
	parsedMetricsBackend, err := parseMetricsBackend(*metricsBackend)
	if err != nil {
		log.Fatalf("Invalid metrics backend: %v", err)
	}

	// --metrics-backend only controls route discovery in kubeconfig mode.
	// Fail fast if it's set in any other mode to avoid silent misconfiguration.
	if parsedAuthMode != mcp.AuthModeKubeConfig && isFlagExplicitlySet("metrics-backend") {
		log.Fatalf("--metrics-backend has no effect with --auth-mode %s; "+
			"set PROMETHEUS_URL to point at your Thanos/Prometheus instance instead", parsedAuthMode)
	}

	// Determine metrics backend URL - pass the backend type
	metricsBackendURL, metricsURLSource, err := determineMetricsBackendURL(parsedAuthMode, parsedMetricsBackend)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Determine Alertmanager URL
	alertmanagerURL, alertmanagerURLSource, err := determineAlertmanagerURL(parsedAuthMode)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Parse guardrails configuration
	parsedGuardrails, err := prometheus.ParseGuardrails(*guardrails)
	if err != nil {
		log.Fatalf("Invalid guardrails configuration: %v", err)
	}

	// Set max metric cardinality and max label cardinality if guardrails are enabled
	if parsedGuardrails != nil {
		parsedGuardrails.MaxMetricCardinality = *maxMetricCardinality
		parsedGuardrails.MaxLabelCardinality = *maxLabelCardinality
	}

	// Create MCP options
	opts := mcp.ObsMCPOptions{
		AuthMode:            parsedAuthMode,
		MetricsBackendURL:   metricsBackendURL,
		AlertmanagerURL:     alertmanagerURL,
		Insecure:            *insecure,
		Guardrails:          parsedGuardrails,
		SummarizeRangeQuery: *summarizeRangeQuery,
	}

	// Create MCP server
	mcpServer, err := mcp.NewMCPServer(opts)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	slog.Info("Starting server",
		"auth_mode", opts.AuthMode,
		"metrics_backend_url", opts.MetricsBackendURL,
		"metrics_backend_url_source", metricsURLSource,
		"alertmanager_url", opts.AlertmanagerURL,
		"alertmanager_url_source", alertmanagerURLSource,
		"guardrails", opts.Guardrails,
	)

	// Choose server mode based on flags
	if *listen != "" {
		// HTTP mode
		ctx := context.Background()
		if err := mcp.Serve(ctx, mcpServer, *listen); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	} else {
		// Start server on stdio (default mode)
		stdioServer := server.NewStdioServer(mcpServer)
		if err := stdioServer.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}

func parseMetricsBackend(backend string) (k8s.MetricsBackend, error) {
	switch strings.ToLower(backend) {
	case "thanos", "":
		return k8s.MetricsBackendThanos, nil
	case "prometheus":
		return k8s.MetricsBackendPrometheus, nil
	default:
		return "", fmt.Errorf("unknown metrics backend %q, must be 'thanos' or 'prometheus'", backend)
	}
}

// determineMetricsBackendURL determines the metrics backend URL based on auth mode and environment.
// Returns the resolved URL, a source description for logging, and an error if the configuration is invalid.
func determineMetricsBackendURL(authMode mcp.AuthMode, backend k8s.MetricsBackend) (url, source string, err error) {
	if prometheusURL := os.Getenv("PROMETHEUS_URL"); prometheusURL != "" {
		return prometheusURL, "PROMETHEUS_URL env var", nil
	}

	if authMode == mcp.AuthModeKubeConfig {
		slog.Info("No PROMETHEUS_URL set, attempting route discovery", "backend", backend)
		url, err := k8s.GetMetricsBackendURL(backend)
		if err != nil {
			slog.Warn("Route discovery failed, falling back to default", "err", err, "default", defaultPrometheusURL)
			return defaultPrometheusURL, "default (route discovery failed)", nil
		}
		return url, "route discovery", nil
	}

	// serviceaccount and header modes are designed for deployments where the URL
	// is always known ahead of time. Falling back to localhost is never correct.
	return "", "", fmt.Errorf(
		"PROMETHEUS_URL must be set when using --auth-mode %s\n"+
			"  Set it via environment variable or use --auth-mode kubeconfig for auto-discovery",
		authMode,
	)
}

// determineAlertmanagerURL determines the Alertmanager URL based on auth mode and environment.
// Returns the resolved URL, a source description for logging, and an error if the configuration is invalid.
func determineAlertmanagerURL(authMode mcp.AuthMode) (url, source string, err error) {
	if alertmanagerURL := os.Getenv("ALERTMANAGER_URL"); alertmanagerURL != "" {
		return alertmanagerURL, "ALERTMANAGER_URL env var", nil
	}

	if authMode == mcp.AuthModeKubeConfig {
		slog.Info("No ALERTMANAGER_URL set, attempting route discovery")
		url, err := k8s.GetAlertmanagerURL()
		if err != nil {
			slog.Warn("Route discovery failed, falling back to default", "err", err, "default", defaultAlertmanagerURL)
			return defaultAlertmanagerURL, "default (route discovery failed)", nil
		}
		return url, "route discovery", nil
	}

	return "", "", fmt.Errorf(
		"ALERTMANAGER_URL must be set when using --auth-mode %s\n"+
			"  Set it via environment variable or use --auth-mode kubeconfig for auto-discovery",
		authMode,
	)
}

// isFlagExplicitlySet reports whether the named flag was explicitly provided on
// the command line (as opposed to relying on its default value).
func isFlagExplicitlySet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// configureLogging sets up the slog logger with the specified log level
func configureLogging(levelStr string) {
	level := promslog.NewLevel()
	err := level.Set(levelStr)
	if err != nil {
		log.Fatal(err.Error())
	}

	format := promslog.NewFormat()
	err = format.Set("logfmt")
	if err != nil {
		log.Fatal(err.Error())
	}

	logger := promslog.New(&promslog.Config{
		Level:  level,
		Format: format,
		Style:  promslog.GoKitStyle,
	})
	slog.SetDefault(logger)
}
