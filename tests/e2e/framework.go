//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	defaultNamespace        = "obs-mcp"
	defaultServiceName      = "obs-mcp"
	defaultServicePort      = 9100
	defaultLocalPort        = 9100
	defaultPodLabelSelector = "app.kubernetes.io/name=obs-mcp"
	defaultTimeout          = 30 * time.Second
)

// TestConfig holds configuration and runtime state for e2e tests
type TestConfig struct {
	// Configuration
	Namespace        string
	ServiceName      string
	ServicePort      int
	LocalPort        int
	PodLabelSelector string
	Timeout          time.Duration

	// Runtime state
	MCPURL          string
	PortForwarder   *portforward.PortForwarder
	PortForwardStop chan struct{}
	cleanedUp       bool
}

// NewTestConfig creates a new TestConfig with defaults or env overrides
func NewTestConfig() *TestConfig {
	namespace := os.Getenv("OBS_MCP_NAMESPACE")
	if namespace == "" {
		namespace = defaultNamespace
	}
	config := &TestConfig{
		Namespace:        namespace,
		ServiceName:      defaultServiceName,
		ServicePort:      defaultServicePort,
		LocalPort:        defaultLocalPort,
		PodLabelSelector: defaultPodLabelSelector,
		Timeout:          defaultTimeout,
	}
	fmt.Printf("Test config: namespace=%s, service=%s, port=%d, timeout=%v\n",
		config.Namespace, config.ServiceName, config.ServicePort, config.Timeout)
	return config
}

// Setup initializes the test environment with a cancellable context
func (c *TestConfig) Setup(ctx context.Context) error {
	mcpURL, needsPortForward := c.getMCPURL()
	c.MCPURL = mcpURL

	if needsPortForward {
		pf, stopChan, err := c.startPortForward(ctx)
		if err != nil {
			return fmt.Errorf("failed to start port-forward: %w", err)
		}
		c.PortForwarder = pf
		c.PortForwardStop = stopChan
	}

	// Wait for service to be ready
	if err := c.waitForReady(ctx, c.MCPURL+"/health"); err != nil {
		c.Cleanup()
		return fmt.Errorf("failed waiting for obs-mcp: %w", err)
	}

	fmt.Printf("obs-mcp is ready at %s\n", c.MCPURL)
	return nil
}

// Cleanup stops port-forward if it was started. Safe to call multiple times.
func (c *TestConfig) Cleanup() {
	if c.cleanedUp {
		return
	}
	c.cleanedUp = true
	if c.PortForwardStop != nil {
		close(c.PortForwardStop)
	}
	if c.PortForwarder != nil {
		c.PortForwarder.Close()
	}
}

// getMCPURL determines the appropriate URL for accessing obs-mcp service
func (c *TestConfig) getMCPURL() (string, bool) {
	// 1. Explicit override via environment variable
	if envURL := os.Getenv("OBS_MCP_URL"); envURL != "" {
		fmt.Printf("Using OBS_MCP_URL from environment: %s\n", envURL)
		return envURL, false // No port-forward needed
	}

	// 2. Detect in-cluster environment (e.g., OpenShift Prow)
	k8sHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	fmt.Printf("KUBERNETES_SERVICE_HOST=%q\n", k8sHost)
	if k8sHost != "" {
		// Use FQDN to ensure cross-namespace DNS resolution works
		inClusterURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", c.ServiceName, c.Namespace, c.ServicePort)
		fmt.Printf("Detected in-cluster environment, using service DNS: %s\n", inClusterURL)
		return inClusterURL, false // No port-forward needed
	}

	// 3. External access - need port-forward
	fmt.Println("External environment detected, will use port-forward")
	return fmt.Sprintf("http://localhost:%d", c.LocalPort), true
}

// getKubeConfig returns the appropriate Kubernetes config
func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return kubeConfig.ClientConfig()
}

// startPortForward creates a port-forward to the obs-mcp service using client-go
func (c *TestConfig) startPortForward(ctx context.Context) (*portforward.PortForwarder, chan struct{}, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get kube config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Get pod name from service
	pods, err := clientset.CoreV1().Pods(c.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: c.PodLabelSelector,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return nil, nil, fmt.Errorf("no pods found for obs-mcp in namespace %s", c.Namespace)
	}

	// Use the first running pod
	targetPod := findRunningPod(pods.Items)
	if targetPod == "" {
		return nil, nil, fmt.Errorf("no running pods found for obs-mcp")
	}

	fmt.Printf("Found running pod: %s\n", targetPod)

	// Build the port-forward URL
	reqURL, err := url.Parse(fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward",
		config.Host, c.Namespace, targetPod))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Create SPDY transport
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, reqURL)

	// Create channels for port-forward lifecycle
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	// Create port forwarder
	ports := []string{fmt.Sprintf("%d:%d", c.LocalPort, c.ServicePort)}

	// Use a buffer to capture output
	var outBuf, errBuf bytes.Buffer
	pf, err := portforward.New(dialer, ports, stopChan, readyChan, &outBuf, &errBuf)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create port forwarder: %w", err)
	}

	// Start port-forward in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := pf.ForwardPorts(); err != nil {
			errChan <- err
		}
	}()

	// Wait for port-forward to be ready, error, or cancellation
	select {
	case <-ctx.Done():
		close(stopChan)
		return nil, nil, fmt.Errorf("cancelled: %w", ctx.Err())
	case <-readyChan:
		fmt.Printf("Port-forward established: localhost:%d -> %s:%d\n", c.LocalPort, targetPod, c.ServicePort)
		return pf, stopChan, nil
	case err := <-errChan:
		return nil, nil, fmt.Errorf("port-forward failed: %w", err)
	case <-time.After(c.Timeout):
		close(stopChan)
		return nil, nil, fmt.Errorf("timeout (%v) waiting for port-forward to be ready", c.Timeout)
	}
}

// findRunningPod returns the name of the first running pod from the list
func findRunningPod(pods []corev1.Pod) string {
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name
		}
	}
	return ""
}

// waitForReady polls the target URL until it returns HTTP 200, timeout occurs, or context is cancelled
func (c *TestConfig) waitForReady(ctx context.Context, targetURL string) error {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	fmt.Printf("Waiting for %s to be ready (timeout: %v)\n", targetURL, c.Timeout)
	attempt := 0
	var lastErr error
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return fmt.Errorf("cancelled waiting for %s", targetURL)
			}
			return fmt.Errorf("timeout waiting for %s to be ready (last error: %v)", targetURL, lastErr)
		case <-ticker.C:
			attempt++
			resp, err := http.Get(targetURL)
			if err != nil {
				lastErr = err
				fmt.Printf("Health check attempt %d failed: %v\n", attempt, err)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("Health check succeeded after %d attempts\n", attempt)
				return nil
			}
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			fmt.Printf("Health check attempt %d: status=%d\n", attempt, resp.StatusCode)
		}
	}
}
