// Package clusterinspector provides an api.ClusterInspector stub that reports
// empty discovery results.
//
// It mirrors the upstream documentationInspector in kubernetes-mcp-server's
// internal/tools/update-readme/main.go, but returns an empty
// metav1.APIResourceList instead of the populated documentation resource list.
//
// obs-mcp wires this non-nil inspector into api.ToolsetContext so that the
// TargetCompatibilityFilters closures captured by the toolsets — which call
// api.AnyTargetHasGVK and therefore dereference Inspector.Discovery() — can be
// evaluated without a nil-pointer dereference. obs-mcp does not enable
// target-compatibility tool filtering. This inspector does not access a real
// cluster and deliberately reports no GVKs for every group/version.
package clusterinspector

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Inspector is an api.ClusterInspector that reports empty discovery results.
//
// It also implements api.AggregateDiscovery and api.TargetProvider: api.Results
// (returned by ServerResourcesForGroupVersion) calls TargetProvider.GetTargets
// during evaluation, so the inspector passes itself as the target provider,
// exactly as the upstream documentationInspector does.
type Inspector struct{}

// New returns a cluster inspector that reports no API resources.
func New() *Inspector { return &Inspector{} }

// Discovery returns the inspector itself as an api.AggregateDiscovery.
func (i *Inspector) Discovery() api.AggregateDiscovery { return i }

// Unstructured returns nil. obs-mcp compatibility filters only use Discovery
// (via api.AnyTargetHasGVK), so no unstructured access is needed.
func (i *Inspector) Unstructured() api.AggregateUnstructured { return nil }

// ServerResourcesForGroupVersion returns an empty APIResourceList for every
// group/version, so api.AnyTargetHasGVK reports false for all GVKs.
func (i *Inspector) ServerResourcesForGroupVersion(ctx context.Context, _ string) api.Results[*metav1.APIResourceList] {
	return api.NewResults(ctx, i, func(context.Context, string) (*metav1.APIResourceList, error) {
		return &metav1.APIResourceList{}, nil
	})
}

// IsMultiTarget reports whether the provider manages multiple targets.
func (i *Inspector) IsMultiTarget() bool { return false }

// GetTargets returns the single empty-named target, mirroring upstream.
func (i *Inspector) GetTargets(context.Context) ([]string, error) { return []string{""}, nil }

// GetDefaultTarget returns the default target name.
func (i *Inspector) GetDefaultTarget() string { return "" }

// GetTargetParameterName returns the tool parameter name used for target selection.
func (i *Inspector) GetTargetParameterName() string { return "" }

var (
	_ api.ClusterInspector   = (*Inspector)(nil)
	_ api.AggregateDiscovery = (*Inspector)(nil)
	_ api.TargetProvider     = (*Inspector)(nil)
)
