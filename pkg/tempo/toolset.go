package tempo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
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

func withTempoInstanceParams() mcp.ToolOption {
	// Add parameters to identify a Tempo instance and tenant
	additionalParameters := []mcp.ToolOption{
		mcp.WithString("tempoNamespace",
			mcp.Required(),
			mcp.Description("The namespace of the Tempo instance to query"),
		),
		mcp.WithString("tempoName",
			mcp.Required(),
			mcp.Description("The name of the Tempo instance to query"),
		),
		mcp.WithString("tenant",
			mcp.Required(),
			mcp.Description("The tenant to query"),
		),
	}

	return func(t *mcp.Tool) {
		for _, opt := range additionalParameters {
			opt(t)
		}
	}
}

// getTempoClient returns a Tempo client based on the tempoNamespace, tempoName and tenant parameters.
func (t *TempoToolset) getTempoClient(ctx context.Context, request mcp.CallToolRequest) (*TempoClient, error) {
	namespace, err := request.RequireString("tempoNamespace")
	if err != nil {
		return nil, err
	}
	if namespace == "" {
		return nil, errors.New("tempoNamespace parameter must not be empty")
	}

	name, err := request.RequireString("tempoName")
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("tempoName parameter must not be empty")
	}

	tenant, err := request.RequireString("tenant")
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("tenant '%s' does not exist for instance '%s' in namespace '%s'", tenant, name, namespace)
	}

	url := instance.GetURL(tenant)
	httpClient, err := t.httpClientFactory(ctx)
	if err != nil {
		return nil, err
	}

	return NewTempoClient(httpClient, url), nil
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
