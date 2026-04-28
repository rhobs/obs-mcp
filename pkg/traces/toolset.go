package traces

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"

	tempoclient "github.com/rhobs/obs-mcp/pkg/traces/tempo"
)

const ToolsetName = "traces"

// Toolset implements the observability toolset for Tempo.
type Toolset struct {
	NewTempoClient func(params api.ToolHandlerParams, url string) (tempoclient.Loader, error)
}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the name of the toolset.
func (t *Toolset) GetName() string {
	return ToolsetName
}

// GetDescription returns a human-readable description of the toolset.
func (t *Toolset) GetDescription() string {
	return "Toolset for querying Tempo"
}

// GetTools returns all tools provided by this toolset.
func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return []api.ServerTool{
		ListInstancesTool.ToServerTool(ToServerHandler(t.NewTempoClient, t.ListInstancesHandler)),
		GetTraceByIDTool.ToServerTool(ToServerHandler(t.NewTempoClient, t.GetTraceByIDHandler)),
		SearchTracesTool.ToServerTool(ToServerHandler(t.NewTempoClient, t.SearchTracesHandler)),
		SearchTagsTool.ToServerTool(ToServerHandler(t.NewTempoClient, t.SearchTagsHandler)),
		SearchTagValuesTool.ToServerTool(ToServerHandler(t.NewTempoClient, t.SearchTagValuesHandler)),
	}
}

// GetPrompts returns prompts provided by this toolset.
func (t *Toolset) GetPrompts() []api.ServerPrompt {
	// Currently, prompts are not supported through this toolset
	// The workflow instructions are embedded in the tool descriptions
	return nil
}
