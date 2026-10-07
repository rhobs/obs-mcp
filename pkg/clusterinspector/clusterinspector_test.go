package clusterinspector

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestServerResourcesForGroupVersionEmpty verifies the inspector reports an
// empty APIResourceList for any group/version without error.
func TestServerResourcesForGroupVersionEmpty(t *testing.T) {
	insp := New()
	list, err := insp.Discovery().
		ServerResourcesForGroupVersion(context.Background(), "tempo.grafana.com/v1alpha1").
		Default()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil APIResourceList")
	}
	if len(list.APIResources) != 0 {
		t.Fatalf("expected empty APIResources, got %d", len(list.APIResources))
	}
}

// TestAnyTargetHasGVKReportsFalse verifies that api.AnyTargetHasGVK evaluates
// without panicking and reports every GVK as absent against empty discovery.
func TestAnyTargetHasGVKReportsFalse(t *testing.T) {
	insp := New()
	gvks := []schema.GroupVersionKind{
		{Group: "tempo.grafana.com", Version: "v1alpha1", Kind: "TempoStack"},
		{Group: "loki.grafana.com", Version: "v1", Kind: "LokiStack"},
		{Group: "opentelemetry.io", Version: "v1beta1", Kind: "OpenTelemetryCollector"},
	}
	for _, gvk := range gvks {
		if api.AnyTargetHasGVK(context.Background(), insp, gvk) {
			t.Errorf("expected AnyTargetHasGVK(%s) to be false for empty-discovery inspector", gvk)
		}
	}
}

// TestInspectorImplementsContracts is a compile-time-style assertion that the
// stub satisfies all three upstream interfaces exercised at runtime.
func TestInspectorImplementsContracts(t *testing.T) {
	var (
		_ api.ClusterInspector   = New()
		_ api.AggregateDiscovery = New()
		_ api.TargetProvider     = New()
	)
	if New().Unstructured() != nil {
		t.Fatal("expected Unstructured() to return nil")
	}
	targets, err := New().GetTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error from GetTargets: %v", err)
	}
	if len(targets) != 1 || targets[0] != "" {
		t.Fatalf("expected a single empty-named target, got %#v", targets)
	}
}
