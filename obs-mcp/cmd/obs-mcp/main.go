package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/inecas/obs-mcp/pkg/k8s"
	"github.com/inecas/obs-mcp/pkg/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Parse command line flags
	var listen = flag.String("listen", "", "Listen address for HTTP mode (e.g., :9100, 127.0.0.1:8080)")
	var authMode = flag.String("auth-mode", "", "Authentication mode: kubeconfig, serviceaccount, or header")
	var insecure = flag.Bool("insecure", false, "Skip TLS certificate verification")

	var noGuardrails = flag.Bool("no-guardrails", false, "Disable guardrails")
	flag.Parse()

	// Parse and validate auth mode
	parsedAuthMode, err := mcp.ParseAuthMode(*authMode)
	if err != nil {
		log.Fatalf("Invalid auth mode: %v", err)
	}

	// Determine Prometheus URL
	promURL := determinePrometheusURL(parsedAuthMode)

	// Create MCP options
	opts := mcp.ObsMCPOptions{
		AuthMode:      parsedAuthMode,
		PromURL:       promURL,
		Insecure:      *insecure,
		UseGuardrails: !*noGuardrails,
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
			return "http://localhost:9090"
		}

		slog.Info("Discovered Thanos Querier URL", "url", url)
		return url
	}

	// Default to localhost for all other auth modes
	return "http://localhost:9090"
}
