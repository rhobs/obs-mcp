package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	mcplib "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rhobs/obs-mcp/pkg/mcp"
)

func main() {
	tools := mcp.AllTools()

	if err := generateMarkdown(tools, "TOOLS.md"); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating TOOLS.md: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ TOOLS.md generated successfully")
	fmt.Printf("  Documented %d tools:\n", len(tools))
	for i := range tools {
		fmt.Printf("    - %s\n", tools[i].Name)
	}
	fmt.Println("\n💡 Reminder: When adding a new tool, register it in pkg/mcp/tools.go AllTools()")
}

type fieldInfo struct {
	Name        string
	Type        string
	Required    bool
	Description string
	Pattern     string
}

// Schema represents a JSON schema with properties and required fields
type Schema struct {
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property represents a JSON schema property
type Property struct {
	Type        any       `json:"type,omitempty"` // can be string or []string
	Description string    `json:"description,omitempty"`
	Pattern     string    `json:"pattern,omitempty"`
	Items       *Property `json:"items,omitempty"`
}

// parseSchema converts the value of any type (interface{}, any) to Schema using JSON marshaling
// and unmarshaling. The reason is that the `Tool` type (https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp#Tool)
// defines outputSchema and inputSchema as values of any type.
func parseSchema(schemaInterface any) (*Schema, error) {
	if schemaInterface == nil {
		return &Schema{}, nil
	}

	data, err := json.Marshal(schemaInterface)
	if err != nil {
		return nil, err
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

// getTypeString extracts type string from Property.Type
func (p *Property) getTypeString() string {
	switch t := p.Type.(type) {
	case string:
		return t
	case []any:
		// Handle nullable types like ["null", "string"]
		for _, typ := range t {
			if typeStr, ok := typ.(string); ok && typeStr != "null" {
				return typeStr
			}
		}
	}
	return ""
}

// getDisplayType returns the display type for the property
func (p *Property) getDisplayType() string {
	baseType := p.getTypeString()
	if baseType == "array" && p.Items != nil {
		itemType := p.Items.getTypeString()
		if itemType != "" {
			return itemType + "[]"
		}
		// For object arrays, just return "object[]"
		return "object[]"
	}
	return baseType
}

// extractFieldsFromSchema converts a Schema to []fieldInfo
func extractFieldsFromSchema(schema *Schema, sortByRequired bool) []fieldInfo {
	if schema == nil || len(schema.Properties) == 0 {
		return nil
	}

	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	var fields []fieldInfo
	for name, prop := range schema.Properties {
		field := fieldInfo{
			Name:        name,
			Type:        prop.getDisplayType(),
			Required:    requiredSet[name],
			Description: prop.Description,
			Pattern:     prop.Pattern,
		}
		fields = append(fields, field)
	}

	if sortByRequired {
		sort.Slice(fields, func(i, j int) bool {
			if fields[i].Required != fields[j].Required {
				return fields[i].Required
			}
			return fields[i].Name < fields[j].Name
		})
	} else {
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Name < fields[j].Name
		})
	}

	return fields
}

func extractParams(tool *mcplib.Tool) []fieldInfo {
	schema, err := parseSchema(tool.InputSchema)
	if err != nil {
		return nil
	}
	return extractFieldsFromSchema(schema, true) // sort by required
}

func extractOutputSchema(tool *mcplib.Tool) []fieldInfo {
	schema, err := parseSchema(tool.OutputSchema)
	if err != nil {
		return nil
	}
	return extractFieldsFromSchema(schema, false) // sort by name only
}

// formatTable generates a formatted markdown table with aligned columns
func formatTable(headers, alignments []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	// Calculate max width for each column
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder

	// Header row
	sb.WriteString("|")
	for i, h := range headers {
		sb.WriteString(fmt.Sprintf(" %-*s |", widths[i], h))
	}
	sb.WriteString("\n")

	// Separator row with alignment
	sb.WriteString("|")
	for i, w := range widths {
		align := "l" // default left
		if i < len(alignments) {
			align = alignments[i]
		}
		switch align {
		case "c": // center
			sb.WriteString(fmt.Sprintf(" :%s: |", strings.Repeat("-", w-2)))
		case "r": // right
			sb.WriteString(fmt.Sprintf(" %s: |", strings.Repeat("-", w-1)))
		default: // left
			sb.WriteString(fmt.Sprintf(" :%s |", strings.Repeat("-", w-1)))
		}
	}
	sb.WriteString("\n")

	// Data rows
	for _, row := range rows {
		sb.WriteString("|")
		for i, cell := range row {
			if i < len(widths) {
				sb.WriteString(fmt.Sprintf(" %-*s |", widths[i], cell))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func generateMarkdown(tools []mcplib.Tool, filename string) error {
	var sb strings.Builder

	sb.WriteString("<!-- This file is auto-generated. Do not edit manually. -->\n")
	sb.WriteString("<!-- Run 'make generate-tools-doc' to regenerate. -->\n\n")

	sb.WriteString("# Available Tools\n\n")
	sb.WriteString("This MCP server exposes the following tools for interacting with Prometheus/Thanos:\n\n")

	for i := range tools {
		tool := &tools[i]
		sb.WriteString(fmt.Sprintf("## `%s`\n\n", tool.Name))

		// Parse description - first paragraph is the main description,
		// subsequent paragraphs become usage tips
		paragraphs := strings.Split(strings.TrimSpace(tool.Description), "\n\n")
		sb.WriteString(fmt.Sprintf("> %s\n\n", strings.TrimSpace(paragraphs[0])))

		if len(paragraphs) > 1 {
			sb.WriteString("**Usage Tips:**\n\n")
			for _, para := range paragraphs[1:] {
				// Join lines within a paragraph and create a single bullet
				lines := strings.Split(para, "\n")
				var joined []string
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						joined = append(joined, line)
					}
				}
				if len(joined) > 0 {
					sb.WriteString(fmt.Sprintf("- %s\n", strings.Join(joined, " ")))
				}
			}
			sb.WriteString("\n")
		}

		// Parameters
		params := extractParams(tool)
		if len(params) == 0 {
			sb.WriteString(formatTable(
				[]string{"", ""},
				[]string{"l", "l"},
				[][]string{{"**Parameters**", "None"}},
			))
			sb.WriteString("\n")
		} else {
			sb.WriteString("**Parameters:**\n\n")
			var rows [][]string
			for _, p := range params {
				req := ""
				if p.Required {
					req = "✅"
				}
				rows = append(rows, []string{
					fmt.Sprintf("`%s`", p.Name),
					fmt.Sprintf("`%s`", p.Type),
					req,
					p.Description,
				})
			}
			sb.WriteString(formatTable(
				[]string{"Parameter", "Type", "Required", "Description"},
				[]string{"l", "l", "c", "l"},
				rows,
			))
			sb.WriteString("\n")

			for _, p := range params {
				if p.Pattern != "" {
					sb.WriteString("> [!NOTE]\n")
					sb.WriteString(fmt.Sprintf("> Parameters with patterns must match: `%s`\n\n", p.Pattern))
					break
				}
			}
		}

		// Output Schema
		outputFields := extractOutputSchema(tool)
		if len(outputFields) > 0 {
			sb.WriteString("**Output Schema:**\n\n")
			var rows [][]string
			for _, f := range outputFields {
				rows = append(rows, []string{
					fmt.Sprintf("`%s`", f.Name),
					fmt.Sprintf("`%s`", f.Type),
					f.Description,
				})
			}
			sb.WriteString(formatTable(
				[]string{"Field", "Type", "Description"},
				[]string{"l", "l", "l"},
				rows,
			))
			sb.WriteString("\n")
		}

		if i < len(tools)-1 {
			sb.WriteString("---\n\n")
		}
	}

	return os.WriteFile(filename, []byte(sb.String()), 0o644)
}
