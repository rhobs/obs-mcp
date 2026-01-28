package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"

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

func extractParams(tool *mcplib.Tool) []fieldInfo {
	requiredSet := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		requiredSet[r] = true
	}

	var params []fieldInfo
	for name, prop := range tool.InputSchema.Properties {
		p := fieldInfo{
			Name:     name,
			Required: requiredSet[name],
		}
		if propMap, ok := prop.(map[string]any); ok {
			if t, ok := propMap["type"].(string); ok {
				p.Type = t
			}
			if d, ok := propMap["description"].(string); ok {
				p.Description = d
			}
			if pat, ok := propMap["pattern"].(string); ok {
				p.Pattern = pat
			}
		}
		params = append(params, p)
	}

	sort.Slice(params, func(i, j int) bool {
		if params[i].Required != params[j].Required {
			return params[i].Required
		}
		return params[i].Name < params[j].Name
	})

	return params
}

func extractOutputSchema(tool *mcplib.Tool) []fieldInfo {
	var fields []fieldInfo

	if len(tool.OutputSchema.Properties) == 0 {
		return fields
	}

	requiredSet := make(map[string]bool)
	for _, r := range tool.OutputSchema.Required {
		requiredSet[r] = true
	}

	for name, prop := range tool.OutputSchema.Properties {
		f := fieldInfo{
			Name:     name,
			Required: requiredSet[name],
		}
		if propMap, ok := prop.(map[string]any); ok {
			if t, ok := propMap["type"].(string); ok {
				f.Type = t
			}
			if f.Type == "array" {
				if items, ok := propMap["items"].(map[string]any); ok {
					if itemType, ok := items["type"].(string); ok {
						f.Type = itemType + "[]"
					} else if _, ok := items["properties"]; ok {
						f.Type = "object[]"
					}
				}
			}
			if d, ok := propMap["description"].(string); ok {
				f.Description = d
			}
		}
		fields = append(fields, f)
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	return fields
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
