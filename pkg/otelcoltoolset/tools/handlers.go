package tools

import (
	"fmt"
	"log/slog"

	"github.com/containers/kubernetes-mcp-server/pkg/api"

	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/otelcoltoolset/config"
)

// getConfig retrieves the otelcol toolset configuration from params.
func getConfig(params api.ToolHandlerParams) *config.Config {
	if cfg, ok := params.GetToolsetConfig(config.OtelColToolSetName); ok {
		if otelcolCfg, ok := cfg.(*config.Config); ok {
			return otelcolCfg
		}
	}
	return &config.Config{}
}

// getSchemaLoader creates a SchemaLoader using the toolset configuration.
func getSchemaLoader(params api.ToolHandlerParams) (otelcol.SchemaLoader, error) {
	cfg := getConfig(params)

	if cfg.SchemaDir != "" {
		loader, err := otelcol.NewSchemaLoaderFromDir(cfg.SchemaDir)
		if err != nil {
			slog.Warn("Failed to create schema loader from directory, falling back to embedded", "schemaDir", cfg.SchemaDir, "error", err)
			return otelcol.NewSchemaLoader(), nil
		}
		return loader, nil
	}

	return otelcol.NewSchemaLoader(), nil
}

// ListComponentsHandler handles the listing of available components.
func ListComponentsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	loader, err := getSchemaLoader(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create schema loader: %w", err)), nil
	}

	return otelcol.ListComponentsHandler(params.Context, loader, otelcol.BuildListComponentsInput(params.GetArguments())).ToToolsetResult()
}

// GetComponentSchemaHandler handles getting a component's schema.
func GetComponentSchemaHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	loader, err := getSchemaLoader(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create schema loader: %w", err)), nil
	}

	return otelcol.GetComponentSchemaHandler(params.Context, loader, otelcol.BuildGetComponentSchemaInput(params.GetArguments())).ToToolsetResult()
}

// ValidateConfigHandler handles validating a component configuration.
func ValidateConfigHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	loader, err := getSchemaLoader(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create schema loader: %w", err)), nil
	}

	return otelcol.ValidateConfigHandler(params.Context, loader, otelcol.BuildValidateConfigInput(params.GetArguments())).ToToolsetResult()
}

// GetVersionsHandler handles listing available versions.
func GetVersionsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	loader, err := getSchemaLoader(params)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create schema loader: %w", err)), nil
	}

	return otelcol.GetVersionsHandler(params.Context, loader, otelcol.BuildGetVersionsInput(params.GetArguments())).ToToolsetResult()
}
