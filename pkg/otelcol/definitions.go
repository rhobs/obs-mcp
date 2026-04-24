package otelcol

import (
	"strings"

	"github.com/rhobs/obs-mcp/pkg/tools"
)

// componentTypeList returns a comma-separated list of valid component types for documentation.
func componentTypeList() string {
	types := ValidComponentTypes()
	strs := make([]string, len(types))
	for i, t := range types {
		strs[i] = string(t)
	}
	return strings.Join(strs, ", ")
}

// All tool definitions for OpenTelemetry Collector
var (
	ListComponents = tools.ToolDef[ListComponentsOutput]{
		Name:        "list_otelcol_components",
		Description: ListComponentsPrompt,
		Title:       "List OpenTelemetry Collector Components",
		Params: []tools.ParamDef{
			{
				Name:        "version",
				Type:        tools.ParamTypeString,
				Description: "OpenTelemetry Collector version (e.g., 'v0.100.0'). If not specified, uses the latest available version.",
				Required:    false,
			},
		},
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
	}

	GetComponentSchema = tools.ToolDef[GetComponentSchemaOutput]{
		Name:        "get_otelcol_component_schema",
		Description: GetComponentSchemaPrompt,
		Title:       "Get OpenTelemetry Collector Component Schema",
		Params: []tools.ParamDef{
			{
				Name:        "component_type",
				Type:        tools.ParamTypeString,
				Description: "Type of the component: " + componentTypeList(),
				Required:    true,
			},
			{
				Name:        "component_name",
				Type:        tools.ParamTypeString,
				Description: "Name of the component (e.g., 'otlp', 'prometheus', 'batch', 'debug'). Use list_otelcol_components to discover available components.",
				Required:    true,
			},
			{
				Name:        "version",
				Type:        tools.ParamTypeString,
				Description: "OpenTelemetry Collector version (e.g., 'v0.100.0'). If not specified, uses the latest available version.",
				Required:    false,
			},
		},
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
	}

	ValidateConfig = tools.ToolDef[ValidateConfigOutput]{
		Name:        "validate_otelcol_config",
		Description: ValidateConfigPrompt,
		Title:       "Validate OpenTelemetry Collector Component Configuration",
		Params: []tools.ParamDef{
			{
				Name:        "component_type",
				Type:        tools.ParamTypeString,
				Description: "Type of the component: " + componentTypeList(),
				Required:    true,
			},
			{
				Name:        "component_name",
				Type:        tools.ParamTypeString,
				Description: "Name of the component (e.g., 'otlp', 'prometheus', 'batch', 'debug')",
				Required:    true,
			},
			{
				Name:        "config",
				Type:        tools.ParamTypeString,
				Description: "The configuration to validate, as a YAML or JSON string",
				Required:    true,
			},
			{
				Name:        "format",
				Type:        tools.ParamTypeString,
				Description: "Format of the config: 'yaml' (default) or 'json'",
				Required:    false,
			},
			{
				Name:        "version",
				Type:        tools.ParamTypeString,
				Description: "OpenTelemetry Collector version (e.g., 'v0.100.0'). If not specified, uses the latest available version.",
				Required:    false,
			},
		},
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
	}

	GetVersions = tools.ToolDef[GetVersionsOutput]{
		Name:        "get_otelcol_versions",
		Description: GetVersionsPrompt,
		Title:       "Get Available OpenTelemetry Collector Versions",
		Params:      []tools.ParamDef{},
		ReadOnly:    true,
		Destructive: false,
		Idempotent:  true,
		OpenWorld:   true,
	}
)
