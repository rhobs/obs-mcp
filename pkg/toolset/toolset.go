package toolset

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/rhobs/obs-mcp/pkg/prompts"
	"github.com/rhobs/obs-mcp/pkg/toolset/tools"
)

// Toolset implements the observability toolset for advanced Prometheus monitoring.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the name of the toolset.
func (t *Toolset) GetName() string {
	return "obs-mcp"
}

// GetDescription returns a human-readable description of the toolset.
func (t *Toolset) GetDescription() string {
	return prompts.ServerPrompt
}

// GetTools returns all tools provided by this toolset.
func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return slices.Concat(
		tools.InitListMetrics(),
		tools.InitExecuteInstantQuery(),
		tools.InitExecuteRangeQuery(),
		tools.InitGetLabelNames(),
		tools.InitGetLabelValues(),
		tools.InitGetSeries(),
		tools.InitGetAlerts(),
		tools.InitGetSilences(),
	)
}

// GetPrompts returns prompts provided by this toolset.
func (t *Toolset) GetPrompts() []api.ServerPrompt {
	// Currently, prompts are not supported through this toolset
	// The workflow instructions are embedded in the tool descriptions
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
