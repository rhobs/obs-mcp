package tempo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"k8s.io/client-go/dynamic"

	"github.com/rhobs/obs-mcp/pkg/prometheus"
	"github.com/rhobs/obs-mcp/pkg/tempo/discovery"
)

// HTTPClientFactory creates an HTTP client for Tempo requests
type HTTPClientFactory func(ctx context.Context) (*http.Client, error)

type TempoToolset struct {
	discovery         *discovery.TempoDiscovery
	useRoute          bool
	httpClientFactory HTTPClientFactory
}

func NewTempoToolset(k8sClient *dynamic.DynamicClient, useRoute bool, httpClientFactory HTTPClientFactory) *TempoToolset {
	d := discovery.New(k8sClient, useRoute)

	return &TempoToolset{
		discovery:         d,
		useRoute:          useRoute,
		httpClientFactory: httpClientFactory,
	}
}

// GetTempoClient returns a Tempo client for the given instance and tenant after validating against cluster discovery.
func (t *TempoToolset) GetTempoClient(ctx context.Context, namespace, name, tenant string) (*TempoClient, error) {
	if namespace == "" {
		return nil, errors.New("tempoNamespace parameter must not be empty")
	}
	if name == "" {
		return nil, errors.New("tempoName parameter must not be empty")
	}
	if tenant == "" {
		return nil, errors.New("tenant parameter must not be empty")
	}

	instances, err := t.discovery.ListInstances(ctx)
	if err != nil {
		return nil, err
	}

	// Make sure this Tempo instance exists in cluster. Otherwise, an attacker could potentially trick the MCP tool to connect to non-Tempo services.
	instance, err := findInstanceByName(instances, namespace, name)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(instance.Tenants, tenant) {
		return nil, fmt.Errorf("tenant %q is not configured for Tempo instance %q in namespace %q", tenant, name, namespace)
	}

	u := instance.GetURL(tenant)
	httpClient, err := t.httpClientFactory(ctx)
	if err != nil {
		return nil, err
	}

	return NewTempoClient(httpClient, u), nil
}

func findInstanceByName(instances []discovery.TempoInstance, namespace, name string) (discovery.TempoInstance, error) {
	for _, instance := range instances {
		if instance.Namespace == namespace && instance.Name == name {
			return instance, nil
		}
	}

	return discovery.TempoInstance{}, fmt.Errorf("instance '%s' in namespace '%s' not found", name, namespace)
}

func parseDate(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}

	ts, err := prometheus.ParseTimestamp(s)
	if err != nil {
		return 0, err
	}
	return ts.Unix(), nil
}
