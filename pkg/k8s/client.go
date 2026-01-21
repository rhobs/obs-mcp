package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	openShiftRouteAPI      = "/apis/route.openshift.io/v1"
	monitoringNamespace    = "openshift-monitoring"
	routesResource         = "routes"
	thanosQuerierRouteName = "thanos-querier"
	prometheusRouteName    = "prometheus-k8s"
)

// MetricsBackend represents the type of metrics backend
type MetricsBackend string

const (
	MetricsBackendPrometheus MetricsBackend = "prometheus"
	MetricsBackendThanos     MetricsBackend = "thanos"
)

// GetClientConfig returns a Kubernetes REST config using kubeconfig
func GetClientConfig() (*rest.Config, error) {
	// Try to load from kubeconfig first
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return config, nil
}

// GetKubeClient returns a Kubernetes client
func GetKubeClient() (*kubernetes.Clientset, error) {
	config, err := GetClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// GetMetricsBackendURL discovers the metrics backend endpoint in OpenShift.
func GetMetricsBackendURL(backend MetricsBackend) (string, error) {
	if backend == MetricsBackendPrometheus {
		return discoverRoute(prometheusRouteName)
	}

	// Thanos with fallback to Prometheus
	url, err := discoverRoute(thanosQuerierRouteName)
	if err == nil {
		return url, nil
	}
	slog.Info("Thanos route not found, falling back to prometheus", "error", err)
	return discoverRoute(prometheusRouteName)
}

// discoverRoute attempts to find a route and logs the result.
func discoverRoute(routeName string) (string, error) {
	url, err := getRouteURL(routeName)
	if err != nil {
		slog.Error("Failed to discover route", "route", routeName, "error", err)
		return "", err
	}
	slog.Info("Successfully discovered route", "route", routeName, "url", url)
	return url, nil
}

func getRouteURL(routeName string) (string, error) {
	ctx := context.Background()

	kubeClient, err := GetKubeClient()
	if err != nil {
		return "", fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	restClient := kubeClient.CoreV1().RESTClient()
	result := restClient.Get().
		AbsPath(openShiftRouteAPI).
		Namespace(monitoringNamespace).
		Resource(routesResource).
		Name(routeName).
		Do(ctx)

	if result.Error() != nil {
		return "", fmt.Errorf("failed to load route %s: %w", routeName, result.Error())
	}

	body, err := result.Raw()
	if err != nil {
		return "", fmt.Errorf("failed to parse the route results: %w", err)
	}

	host := parseHostFromRouteBody(string(body))
	if host == "" {
		return "", fmt.Errorf("no host found in route %s", routeName)
	}
	return host, nil
}

func parseHostFromRouteBody(body string) string {
	if strings.Contains(body, `"host":`) {
		parts := strings.Split(body, `"host":"`)
		if len(parts) > 1 {
			hostPart := strings.Split(parts[1], `"`)[0]
			if hostPart != "" {
				return "https://" + hostPart
			}
		}
	}
	return ""
}
