package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/prometheus/common/promslog"

	"github.com/mark3labs/mcp-go/server"

	"github.com/rhobs/obs-mcp/pkg/k8s"
	"github.com/rhobs/obs-mcp/pkg/mcp"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

const (
	defaultPrometheusURL = "http://localhost:9090"
)

func main() {
	// Parse command line flags
	var listen = flag.String("listen", "", "Listen address for HTTP mode (e.g., :9100, 127.0.0.1:8080)")
	var authMode = flag.String("auth-mode", "", "Authentication mode: kubeconfig, serviceaccount, or header")
	var insecure = flag.Bool("insecure", false, "Skip TLS certificate verification")
	var logLevel = flag.String("log-level", "info", "Log level: debug, info, warn, error")

	var guardrails = flag.String("guardrails", "all", "Guardrails configuration: 'all' (default), 'none', or comma-separated list of guardrails to enable (disallow-explicit-name-label, require-label-matcher, disallow-blanket-regex)")
	var maxMetricCardinality = flag.Uint64("guardrails.max-metric-cardinality", 20000, "Maximum allowed series count per metric (0 = disabled)")
	var maxLabelCardinality = flag.Uint64("guardrails.max-label-cardinality", 500, "Maximum allowed label value count for blanket regex (0 = always disallow blanket regex). Only takes effect if disallow-blanket-regex is enabled.")
	flag.Parse()

	// Configure slog with specified log level
	configureLogging(*logLevel)

	// Parse and validate auth mode
	parsedAuthMode, err := mcp.ParseAuthMode(*authMode)
	if err != nil {
		log.Fatalf("Invalid auth mode: %v", err)
	}

	// Determine Prometheus URL
	promURL := determinePrometheusURL(parsedAuthMode)

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
		AuthMode:   parsedAuthMode,
		PromURL:    promURL,
		Insecure:   *insecure,
		Guardrails: parsedGuardrails,
	}

	// Create MCP server
	mcpServer, err := mcp.NewMCPServer(opts)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	slog.Info("Starting server", "PromURL", opts.PromURL, "AuthMode", opts.AuthMode)

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

// determinePrometheusURL determines the Prometheus URL based on auth mode and environment
func determinePrometheusURL(authMode mcp.AuthMode) string {
	// Get Prometheus URL from environment variable
	promURL := os.Getenv("PROMETHEUS_URL")

	// If URL is provided, use it
	if promURL != "" {
		return promURL
	}

	// For kubeconfig mode, attempt to discover Thanos Querier
	if authMode == mcp.AuthModeKubeConfig {
		slog.Info("No Prometheus URL provided, attempting to use kubeconfig to discover Thanos Querier")

		url, err := k8s.GetThanosQuerierURL()
		if err != nil {
			slog.Warn("Failed to discover Thanos Querier via kubeconfig, falling back to localhost", "err", err)
			return defaultPrometheusURL
		}

		slog.Info("Discovered Thanos Querier URL", "url", url)
		return url
	}

	// Default to localhost for all other auth modes
	return defaultPrometheusURL
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
