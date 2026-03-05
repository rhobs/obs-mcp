package tempo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/rhobs/obs-mcp/pkg/tempo/discovery"
)

var (
	tempoStackGVR = schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempostacks",
	}
	tempoMonolithicGVR = schema.GroupVersionResource{
		Group:    "tempo.grafana.com",
		Version:  "v1alpha1",
		Resource: "tempomonolithics",
	}
)

func newTempoStack(namespace, name string, tenants []string) *unstructured.Unstructured {
	auth := make([]any, 0, len(tenants))
	for _, t := range tenants {
		auth = append(auth, map[string]any{"tenantName": t})
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "tempo.grafana.com",
		Version: "v1alpha1",
		Kind:    "TempoStack",
	})
	obj.SetNamespace(namespace)
	obj.SetName(name)
	obj.Object["spec"] = map[string]any{
		"tenants": map[string]any{
			"mode":           "openshift",
			"authentication": auth,
		},
	}
	obj.Object["status"] = map[string]any{
		"conditions": []any{
			map[string]any{
				"type":   "Ready",
				"status": string(metav1.ConditionTrue),
			},
		},
	}
	return obj
}

func newFakeDynamicClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			tempoStackGVR:      "TempoStackList",
			tempoMonolithicGVR: "TempoMonolithicList",
		},
		objects...,
	)
}

func TestListInstancesHandler_Success(t *testing.T) {
	fakeClient := newFakeDynamicClient(
		newTempoStack("ns1", "stack1", []string{"tenant-a", "tenant-b"}),
		newTempoStack("ns2", "stack2", []string{"tenant-c"}),
	)

	toolset := &Toolset{}
	result := toolset.ListInstancesHandler(ToolParams{
		context:       t.Context(),
		dynamicClient: fakeClient,
		config:        &Config{UseRoute: false},
	})
	require.False(t, result.IsError(), "unexpected error: %v", result.Error)

	var output struct {
		Instances []discovery.TempoInstance `json:"instances"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.JSONText), &output))
	require.Len(t, output.Instances, 2)

	inst := output.Instances[0]
	require.Equal(t, "ns1", inst.Namespace)
	require.Equal(t, "stack1", inst.Name)
	require.Equal(t, []string{"tenant-a", "tenant-b"}, inst.Tenants)
	require.Equal(t, "Ready", inst.Status)

	inst2 := output.Instances[1]
	require.Equal(t, "ns2", inst2.Namespace)
	require.Equal(t, "stack2", inst2.Name)
}
