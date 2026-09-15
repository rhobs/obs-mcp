package toolset

import (
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"

	"github.com/rhobs/obs-mcp/pkg/alertmanagement"
	"github.com/rhobs/obs-mcp/pkg/logs"
	"github.com/rhobs/obs-mcp/pkg/metrics"
	"github.com/rhobs/obs-mcp/pkg/otelcol"
	"github.com/rhobs/obs-mcp/pkg/traces"
)

func TestAllToolsDeclareValidRBAC(t *testing.T) {
	toolsets := []api.Toolset{
		&metrics.Toolset{},
		&logs.Toolset{},
		&traces.Toolset{},
		&otelcol.Toolset{},
		&alertmanagement.Toolset{},
	}

	for _, ts := range toolsets {
		t.Run(ts.GetName(), func(t *testing.T) {
			tools := ts.GetTools(nil)
			if len(tools) == 0 {
				t.Fatal("expected tools")
			}
			for _, tool := range tools {
				t.Run(tool.Tool.Name, func(t *testing.T) {
					if tool.RBAC == nil {
						t.Fatal("RBAC metadata is required for feature parity with kubernetes-mcp-server")
					}
					if err := tool.RBAC.Validate(); err != nil {
						t.Fatalf("invalid RBAC metadata: %v", err)
					}
				})
			}
		})
	}
}
