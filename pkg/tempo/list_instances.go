package tempo

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListInstancesTool() mcp.Tool {
	tool := mcp.NewTool("tempo_list_instances",
		mcp.WithDescription("List all Tempo instances. The assistant should display the instances in a table."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	// workaround for tool with no parameter
	// see https://github.com/containers/kubernetes-mcp-server/pull/341/files#diff-8f8a99cac7a7cbb9c14477d40539efa1494b62835603244ba9f10e6be1c7e44c
	tool.InputSchema = mcp.ToolInputSchema{}
	tool.RawInputSchema = []byte(`{"type":"object","properties":{}}`)
	return tool
}
func (t *TempoToolset) ListInstancesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	instances, err := t.discovery.ListInstances(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultStructuredOnly(map[string]any{
		"instances": instances,
	}), nil
}
