package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/oklog/run"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/version"

	"github.com/rhobs/obs-mcp/pkg/auth"
	"github.com/rhobs/obs-mcp/pkg/health"
	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/logs"
	mcpserver "github.com/rhobs/obs-mcp/pkg/mcp"
	"github.com/rhobs/obs-mcp/pkg/metrics/prometheus"
	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/traces"
)

const (
	defaultPrometheusURL   = "http://localhost:9090"
	defaultAlertmanagerURL = "http://localhost:9093"
	defaultLokiURL         = "http://localhost:3100"
)

func main() { //nolint:gocyclo // main wires up flags, config, and run group
	var showVersion = flag.Bool("version", false, "Print version and exit")
	var listen = flag.String("listen", "", "Listen address for HTTP mode (e.g., :9100, 127.0.0.1:8080)")
	var listenInternal = flag.String("listen-internal", "", "Listen address for internal health server (metrics, pprof, health e.g., :8081, 127.0.0.1:8081). Off by default.")
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
		log.Println("obs-mcp", "build_info", version.Info(), "build_context", version.BuildContext())
		os.Exit(0)
	}

	reg := prom.NewRegistry()
	reg.MustRegister(
		versioncollector.NewCollector("obs-mcp"),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// Configure slog with specified log level
	configureLogging(*logLevel)

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
		metricsBackendURL, metricsURLSource, err = determineMetricsBackendURL(parsedAuthMode, parsedMetricsBackend)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	alertmanagerURL := ""
	alertmanagerURLSource := ""
	if slices.Contains(parsedToolsets, mcpserver.ToolsetMetrics) {
		alertmanagerURL, alertmanagerURLSource, err = determineAlertmanagerURL(parsedAuthMode)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	// Determine Loki URL only when logs toolset is enabled.
	lokiResolvedURL := ""
	lokiURLSource := ""
	if slices.Contains(parsedToolsets, logs.ToolsetName) {
		lokiResolvedURL, lokiURLSource, err = determineLokiURL(parsedAuthMode, *lokiURL, *lokiUseRoute)
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
	if slices.Contains(parsedToolsets, traces.ToolsetName) {
		tempoResolvedURL, tempoURLSource = determineTempoURL(*tempoURL)
	}

	// Create MCP options
	opts := mcpserver.ObsMCPOptions{
		Toolsets:               parsedToolsets,
		AuthMode:               parsedAuthMode,
		MetricsBackendURL:      metricsBackendURL,
		AlertmanagerURL:        alertmanagerURL,
		Insecure:               *insecure,
		Guardrails:             parsedGuardrails,
		FullRangeQueryResponse: *fullRangeQueryResponse,
		Traces: &traces.Config{
			AuthMode: parsedAuthMode,
			Insecure: *insecure,
			TempoURL: tempoResolvedURL,
			UseRoute: *tracesUseRoute,
		},
		Otelcol: otelcol.NewDefaultConfig(),
		Logs: &logs.Config{
			AuthMode: parsedAuthMode,
			Insecure: *insecure,
			LokiURL:  lokiResolvedURL,
			UseRoute: *lokiUseRoute,
		},
		Registry: reg,
	}

	if err := validateConfigs(opts); err != nil {
		log.Fatalf("%v", err)
	}

	// Create MCP server
	mcpServer, err := mcpserver.NewMCPServer(opts)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	slog.Info("starting obs-mcp", "build_info", version.Info(), "build_context", version.BuildContext())

	slog.Info("Starting server",
		"toolsets", opts.Toolsets,
		"auth_mode", opts.AuthMode,
		"metrics_backend_url", opts.MetricsBackendURL,
		"metrics_backend_url_source", metricsURLSource,
		"alertmanager_url", opts.AlertmanagerURL,
		"alertmanager_url_source", alertmanagerURLSource,
		"loki_url", opts.Logs.LokiURL,
		"loki_url_source", lokiURLSource,
		"tempo_url", tempoResolvedURL,
		"tempo_url_source", tempoURLSource,
		"guardrails", opts.Guardrails,
	)

	var g run.Group

	ctx, cancel := context.WithCancel(context.Background())
	g.Add(func() error {
		<-ctx.Done()
		return ctx.Err()
	}, func(error) {
		cancel()
	})

	// Add signal handler to run group
	{
		cancelCh := make(chan struct{})
		g.Add(func() error {
			return interrupt(cancelCh)
		}, func(error) {
			close(cancelCh)
		})
	}

	// Add internal health server to run group
	if listenInternal != nil && *listenInternal != "" {
		healthServer := health.NewServer(reg)
		httpServer, shutdown := healthServer.ListenAndServe(*listenInternal)
		g.Add(func() error {
			slog.Info("Internal health server starting", "listen_addr", *listenInternal)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("internal health server failed: %w", err)
			}
			return nil
		}, shutdown)
	}

	// Choose server mode based on flags
	if *listen != "" {
		// HTTP mode
		httpServer, shutdown := mcpserver.NewHTTPServer(mcpServer, *listen, reg, parsedAuthMode)
		g.Add(func() error {
			slog.Info("HTTP server starting", "listen_addr", *listen)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("HTTP server failed: %w", err)
			}
			return nil
		}, shutdown)
	} else {
		// stdio mode
		transport := &mcp.StdioTransport{}
		g.Add(func() error {
			slog.Info("Starting stdio MCP server")
			if _, err := mcpServer.Connect(ctx, transport, nil); err != nil {
				return fmt.Errorf("stdio server failed: %w", err)
			}
			return nil
		}, func(error) {
			slog.Info("Shutting down stdio MCP server")
			// Context cancellation handled by the context actor above
		})
	}

	if err := g.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	slog.Info("Exiting")
}

func interrupt(cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-c:
		slog.Info("Caught signal, exiting", "signal", s)
		return nil
	case <-cancel:
		return errors.New("canceled")
	}
}

func validateConfigs(opts mcpserver.ObsMCPOptions) error {
	if slices.Contains(opts.Toolsets, logs.ToolsetName) {
		if err := opts.Logs.Validate(); err != nil {
			return fmt.Errorf("invalid logs config: %w", err)
		}
	}
	if slices.Contains(opts.Toolsets, traces.ToolsetName) {
		if err := opts.Traces.Validate(); err != nil {
			return fmt.Errorf("invalid traces config: %w", err)
		}
	}
	if slices.Contains(opts.Toolsets, otelcol.ToolsetName) {
		if err := opts.Otelcol.Validate(); err != nil {
			return fmt.Errorf("invalid otelcol config: %w", err)
		}
	}
	return nil
}

func parseToolsets(toolsets string) []string {
	parts := strings.Split(toolsets, ",")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if !slices.Contains(mcpserver.AllToolsets, p) {
			log.Fatalf("Unknown toolset %q, must be one of: %s", p, strings.Join(mcpserver.AllToolsets, ", "))
		}
		parts[i] = p
	}
	return parts
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
func determineMetricsBackendURL(authMode auth.AuthMode, backend k8s.MetricsBackend) (url, source string, err error) {
	if prometheusURL := os.Getenv("PROMETHEUS_URL"); prometheusURL != "" {
		return prometheusURL, "PROMETHEUS_URL env var", nil
	}

	if authMode == auth.AuthModeKubeConfig {
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
func determineAlertmanagerURL(authMode auth.AuthMode) (url, source string, err error) {
	if alertmanagerURL := os.Getenv("ALERTMANAGER_URL"); alertmanagerURL != "" {
		return alertmanagerURL, "ALERTMANAGER_URL env var", nil
	}

	if authMode == auth.AuthModeKubeConfig {
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

func determineTempoURL(flagURL string) (url, source string) {
	if flagURL != "" {
		return flagURL, "--traces.tempo-url flag"
	}
	if tempoURL := os.Getenv("TEMPO_URL"); tempoURL != "" {
		return tempoURL, "TEMPO_URL env var"
	}
	slog.Info("No Tempo URL configured; Tempo tools require tempoNamespace+tempoName discovery parameters or explicit Tempo URL")
	return "", "unset"
}

func determineLokiURL(authMode auth.AuthMode, flagURL string, useRoute bool) (url, source string, err error) {
	if flagURL != "" {
		return flagURL, "--loki-url flag", nil
	}
	if lokiURL := os.Getenv("LOKI_URL"); lokiURL != "" {
		return lokiURL, "LOKI_URL env var", nil
	}
	if authMode == auth.AuthModeKubeConfig && !useRoute {
		slog.Warn("No Loki URL configured, falling back to default", "default", defaultLokiURL)
		return defaultLokiURL, "default", nil
	}
	slog.Warn("No Loki URL configured; Loki tools require lokiNamespace+lokiName discovery parameters or explicit Loki URL")
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
