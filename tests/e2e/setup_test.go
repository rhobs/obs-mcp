//go:build e2e && !openshift

package e2e

import (
	"context"
	"fmt"
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

	mcpClient = NewMCPClient(testConfig.MCPURL)
	setupThanosDetection()

	code := m.Run()
	testConfig.Cleanup()
	os.Exit(code)
}
