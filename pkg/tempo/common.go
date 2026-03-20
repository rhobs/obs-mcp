package tempo

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/rhobs/obs-mcp/pkg/prometheus"
	"github.com/rhobs/obs-mcp/pkg/resultutil"
	tempoclient "github.com/rhobs/obs-mcp/pkg/tempo/client"
	"github.com/rhobs/obs-mcp/pkg/tempo/discovery"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

var (
	tempoNamespaceParameter = tools.ParamDef{
		Name:        "tempoNamespace",
		Type:        tools.ParamTypeString,
		Description: "The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces.",
		Required:    true,
	}
	tempoNameParameter = tools.ParamDef{
		Name:        "tempoName",
		Type:        tools.ParamTypeString,
		Description: "The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names.",
		Required:    true,
	}
	tempoTenantParameter = tools.ParamDef{
		Name:        "tenant",
		Type:        tools.ParamTypeString,
		Description: "The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance.",
		Required:    false,
	}
)

// getTempoClient returns a Tempo client based on the tempoNamespace, tempoName and tenant parameters.
func (t *Toolset) getTempoClient(params ToolParams) (tempoclient.Loader, error) {
	args := params.arguments

	namespace := tools.GetString(args, "tempoNamespace", "")
	if namespace == "" {
		return nil, errors.New("tempoNamespace parameter must not be empty")
	}

	name := tools.GetString(args, "tempoName", "")
	if name == "" {
		return nil, errors.New("tempoName parameter must not be empty")
	}

	instances, err := discovery.ListInstances(params.context, params.dynamicClient, params.config.UseRoute)
	if err != nil {
		return nil, err
	}

	// Make sure this Tempo instance exists in cluster. Otherwise, an attacker could potentially trick the MCP tool to connect to non-Tempo services.
	instance, err := findInstanceByName(instances, namespace, name)
	if err != nil {
		return nil, err
	}

	tenant := tools.GetString(args, "tenant", "")
	if instance.Multitenancy {
		if tenant == "" {
			return nil, errors.New("tenant parameter must not be empty for multi-tenant instance")
		}
		if !slices.Contains(instance.Tenants, tenant) {
			return nil, fmt.Errorf("tenant '%s' does not exist for instance '%s' in namespace '%s'", tenant, name, namespace)
		}
	}

	url := instance.GetURL(tenant)
	httpClient, err := getHTTPClient(params.restConfig)
	if err != nil {
		return nil, err
	}

	return tempoclient.NewTempoLoader(httpClient, url), nil
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

// ToolParams is a subset of api.ToolHandlerParams and contains only fields required by tempo tool handlers.
type ToolParams struct {
	context       context.Context
	arguments     map[string]any
	dynamicClient dynamic.Interface
	restConfig    *rest.Config
	config        *Config
}

func ToMCPHandler(restConfig *rest.Config, dynamicClient dynamic.Interface, config *Config, handler func(params ToolParams) *resultutil.Result) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result := handler(ToolParams{
			context:       ctx,
			arguments:     request.GetArguments(),
			dynamicClient: dynamicClient,
			restConfig:    restConfig,
			config:        config,
		})
		return result.ToMCPResult()
	}
}

func ToServerHandler(handler func(params ToolParams) *resultutil.Result) api.ToolHandlerFunc {
	return func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
		config := getConfig(params)
		result := handler(ToolParams{
			context:       params.Context,
			arguments:     params.GetArguments(),
			dynamicClient: params.DynamicClient(),
			restConfig:    params.RESTConfig(),
			config:        config,
		})
		return result.ToToolsetResult()
	}
}
