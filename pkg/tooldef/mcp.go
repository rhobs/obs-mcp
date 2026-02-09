package tooldef

import "github.com/mark3labs/mcp-go/mcp"

// ToMCPTool converts a ToolDef to an mcp.Tool
func (d ToolDef) ToMCPTool() mcp.Tool {
	opts := []mcp.ToolOption{mcp.WithDescription(d.Description)}

	for _, param := range d.Params {
		switch param.Type {
		case ParamTypeString:
			stringOpts := []mcp.PropertyOption{mcp.Description(param.Description)}
			if param.Required {
				stringOpts = append(stringOpts, mcp.Required())
			}
			if param.Pattern != "" {
				stringOpts = append(stringOpts, mcp.Pattern(param.Pattern))
			}
			opts = append(opts, mcp.WithString(param.Name, stringOpts...))

		case ParamTypeBoolean:
			boolOpts := []mcp.PropertyOption{mcp.Description(param.Description)}
			if param.Required {
				boolOpts = append(boolOpts, mcp.Required())
			}
			opts = append(opts, mcp.WithBoolean(param.Name, boolOpts...))
		}
	}

	tool := mcp.NewTool(d.Name, opts...)

	// Workaround for tools with no parameters
	// See https://github.com/containers/kubernetes-mcp-server/pull/341/files
	if len(d.Params) == 0 {
		tool.InputSchema = mcp.ToolInputSchema{}
		tool.RawInputSchema = []byte(`{"type":"object","properties":{}}`)
	}

	return tool
}
