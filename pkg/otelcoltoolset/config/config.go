package config

import (
	"context"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	serverconfig "github.com/containers/kubernetes-mcp-server/pkg/config"
)

const OtelColToolSetName = "otelcol"

// Config holds OpenTelemetry Collector toolset configuration.
type Config struct {
	// SchemaDir is an optional path to a directory containing component schemas.
	// If not specified, embedded schemas from the collectorschema package are used.
	SchemaDir string `toml:"schema_dir,omitempty"`

	// DefaultVersion is the default OpenTelemetry Collector version to use when
	// version is not explicitly specified in tool calls. If not set, the latest
	// available version is used.
	DefaultVersion string `toml:"default_version,omitempty"`
}

var _ api.ExtendedConfig = (*Config)(nil)

// Validate checks that the configuration values are valid.
func (c *Config) Validate() error {
	return nil
}

func otelColToolsetParser(_ context.Context, primitive toml.Primitive, md toml.MetaData) (api.ExtendedConfig, error) {
	var cfg Config
	if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func init() {
	serverconfig.RegisterToolsetConfig(OtelColToolSetName, otelColToolsetParser)
}
