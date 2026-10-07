package toolset

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"

	"github.com/rhobs/obs-mcp/pkg/clusterinspector"
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
	}

	for _, ts := range toolsets {
		t.Run(ts.GetName(), func(t *testing.T) {
			tools := ts.GetTools(context.Background(), api.ToolsetContext{Inspector: clusterinspector.New()})
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
