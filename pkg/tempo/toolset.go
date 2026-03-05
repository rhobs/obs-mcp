package tempo

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

// Toolset implements the observability toolset for Tempo.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the name of the toolset.
func (t *Toolset) GetName() string {
	return "tempo"
}

// GetDescription returns a human-readable description of the toolset.
func (t *Toolset) GetDescription() string {
	return "Toolset for querying Tempo"
}

// GetTools returns all tools provided by this toolset.
func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return []api.ServerTool{
		ListInstancesTool.ToServerTool(ToServerHandler(t.ListInstancesHandler)),
		GetTraceByIdTool.ToServerTool(ToServerHandler(t.GetTraceByIdHandler)),
		SearchTracesTool.ToServerTool(ToServerHandler(t.SearchTracesHandler)),
		SearchTagsTool.ToServerTool(ToServerHandler(t.SearchTagsHandler)),
		SearchTagValuesTool.ToServerTool(ToServerHandler(t.SearchTagValuesHandler)),
	}
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
