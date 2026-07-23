//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

var (
	testConfig *TestConfig
	mcpClient  *MCPClient
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, cleaning up...")
		cancel()
		if testConfig != nil {
			testConfig.Cleanup()
		}
		os.Exit(130) // Standard exit code for SIGINT
	}()

	testConfig = NewTestConfig()
	if err := testConfig.Setup(ctx); err != nil {
		fmt.Printf("Failed to setup test environment: %v\n", err)
		os.Exit(1)
	}

	// Token creation fails in OpenShift Prow e2e tests due to limited permissions of the serviceaccount running the tests,
	// however there we use kubeconfig auth mode on OpenShift and therefore do not need to send an auth header.
	token, err := createServiceAccountToken(testConfig.Namespace, testConfig.ServiceAccountName)
	if err != nil {
		slog.Warn("Failed to create a service account token, continuing without authenticating MCP tool calls", "err", err)
	}

	mcpClient = NewMCPClient(testConfig.MCPURL, token)
	setupThanosDetection()

	code := m.Run()
	testConfig.Cleanup()
	os.Exit(code)
}
