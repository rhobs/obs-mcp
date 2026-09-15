package alertmanagement

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

const ToolsetName = "observability/alert-management"

// Toolset implements the alert management toolset backed by the
// monitoring-plugin management API.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return ToolsetName
}

func (t *Toolset) GetDescription() string {
	return "Toolset for listing OpenShift alerts and managing alert rules via the monitoring-plugin management API."
}

func (t *Toolset) GetTools(_ api.FilteringProvider) []api.ServerTool {
	return []api.ServerTool{
		initListAlerts(),
		initListAlertRules(),
		initPreviewAlertRule(),
		initCreateAlertRule(),
		initUpdateAlertRule(),
		initDeleteAlertRules(),
	}
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

func (t *Toolset) GetResources() []api.ServerResource {
	return nil
}

func (t *Toolset) GetResourceTemplates() []api.ServerResourceTemplate {
	return nil
}
