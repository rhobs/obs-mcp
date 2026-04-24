package tools

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"

	"github.com/rhobs/obs-mcp/pkg/otelcol"
)

// InitListComponents creates the list_otelcol_components tool.
func InitListComponents() []api.ServerTool {
	return []api.ServerTool{
		otelcol.ListComponents.ToServerTool(ListComponentsHandler),
	}
}

// InitGetComponentSchema creates the get_otelcol_component_schema tool.
func InitGetComponentSchema() []api.ServerTool {
	return []api.ServerTool{
		otelcol.GetComponentSchema.ToServerTool(GetComponentSchemaHandler),
	}
}

// InitValidateConfig creates the validate_otelcol_config tool.
func InitValidateConfig() []api.ServerTool {
	return []api.ServerTool{
		otelcol.ValidateConfig.ToServerTool(ValidateConfigHandler),
	}
}

// InitGetVersions creates the get_otelcol_versions tool.
func InitGetVersions() []api.ServerTool {
	return []api.ServerTool{
		otelcol.GetVersions.ToServerTool(GetVersionsHandler),
	}
}
