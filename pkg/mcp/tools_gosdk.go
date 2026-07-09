// This file is adapted from
// https://github.com/containers/kubernetes-mcp-server/blob/3d4a06dcfdfc1c558c83d15902ebc899bb231ca6/pkg/mcp/tools_gosdk.go
// https://github.com/containers/kubernetes-mcp-server/blob/3d4a06dcfdfc1c558c83d15902ebc899bb231ca6/pkg/mcp/mcp.go
// Original code is licensed under the Apache License, Version 2.0.
//
// The copy changes ServerToolToGoSdkTool to accept plain arguments
// (kubernetes.Manager, api.BaseConfig) instead of *Server, decoupling it from
// the kubernetes-mcp-server infrastructure (config hot reload, multi-cluster
// targeting, confirmation rules) and its transitive dependencies.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"
)

func ServerToolToGoSdkTool(mgr *kubernetes.Manager, cfg api.BaseConfig, tool api.ServerTool) (*mcp.Tool, mcp.ToolHandler, error) {
	// Validate the input schema upfront to mirror the SDK's AddTool panic
	// surface. This keeps applyToolsets' two-phase model panic-free at commit
	// time even if a misconfigured tool slips through the toolset boundary.
	inputSchema := tool.Tool.InputSchema
	if inputSchema == nil {
		return nil, nil, fmt.Errorf("tool %q: missing input schema", tool.Tool.Name)
	}
	if inputSchema.Type != "object" {
		return nil, nil, fmt.Errorf("tool %q: input schema must have type %q (got %q)", tool.Tool.Name, "object", inputSchema.Type)
	}
	// Ensure InputSchema.Properties is initialized for OpenAI API compatibility
	// https://github.com/containers/kubernetes-mcp-server/issues/717
	if inputSchema.Properties == nil {
		inputSchema.Properties = make(map[string]*jsonschema.Schema)
	}
	goSdkTool := &mcp.Tool{
		Name:        tool.Tool.Name,
		Description: tool.Tool.Description,
		Title:       tool.Tool.Annotations.Title,
		Meta:        mcp.Meta(tool.Tool.Meta),
		Annotations: &mcp.ToolAnnotations{
			Title:           tool.Tool.Annotations.Title,
			ReadOnlyHint:    ptr.Deref(tool.Tool.Annotations.ReadOnlyHint, false),
			DestructiveHint: tool.Tool.Annotations.DestructiveHint,
			IdempotentHint:  ptr.Deref(tool.Tool.Annotations.IdempotentHint, false),
			OpenWorldHint:   tool.Tool.Annotations.OpenWorldHint,
		},
		InputSchema: inputSchema,
	}
	// A nil *jsonschema.Schema assigned to an "any" field (goSdkTool.OutputSchema)
	// becomes a typed nil (non-nil interface), triggering a panic in AddTool.
	// Therefore, only assign this field when non-nil.
	//
	// Unlike InputSchema above, Properties is intentionally left untouched: the
	// Properties initialization there is an OpenAI-input-specific workaround
	// (#717), whereas the output schema is advertised to MCP clients rather than
	// sent as OpenAI function parameters, so it needs no equivalent.
	outputSchema := tool.Tool.OutputSchema
	if outputSchema != nil {
		if outputSchema.Type != "object" {
			return nil, nil, fmt.Errorf("tool %q: output schema must have type %q (got %q)", tool.Tool.Name, "object", outputSchema.Type)
		}
		goSdkTool.OutputSchema = outputSchema
	}
	goSdkHandler := func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		toolCallRequest, err := GoSdkToolCallRequestToToolCallRequest(request)
		if err != nil {
			return nil, fmt.Errorf("%v for tool %s", err, tool.Tool.Name)
		}

		// get the correct derived Kubernetes client for the target specified in the request
		k, err := mgr.Derived(ctx)
		if err != nil {
			return nil, err
		}

		result, err := tool.Handler(api.ToolHandlerParams{
			Context:          ctx,
			BaseConfig:       cfg,
			KubernetesClient: k,
			ToolCallRequest:  toolCallRequest,
		})
		if err != nil {
			return nil, err
		}
		return NewStructuredResult(result.Content, result.StructuredContent, result.Error), nil
	}
	return goSdkTool, goSdkHandler, nil
}

// NewStructuredResult creates an MCP CallToolResult with structured content.
// The Content field contains the JSON-serialized form of structuredContent
// for backward compatibility with MCP clients that don't support structuredContent.
//
// Per the MCP specification, structuredContent must marshal to a JSON object.
// If structuredContent is a slice/array, it is automatically wrapped in
// {"items": [...]} to satisfy this requirement.
//
// Per the MCP specification:
// "For backwards compatibility, a tool that returns structured content SHOULD
// also return the serialized JSON in a TextContent block."
// https://modelcontextprotocol.io/specification/2025-11-25/server/tools#structured-content
//
// Use this for tools that return typed/structured data that MCP clients can
// parse programmatically.
func NewStructuredResult(content string, structuredContent any, err error) *mcp.CallToolResult {
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: err.Error(),
				},
			},
		}
	}
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: content,
			},
		},
	}
	if structuredContent != nil {
		result.StructuredContent = ensureStructuredObject(structuredContent)
	}
	return result
}

// ensureStructuredObject wraps slice/array values in a {"items": ...} object
// because the MCP specification requires structuredContent to be a JSON object.
// A typed nil slice (e.g. []string(nil)) returns nil to avoid {"items": null}.
// Note: this checks the top-level reflect.Kind, so a pointer-to-slice (*[]T)
// would not be wrapped. All current callers pass value types.
func ensureStructuredObject(v any) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		if rv.IsNil() {
			return nil
		}
		return map[string]any{"items": v}
	}
	if rv.Kind() == reflect.Array {
		return map[string]any{"items": v}
	}
	return v
}

type ToolCallRequest struct {
	Name      string
	arguments map[string]any
}

var _ api.ToolCallRequest = (*ToolCallRequest)(nil)

func GoSdkToolCallRequestToToolCallRequest(request *mcp.CallToolRequest) (*ToolCallRequest, error) {
	toolCallParams, ok := request.GetParams().(*mcp.CallToolParamsRaw)
	if !ok {
		return nil, errors.New("invalid tool call parameters for tool call request")
	}
	return GoSdkToolCallParamsToToolCallRequest(toolCallParams)
}

func GoSdkToolCallParamsToToolCallRequest(toolCallParams *mcp.CallToolParamsRaw) (*ToolCallRequest, error) {
	var arguments map[string]any
	if len(toolCallParams.Arguments) > 0 {
		if err := json.Unmarshal(toolCallParams.Arguments, &arguments); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool call arguments: %w", err)
		}
	}
	return &ToolCallRequest{
		Name:      toolCallParams.Name,
		arguments: arguments,
	}, nil
}

func (t *ToolCallRequest) GetArguments() map[string]any {
	return t.arguments
}

type mcpBaseConfig struct {
	api.BaseConfig
	toolsetConfig api.ExtendedConfig
}

func (m *mcpBaseConfig) GetToolsetConfig(name string) (api.ExtendedConfig, bool) {
	if m.toolsetConfig != nil {
		return m.toolsetConfig, true
	}
	return nil, false
}
