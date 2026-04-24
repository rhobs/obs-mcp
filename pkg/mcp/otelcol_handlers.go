package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/resultutil"
)

var schemaLoader otelcol.SchemaLoader

func init() {
	schemaLoader = otelcol.NewSchemaLoader()
}

// ListOtelColComponentsHandler handles the listing of available OpenTelemetry Collector components.
func ListOtelColComponentsHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[otelcol.ListComponentsInput, otelcol.ListComponentsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input otelcol.ListComponentsInput) (*mcp.CallToolResult, otelcol.ListComponentsOutput, error) {
		result := otelcol.ListComponentsHandler(ctx, schemaLoader, input)
		output, err := resultutil.Unwrap[otelcol.ListComponentsOutput](result)
		if err != nil {
			return nil, otelcol.ListComponentsOutput{}, err
		}
		return nil, output, nil
	}
}

// GetOtelColComponentSchemaHandler handles getting a component's schema.
func GetOtelColComponentSchemaHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[otelcol.GetComponentSchemaInput, otelcol.GetComponentSchemaOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input otelcol.GetComponentSchemaInput) (*mcp.CallToolResult, otelcol.GetComponentSchemaOutput, error) {
		result := otelcol.GetComponentSchemaHandler(ctx, schemaLoader, input)
		output, err := resultutil.Unwrap[otelcol.GetComponentSchemaOutput](result)
		if err != nil {
			return nil, otelcol.GetComponentSchemaOutput{}, err
		}
		return nil, output, nil
	}
}

// ValidateOtelColConfigHandler handles validating a component configuration.
func ValidateOtelColConfigHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[otelcol.ValidateConfigInput, otelcol.ValidateConfigOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input otelcol.ValidateConfigInput) (*mcp.CallToolResult, otelcol.ValidateConfigOutput, error) {
		result := otelcol.ValidateConfigHandler(ctx, schemaLoader, input)
		output, err := resultutil.Unwrap[otelcol.ValidateConfigOutput](result)
		if err != nil {
			return nil, otelcol.ValidateConfigOutput{}, err
		}
		return nil, output, nil
	}
}

// GetOtelColVersionsHandler handles listing available versions.
func GetOtelColVersionsHandler(_ ObsMCPOptions) mcp.ToolHandlerFor[otelcol.GetVersionsInput, otelcol.GetVersionsOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input otelcol.GetVersionsInput) (*mcp.CallToolResult, otelcol.GetVersionsOutput, error) {
		result := otelcol.GetVersionsHandler(ctx, schemaLoader, input)
		output, err := resultutil.Unwrap[otelcol.GetVersionsOutput](result)
		if err != nil {
			return nil, otelcol.GetVersionsOutput{}, err
		}
		return nil, output, nil
	}
}
