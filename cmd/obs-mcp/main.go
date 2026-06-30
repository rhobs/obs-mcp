package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/klog/v2"

	"github.com/rhobs/obs-mcp/pkg/auth"
	"github.com/rhobs/obs-mcp/pkg/k8s"
	mcpserver "github.com/rhobs/obs-mcp/pkg/mcp"
	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	"github.com/rhobs/obs-mcp/pkg/traces"
)

var (
	version = "dev"
	commit  = "unknown"
)

const (
	defaultPrometheusURL   = "http://localhost:9090"
	defaultAlertmanagerURL = "http://localhost:9093"
	defaultLokiURL         = "http://localhost:3100"
)

func main() {
	var showVersion = flag.Bool("version", false, "Print version and exit")
	var listen = flag.String("listen", "", "Listen address for HTTP mode (e.g., :9100, 127.0.0.1:8080)")
	var toolsets = flag.String("toolsets", string(mcpserver.ToolsetMetrics), fmt.Sprintf("Comma-separated list of enabled toolsets: %s", strings.Join(mcpserver.AllToolsets, ", ")))
	var authMode = flag.String("auth-mode", "", "Authentication mode: kubeconfig, serviceaccount, or header")
	var insecure = flag.Bool("insecure", false, "Skip TLS certificate verification")
	var logLevel = flag.String("log-level", "info", "Log level: debug, info, warn, error")
	var metricsBackend = flag.String("metrics-backend", "thanos", "Metrics backend: thanos (default, with prometheus fallback) or prometheus (strict, no fallback)")
	var guardrails = flag.String("guardrails", "all",
		"Which safety checks are enforced on PromQL queries.\n"+
			"  'all': enable every guardrail\n"+
			"  'none': disable every guardrail\n"+
			"  Comma-separated list: enable only the named guardrails, e.g.\n"+
			"      disallow-explicit-name-label,require-label-matcher,disallow-blanket-regex,max-metric-cardinality\n"+
			"  Comma-separated list with ! prefix: disable the listed guardrails (enable the rest), e.g.\n"+
			"      !disallow-blanket-regex,!require-label-matcher\n"+
			"  '!tsdb' is a shortcut that disables both TSDB-dependent guardrails at once\n"+
			"      (max-metric-cardinality and disallow-blanket-regex)\n")
	var maxMetricCardinality = flag.Uint64("guardrails.max-metric-cardinality", prometheus.DefaultMaxMetricCardinality, "Maximum allowed series count per metric")
	var maxLabelCardinality = flag.Uint64("guardrails.max-label-cardinality", prometheus.DefaultMaxLabelCardinality,
		"Maximum allowed label value count for blanket regex (0 = always disallow blanket regex).\n"+
			"Only takes effect if disallow-blanket-regex is enabled.")
	var fullRangeQueryResponse = flag.Bool("full-range-query-response", false, "Return full data points for range queries")
	var tempoURL = flag.String("traces.tempo-url", "", "Tempo API base URL (overrides TEMPO_URL when explicitly set)")
	var tracesUseRoute = flag.Bool("traces.use-route", false, "Use Route instead of internal service DNS when connecting to Tempo API")
	var lokiURL = flag.String("loki-url", "", "Loki API base URL (overrides LOKI_URL when explicitly set)")
	var lokiUseRoute = flag.Bool("loki.use-route", false, "Use OpenShift Routes when discovering LokiStack endpoints")
	flag.Parse()

	if *showVersion {
		fmt.Printf("obs-mcp %s (commit: %s)\n", version, commit)
		os.Exit(0)
	}

	// Configure klog with specified log level
	configureLogging(*logLevel)
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Parse and validate auth mode
	parsedAuthMode, err := auth.ParseAuthMode(*authMode)
	if err != nil {
		log.Fatalf("Invalid auth mode: %v", err)
	}

	// Parse and validate metrics backend
	parsedMetricsBackend, err := parseMetricsBackend(*metricsBackend)
	if err != nil {
		log.Fatalf("Invalid metrics backend: %v", err)
	}
	parsedToolsets := parseToolsets(*toolsets)

	// --metrics-backend only controls route discovery in kubeconfig mode.
	// Fail fast if it's set in any other mode to avoid silent misconfiguration.
	if parsedAuthMode != auth.AuthModeKubeConfig && isFlagExplicitlySet("metrics-backend") {
		log.Fatalf("--metrics-backend has no effect with --auth-mode %s; "+
			"set PROMETHEUS_URL to point at your Thanos/Prometheus instance instead", parsedAuthMode)
	}

	metricsBackendURL := ""
	metricsURLSource := ""
	if slices.Contains(parsedToolsets, mcpserver.ToolsetMetrics) {
		metricsBackendURL, metricsURLSource, err = determineMetricsBackendURL(ctx, parsedAuthMode, parsedMetricsBackend)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	alertmanagerURL := ""
	alertmanagerURLSource := ""
	if slices.Contains(parsedToolsets, mcpserver.ToolsetMetrics) {
		alertmanagerURL, alertmanagerURLSource, err = determineAlertmanagerURL(ctx, parsedAuthMode)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	// Determine Loki URL only when logs toolset is enabled.
	lokiResolvedURL := ""
	lokiURLSource := ""
	if slices.Contains(parsedToolsets, mcpserver.ToolsetLogs) {
		lokiResolvedURL, lokiURLSource, err = determineLokiURL(ctx, parsedAuthMode, *lokiURL, *lokiUseRoute)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	// Parse guardrails configuration
	parsedGuardrails, err := prometheus.ParseGuardrails(*guardrails)
	if err != nil {
		log.Fatalf("Invalid guardrails configuration: %v", err)
	}

	// Reject deprecated use of 0 to disable the metric cardinality guardrail.
	if isFlagExplicitlySet("guardrails.max-metric-cardinality") && *maxMetricCardinality == 0 {
		log.Fatalf("--guardrails.max-metric-cardinality=0 is no longer supported to disable the guardrail; "+
			"use '!%s' in --guardrails instead", prometheus.GuardrailMaxMetricCardinality)
	}

	// Reject cardinality flags that have no effect given the active guardrails.
	if isFlagExplicitlySet("guardrails.max-metric-cardinality") &&
		(parsedGuardrails == nil || !parsedGuardrails.ForceMaxMetricCardinality) {
		log.Fatalf("--guardrails.max-metric-cardinality has no effect: add %q to --guardrails to enable the metric cardinality guardrail",
			prometheus.GuardrailMaxMetricCardinality)
	}
	if isFlagExplicitlySet("guardrails.max-label-cardinality") &&
		(parsedGuardrails == nil || !parsedGuardrails.DisallowBlanketRegex) {
		log.Fatalf("--guardrails.max-label-cardinality has no effect: add %q to --guardrails to enable the blanket-regex guardrail",
			prometheus.GuardrailDisallowBlanketRegex)
	}

	// Set max metric cardinality and max label cardinality if the corresponding guardrails are enabled.
	if parsedGuardrails != nil {
		if parsedGuardrails.ForceMaxMetricCardinality {
			parsedGuardrails.MaxMetricCardinality = *maxMetricCardinality
		}
		if parsedGuardrails.DisallowBlanketRegex {
			parsedGuardrails.MaxLabelCardinality = *maxLabelCardinality
		}
	}

	// Determine Tempo URL only when traces toolset is enabled.
	tempoResolvedURL := ""
	tempoURLSource := ""
	if slices.Contains(parsedToolsets, mcpserver.ToolsetTraces) {
		tempoResolvedURL, tempoURLSource = determineTempoURL(ctx, *tempoURL)
	}

	// Create MCP options
	opts := mcpserver.ObsMCPOptions{
		Toolsets:               parsedToolsets,
		AuthMode:               parsedAuthMode,
		MetricsBackendURL:      metricsBackendURL,
		AlertmanagerURL:        alertmanagerURL,
		LokiURL:                lokiResolvedURL,
		Insecure:               *insecure,
		Guardrails:             parsedGuardrails,
		FullRangeQueryResponse: *fullRangeQueryResponse,
		Traces: &traces.Config{
			AuthMode: parsedAuthMode,
			Insecure: *insecure,
			TempoURL: tempoResolvedURL,
			UseRoute: *tracesUseRoute,
		},
		Otelcol:      otelcol.NewDefaultConfig(),
		LokiUseRoute: *lokiUseRoute,
	}

	// Create MCP server
	mcpServer, err := mcpserver.NewMCPServer(opts)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	logger.Info("Starting server",
		"toolsets", opts.Toolsets,
		"auth_mode", opts.AuthMode,
		"metrics_backend_url", opts.MetricsBackendURL,
		"metrics_backend_url_source", metricsURLSource,
		"alertmanager_url", opts.AlertmanagerURL,
		"alertmanager_url_source", alertmanagerURLSource,
		"loki_url", opts.LokiURL,
		"loki_url_source", lokiURLSource,
		"tempo_url", tempoResolvedURL,
		"tempo_url_source", tempoURLSource,
		"guardrails", opts.Guardrails,
	)

	// Choose server mode based on flags
	if *listen != "" {
		// HTTP mode
		if err := mcpserver.Serve(ctx, mcpServer, *listen, opts.AuthMode); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	} else {
		// Start server on stdio (default mode)
		transport := &mcp.StdioTransport{}
		if _, err := mcpServer.Connect(ctx, transport, nil); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}

func parseToolsets(toolsets string) []mcpserver.Toolset {
	parts := strings.Split(toolsets, ",")
	result := make([]mcpserver.Toolset, len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if !slices.Contains(mcpserver.AllToolsets, p) {
			log.Fatalf("Unknown toolset %q, must be one of: %s", p, strings.Join(mcpserver.AllToolsets, ", "))
		}

		result[i] = mcpserver.Toolset(p)
	}
	return result
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
func determineMetricsBackendURL(ctx context.Context, authMode auth.AuthMode, backend k8s.MetricsBackend) (url, source string, err error) {
	logger := klog.FromContext(ctx)
	if prometheusURL := os.Getenv("PROMETHEUS_URL"); prometheusURL != "" {
		return prometheusURL, "PROMETHEUS_URL env var", nil
	}

	if authMode == auth.AuthModeKubeConfig {
		logger.Info("No PROMETHEUS_URL set, attempting route discovery", "backend", string(backend))
		url, err := k8s.GetMetricsBackendURL(ctx, backend)
		if err != nil {
			logger.Info("Route discovery failed, falling back to default", "err", err, "default", defaultPrometheusURL)
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
func determineAlertmanagerURL(ctx context.Context, authMode auth.AuthMode) (url, source string, err error) {
	logger := klog.FromContext(ctx)
	if alertmanagerURL := os.Getenv("ALERTMANAGER_URL"); alertmanagerURL != "" {
		return alertmanagerURL, "ALERTMANAGER_URL env var", nil
	}

	if authMode == auth.AuthModeKubeConfig {
		logger.Info("No ALERTMANAGER_URL set, attempting route discovery")
		url, err := k8s.GetAlertmanagerURL(ctx)
		if err != nil {
			logger.Info("Route discovery failed, falling back to default", "err", err, "default", defaultAlertmanagerURL)
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

func determineTempoURL(ctx context.Context, flagURL string) (url, source string) {
	logger := klog.FromContext(ctx)
	if flagURL != "" {
		return flagURL, "--traces.tempo-url flag"
	}
	if tempoURL := os.Getenv("TEMPO_URL"); tempoURL != "" {
		return tempoURL, "TEMPO_URL env var"
	}
	logger.Info("No Tempo URL configured; Tempo tools require tempoNamespace+tempoName discovery parameters or explicit Tempo URL")
	return "", "unset"
}

func determineLokiURL(ctx context.Context, authMode auth.AuthMode, flagURL string, useRoute bool) (url, source string, err error) {
	logger := klog.FromContext(ctx)
	if flagURL != "" {
		return flagURL, "--loki-url flag", nil
	}
	if lokiURL := os.Getenv("LOKI_URL"); lokiURL != "" {
		return lokiURL, "LOKI_URL env var", nil
	}
	if authMode == auth.AuthModeKubeConfig && !useRoute {
		logger.Info("No Loki URL configured, falling back to default", "default", defaultLokiURL)
		return defaultLokiURL, "default", nil
	}
	logger.Info("No Loki URL configured; Loki tools require lokiNamespace+lokiName discovery parameters or explicit Loki URL")
	return "", "unset", nil
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

// configureLogging sets up klog with the specified log level
func configureLogging(levelStr string) {
	fs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(fs)

	var verbosity string
	switch strings.ToLower(levelStr) {
	case "debug":
		verbosity = "4"
	default:
		verbosity = "0"
	}
	if err := fs.Set("v", verbosity); err != nil {
		log.Fatalf("Failed to set klog verbosity: %v", err)
	}
	if err := fs.Set("logtostderr", "true"); err != nil {
		log.Fatalf("Failed to set klog logtostderr: %v", err)
	}
}
