package tempo

import (
	"github.com/rhobs/obs-mcp/pkg/resultutil"
	"github.com/rhobs/obs-mcp/pkg/tempo/discovery"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

var ListInstancesTool = tools.ToolDef{
	Name: "tempo_list_instances",
	Description: `List all Tempo instances available in the Kubernetes cluster.
Call this tool first to discover available Tempo instances before using other Tempo tools,
as the returned namespace, name, and tenant values are required parameters for all other Tempo tools.`,
	Title:       "List Tempo instances",
	ReadOnly:    true,
	Destructive: false,
	Idempotent:  true,
	OpenWorld:   true,
}

func (t *Toolset) ListInstancesHandler(params ToolParams) *resultutil.Result {
	instances, err := discovery.ListInstances(params.context, params.dynamicClient, params.config.UseRoute)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	return resultutil.NewSuccessResult(map[string]any{
		"instances": instances,
	})
}
