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
	fmt.Println("\n💡 Reminder: When adding a new tool, register it in pkg/tools/definitions.go (or add it to AllTools() in pkg/traces for Tempo); pkg/mcp/tools.go AllTools() merges them.")
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
	if baseType == "" {
		return "object"
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

// sanitizeTableCell makes cell content safe for a single-line GFM table row:
// newlines would otherwise break the table; pipe characters would split columns.
func sanitizeTableCell(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", "<br>")
	s = strings.ReplaceAll(s, "|", "&#124;")
	return s
}

// formatTable writes a compact GFM markdown table (no fixed-width padding).
// Padding every cell to the widest description made each row as long as the
// longest row in the file and hurt readability and diffs.
func formatTable(headers, alignments []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	sanitizedHeaders := make([]string, len(headers))
	for i := range headers {
		sanitizedHeaders[i] = sanitizeTableCell(headers[i])
	}
	sanitizedRows := make([][]string, len(rows))
	for ri, row := range rows {
		sanitizedRows[ri] = make([]string, len(headers))
		for ci := range headers {
			cell := ""
			if ci < len(row) {
				cell = row[ci]
			}
			sanitizedRows[ri][ci] = sanitizeTableCell(cell)
		}
	}

	var sb strings.Builder

	sb.WriteString("|")
	for _, h := range sanitizedHeaders {
		sb.WriteString(" ")
		sb.WriteString(h)
		sb.WriteString(" |")
	}
	sb.WriteString("\n|")
	for i := range sanitizedHeaders {
		align := "l"
		if i < len(alignments) {
			align = alignments[i]
		}
		switch align {
		case "c":
			sb.WriteString(" :---: |")
		case "r":
			sb.WriteString(" ---: |")
		default:
			sb.WriteString(" :--- |")
		}
	}
	sb.WriteString("\n")

	for _, row := range sanitizedRows {
		sb.WriteString("|")
		for _, cell := range row {
			sb.WriteString(" ")
			sb.WriteString(cell)
			sb.WriteString(" |")
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
	sb.WriteString("This MCP server exposes the following tools for Prometheus/Thanos, Alertmanager, and Tempo:\n\n")
	sb.WriteString("> [!NOTE]\n")
	sb.WriteString("> **Types in the tables** follow JSON Schema: `object` is a JSON object (string keys with JSON values); `object[]` is an array of those objects. Scalar types use their usual names (`string`, `number`, `boolean`, and so on). When a field has no explicit schema type (for example a Go `any` payload), this document shows `object` as shorthand for \"structured JSON,\" not a guarantee that only objects are returned at runtime.\n\n")

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
