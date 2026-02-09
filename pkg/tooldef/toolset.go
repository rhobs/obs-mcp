package tooldef

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

// ToServerTool converts a ToolDef to an api.ServerTool
func (d ToolDef) ToServerTool(handler func(api.ToolHandlerParams) (*api.ToolCallResult, error)) api.ServerTool {
	properties := make(map[string]*jsonschema.Schema)
	var required []string

	for _, param := range d.Params {
		schema := &jsonschema.Schema{
			Description: param.Description,
		}

		switch param.Type {
		case ParamTypeString:
			schema.Type = "string"
			if param.Pattern != "" {
				schema.Pattern = param.Pattern
			}
		case ParamTypeBoolean:
			schema.Type = "boolean"
		}

		properties[param.Name] = schema

		if param.Required {
			required = append(required, param.Name)
		}
	}

	inputSchema := &jsonschema.Schema{
		Type:       "object",
		Properties: properties,
	}

	if len(required) > 0 {
		inputSchema.Required = required
	}

	return api.ServerTool{
		Tool: api.Tool{
			Name:        d.Name,
			Description: d.Description,
			InputSchema: inputSchema,
			Annotations: api.ToolAnnotations{
				Title:           d.Title,
				ReadOnlyHint:    ptr.To(d.ReadOnly),
				DestructiveHint: ptr.To(d.Destructive),
				IdempotentHint:  ptr.To(d.Idempotent),
				OpenWorldHint:   ptr.To(d.OpenWorld),
			},
		},
		Handler: handler,
	}
}
