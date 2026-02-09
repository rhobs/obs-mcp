package tooldef

// ToolDef defines a tool that can be converted to different formats (MCP, Toolset, etc.)
type ToolDef struct {
	Name        string
	Description string
	Title       string
	Params      []ParamDef
	ReadOnly    bool
	Destructive bool
	Idempotent  bool
	OpenWorld   bool
}

// ParamDef defines a tool parameter
type ParamDef struct {
	Name        string
	Type        ParamType
	Description string
	Required    bool
	Pattern     string
}

// ParamType represents the type of a parameter
type ParamType string

const (
	ParamTypeString  ParamType = "string"
	ParamTypeBoolean ParamType = "boolean"
)
