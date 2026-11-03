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
//
// Unstructured() is also non-nil: metrics.GetTools probes Perses via
// Inspector.Unstructured(). A nil AggregateUnstructured panics. The stub
// returns empty lists for most resources, and pretends a Perses instance
// exists so show_timeseries stays registered (standalone does not apply
// cluster-gated filtering; evals and TOOLS.md still expect that tool).
package clusterinspector

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var persesGVR = schema.GroupVersionResource{
	Group:    "perses.dev",
	Version:  "v1alpha2",
	Resource: "perses",
}

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

// Unstructured returns the inspector itself as an api.AggregateUnstructured.
func (i *Inspector) Unstructured() api.AggregateUnstructured { return i }

// Resource returns a namespaceable resource handle for the given GVR.
func (i *Inspector) Resource(resource schema.GroupVersionResource) api.AggregateNamespaceableResourceInterface {
	return &resourceHandle{inspector: i, resource: resource}
}

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

// BearerToken returns an empty token for every target. The stub does not talk
// to a cluster, so there is no derived client bearer token to report.
func (i *Inspector) BearerToken(ctx context.Context) api.Results[string] {
	return api.NewResults(ctx, i, func(context.Context, string) (string, error) {
		return "", nil
	})
}

type resourceHandle struct {
	inspector *Inspector
	resource  schema.GroupVersionResource
	namespace string
}

func (r *resourceHandle) Namespace(ns string) api.AggregateResourceInterface {
	return &resourceHandle{inspector: r.inspector, resource: r.resource, namespace: ns}
}

func (r *resourceHandle) Get(ctx context.Context, _ string, _ metav1.GetOptions, _ ...string) api.Results[*unstructured.Unstructured] {
	return api.NewResults(ctx, r.inspector, func(context.Context, string) (*unstructured.Unstructured, error) {
		return &unstructured.Unstructured{}, nil
	})
}

func (r *resourceHandle) List(ctx context.Context, _ metav1.ListOptions) api.Results[*unstructured.UnstructuredList] {
	return api.NewResults(ctx, r.inspector, func(context.Context, string) (*unstructured.UnstructuredList, error) {
		list := &unstructured.UnstructuredList{}
		// Keep show_timeseries registered: metrics.GetTools only adds it when
		// at least one Perses instance is listed.
		if r.resource == persesGVR {
			list.Items = []unstructured.Unstructured{{}}
		}
		return list, nil
	})
}

var (
	_ api.ClusterInspector                        = (*Inspector)(nil)
	_ api.AggregateDiscovery                      = (*Inspector)(nil)
	_ api.AggregateUnstructured                   = (*Inspector)(nil)
	_ api.TargetProvider                          = (*Inspector)(nil)
	_ api.AggregateNamespaceableResourceInterface = (*resourceHandle)(nil)
)
