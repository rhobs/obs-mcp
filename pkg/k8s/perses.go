package k8s

import (
	"context"
	"encoding/json"
	"fmt"

	persesv1alpha1 "github.com/perses/perses-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// MCPHelpAnnotation is the annotation key for MCP help description
	MCPHelpAnnotation = "operator.perses.dev/mcp-help"
)

// GetPersesClient returns a controller-runtime client with types registered
func GetPersesClient() (client.Client, error) {
	config, err := GetClientConfig()
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	if err := persesv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add perses scheme: %w", err)
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create controller-runtime client: %w", err)
	}

	return c, nil
}

// ListDashboards lists all Dashboard objects across all namespaces or in a specific namespace.
// Uses types from github.com/perses/perses-operator/api/v1alpha1
// The labelSelector parameter accepts Kubernetes label selector syntax (e.g., "app=myapp,env=prod").
func ListDashboards(ctx context.Context, namespace, labelSelector string) ([]persesv1alpha1.PersesDashboard, error) {
	c, err := GetPersesClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get perses client: %w", err)
	}

	// Build list options
	listOpts := &client.ListOptions{}
	if namespace != "" {
		listOpts.Namespace = namespace
	}
	if labelSelector != "" {
		selector, err := labels.Parse(labelSelector)
		if err != nil {
			return nil, fmt.Errorf("invalid label selector: %w", err)
		}
		listOpts.LabelSelector = selector
	}

	var dashboardList persesv1alpha1.PersesDashboardList
	if err := c.List(ctx, &dashboardList, listOpts); err != nil {
		return nil, fmt.Errorf("failed to list Dashboards: %w", err)
	}

	return dashboardList.Items, nil
}

// GetDashboard retrieves a specific Dashboard by name and namespace.
// Returns the dashboard name, namespace, and full spec as a map for JSON serialization.
func GetDashboard(ctx context.Context, namespace, name string) (string, string, map[string]interface{}, error) {
	c, err := GetPersesClient()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get perses client: %w", err)
	}

	var dashboard persesv1alpha1.PersesDashboard
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := c.Get(ctx, key, &dashboard); err != nil {
		return "", "", nil, fmt.Errorf("failed to get Dashboard %s/%s: %w", namespace, name, err)
	}

	// Convert spec to map[string]interface{} for JSON serialization
	specMap, err := specToMap(dashboard.Spec)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to convert spec to map: %w", err)
	}

	return dashboard.Name, dashboard.Namespace, specMap, nil
}

// specToMap converts a DashboardSpec to a map[string]interface{} for JSON serialization
func specToMap(spec persesv1alpha1.Dashboard) (map[string]interface{}, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}
