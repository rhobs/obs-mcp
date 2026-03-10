package tempo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rhobs/obs-mcp/pkg/tempo/discovery"
)

func TestListInstancesHandler_Success(t *testing.T) {
	fakeClient := newMockK8sClient(
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
