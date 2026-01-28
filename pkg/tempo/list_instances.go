package tempo

import (
	"context"

	"github.com/rhobs/obs-mcp/pkg/tempo/discovery"
)

// ListInstances returns Tempo instances discovered in the cluster.
func (t *TempoToolset) ListInstances(ctx context.Context) ([]discovery.TempoInstance, error) {
	return t.discovery.ListInstances(ctx)
}
