package k8s

import (
	"context"
	"fmt"
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

// GetThanosQuerierURL discovers the Thanos Querier service URL in OpenShift
func GetThanosQuerierURL() (string, error) {
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
		Name(thanosQuerierRouteName).
		Do(ctx)

	if result.Error() != nil {
		return "", fmt.Errorf("failed to load thanos-querier route: %w", result.Error())
	}

	body, err := result.Raw()
	if err != nil {
		return "", fmt.Errorf("failed to parse the route results: %w", err)
	}

	// Simple string parsing to extract the host
	bodyStr := string(body)
	if strings.Contains(bodyStr, `"host":`) {
		// Extract host field using string manipulation
		parts := strings.Split(bodyStr, `"host":"`)
		if len(parts) > 1 {
			hostPart := strings.Split(parts[1], `"`)[0]
			if hostPart != "" {
				return "https://" + hostPart, nil
			}
		}
	}

	return "", fmt.Errorf("no suitable route found for thanos-querier")
}
