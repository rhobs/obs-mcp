package tempo

import (
	"context"
	"fmt"
	"net/http"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	serverconfig "github.com/containers/kubernetes-mcp-server/pkg/config"
	"k8s.io/client-go/rest"
)

const TOOLSET_NAME = "obs-mcp-tempo"

func init() {
	serverconfig.RegisterToolsetConfig(TOOLSET_NAME, tempoToolsetParser)
}

type Config struct {
	UseRoute bool `toml:"useRoute,omitempty"`
}

var _ api.ExtendedConfig = (*Config)(nil)

var DefaultConfig = &Config{
	UseRoute: true,
}

func (c *Config) Validate() error {
	return nil
}

func tempoToolsetParser(_ context.Context, primitive toml.Primitive, md toml.MetaData) (api.ExtendedConfig, error) {
	var cfg Config
	if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func getConfig(params api.ToolHandlerParams) *Config {
	if cfg, ok := params.GetToolsetConfig(TOOLSET_NAME); ok {
		if tempoCfg, ok := cfg.(*Config); ok {
			return tempoCfg
		}
	}

	// Return default config if not found
	return DefaultConfig
}

func getHTTPClient(restConfig *rest.Config) (*http.Client, error) {
	// Create HTTP client with Kubernetes authentication
	rt, err := rest.TransportFor(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport from REST config: %w", err)
	}

	return &http.Client{
		Transport: rt,
	}, nil
}
