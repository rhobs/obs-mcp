package mcp

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func CreateGetCurrentTimeTool() mcp.Tool {
	tool := mcp.NewTool("get_current_time",
		mcp.WithDescription("Get the current date and time in RFC3339 format"),
	)
	// workaround for tool with no parameter
	// see https://github.com/containers/kubernetes-mcp-server/pull/341/files#diff-8f8a99cac7a7cbb9c14477d40539efa1494b62835603244ba9f10e6be1c7e44c
	tool.InputSchema = mcp.ToolInputSchema{}
	tool.RawInputSchema = []byte(`{"type":"object","properties":{}}`)
	return tool
}

func CurrentTimeHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	currentTime := time.Now().UTC().Format(time.RFC3339)
	return mcp.NewToolResultText(currentTime), nil
}
