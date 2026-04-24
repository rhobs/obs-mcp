package otelcoltoolset

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"

	"github.com/rhobs/obs-mcp/pkg/otelcoltoolset/config"
	toolset_tools "github.com/rhobs/obs-mcp/pkg/otelcoltoolset/tools"
)

// Toolset implements the OpenTelemetry Collector toolset.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the name of the toolset.
func (t *Toolset) GetName() string {
	return config.OtelColToolSetName
}

// GetDescription returns a human-readable description of the toolset.
func (t *Toolset) GetDescription() string {
	return "Toolset for OpenTelemetry Collector configuration assistance including schema validation, component documentation, and version management."
}

// GetTools returns all tools provided by this toolset.
func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return slices.Concat(
		toolset_tools.InitListComponents(),
		toolset_tools.InitGetComponentSchema(),
		toolset_tools.InitValidateConfig(),
		toolset_tools.InitGetVersions(),
	)
}

// GetPrompts returns prompts provided by this toolset.
func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
