package otelcol

import (
	"context"
	"encoding/json"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	serverconfig "github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"

	"github.com/rhobs/obs-mcp/pkg/resultutil"
)

const ToolsetName = "otelcol"

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

// Toolset implements the OpenTelemetry Collector toolset.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the name of the toolset.
func (t *Toolset) GetName() string {
	return ToolsetName
}

// GetDescription returns a human-readable description of the toolset.
func (t *Toolset) GetDescription() string {
	return "Toolset for OpenTelemetry Collector configuration assistance including schema validation, component documentation, and version management."
}

// GetTools returns all tools provided by this toolset.
func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return []api.ServerTool{
		ListComponents.ToServerTool(ToServerHandler(t.ListComponentsHandler)),
		GetComponentSchema.ToServerTool(ToServerHandler(t.GetComponentSchemaHandler)),
		ValidateConfig.ToServerTool(ToServerHandler(t.ValidateConfigHandler)),
		GetVersions.ToServerTool(ToServerHandler(t.GetVersionsHandler)),
	}
}

// GetPrompts returns prompts provided by this toolset.
func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

// ToolParams contains parameters passed to tool handlers.
type ToolParams struct {
	context   context.Context
	arguments map[string]any
	config    *Config
}

// ToServerHandler converts a typed handler function to an api.ToolHandlerFunc.
func ToServerHandler[T any](handler func(params ToolParams) (T, error)) api.ToolHandlerFunc {
	return func(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
		config := getConfig(params)
		output, err := handler(ToolParams{
			context:   params.Context,
			arguments: params.GetArguments(),
			config:    config,
		})
		if err != nil {
			return api.NewToolCallResult("", err), nil
		}

		jsonBytes, err := json.Marshal(output)
		if err != nil {
			return nil, err
		}
		return api.NewToolCallResult(string(jsonBytes), nil), nil
	}
}

// getConfig retrieves the otelcol toolset configuration from params.
func getConfig(params api.ToolHandlerParams) *Config {
	if cfg, ok := params.GetToolsetConfig(ToolsetName); ok {
		if otelcolCfg, ok := cfg.(*Config); ok {
			return otelcolCfg
		}
	}
	return &Config{}
}

// getSchemaLoader creates a SchemaLoader using the toolset configuration.
func getSchemaLoader(config *Config) (SchemaLoader, error) {
	if config.SchemaDir != "" {
		loader, err := NewSchemaLoaderFromDir(config.SchemaDir)
		if err != nil {
			return NewSchemaLoader(), nil
		}
		return loader, nil
	}
	return NewSchemaLoader(), nil
}

// ListComponentsHandlerMethod handles the listing of available components.
func (t *Toolset) ListComponentsHandler(params ToolParams) (ListComponentsOutput, error) {
	loader, err := getSchemaLoader(params.config)
	if err != nil {
		return ListComponentsOutput{}, err
	}

	result := ListComponentsHandler(params.context, loader, BuildListComponentsInput(params.arguments))
	return resultutil.Unwrap[ListComponentsOutput](result)
}

// GetComponentSchemaHandlerMethod handles getting a component's schema.
func (t *Toolset) GetComponentSchemaHandler(params ToolParams) (GetComponentSchemaOutput, error) {
	loader, err := getSchemaLoader(params.config)
	if err != nil {
		return GetComponentSchemaOutput{}, err
	}

	result := GetComponentSchemaHandler(params.context, loader, BuildGetComponentSchemaInput(params.arguments))
	return resultutil.Unwrap[GetComponentSchemaOutput](result)
}

// ValidateConfigHandlerMethod handles validating a component configuration.
func (t *Toolset) ValidateConfigHandler(params ToolParams) (ValidateConfigOutput, error) {
	loader, err := getSchemaLoader(params.config)
	if err != nil {
		return ValidateConfigOutput{}, err
	}

	result := ValidateConfigHandler(params.context, loader, BuildValidateConfigInput(params.arguments))
	return resultutil.Unwrap[ValidateConfigOutput](result)
}

// GetVersionsHandlerMethod handles listing available versions.
func (t *Toolset) GetVersionsHandler(params ToolParams) (GetVersionsOutput, error) {
	loader, err := getSchemaLoader(params.config)
	if err != nil {
		return GetVersionsOutput{}, err
	}

	result := GetVersionsHandler(params.context, loader, BuildGetVersionsInput(params.arguments))
	return resultutil.Unwrap[GetVersionsOutput](result)
}

func otelColToolsetParser(_ context.Context, primitive toml.Primitive, md toml.MetaData) (api.ExtendedConfig, error) {
	var cfg Config
	if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func init() {
	toolsets.Register(&Toolset{})
	serverconfig.RegisterToolsetConfig(ToolsetName, otelColToolsetParser)
}
