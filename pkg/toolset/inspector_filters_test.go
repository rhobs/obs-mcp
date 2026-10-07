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

// TestTargetCompatibilityFiltersDoNotPanic verifies that the compatibility
// filter closures captured by each toolset — which dereference
// ToolsetContext.Inspector via api.AnyTargetHasGVK(...Inspector.Discovery()...) —
// can be evaluated without a nil-pointer panic when a non-nil, empty-discovery
// inspector is supplied. With empty discovery every GVK is reported absent, so
// each closure returns false; the primary guarantee under test is no panic.
func TestTargetCompatibilityFiltersDoNotPanic(t *testing.T) {
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
				for i, filter := range tool.TargetCompatibilityFilters {
					if got := filter(); got {
						t.Errorf("tool %q filter %d: expected false for empty-discovery inspector, got true", tool.Tool.Name, i)
					}
				}
			}
		})
	}
}
