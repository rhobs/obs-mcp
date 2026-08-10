package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"

	"github.com/rhobs/obs-mcp/pkg/mcp"
)

func main() {
	groups := mcp.GroupedTools()

	if err := generateMarkdown(groups, "TOOLS.md"); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating TOOLS.md: %v\n", err)
		os.Exit(1)
	}

	total := 0
	for _, g := range groups {
		total += len(g.Tools)
	}

	fmt.Println("✓ TOOLS.md generated successfully")
	fmt.Printf("  Documented %d tools in %d categories:\n", total, len(groups))
	for _, g := range groups {
		fmt.Printf("    %s %s (%d tools)\n", g.Icon, g.Name, len(g.Tools))
		for i := range g.Tools {
			fmt.Printf("      - %s\n", g.Tools[i].Tool.Name)
		}
	}
	fmt.Println("\n💡 Reminder: When adding a new tool, register it in the relevant package GetTools() method (metrics, logs, traces); pkg/mcp/tools.go GroupedTools() merges them.")
}

type fieldInfo struct {
	Name        string
	Type        string
	Required    bool
	Description string
	Pattern     string
}

// getDisplayType returns the display type for a jsonschema.Schema property.
func getDisplayType(prop *jsonschema.Schema) string {
	baseType := prop.Type
	if baseType == "" && len(prop.Types) > 0 {
		for _, t := range prop.Types {
			if t != "null" {
				baseType = t
				break
			}
		}
	}
	if baseType == "array" && prop.Items != nil {
		itemType := prop.Items.Type
		if itemType != "" {
			return itemType + "[]"
		}
		return "object[]"
	}
	if baseType == "" {
		return "object"
	}
	return baseType
}

// extractFieldsFromSchema converts a jsonschema.Schema to []fieldInfo.
func extractFieldsFromSchema(schema *jsonschema.Schema, sortByRequired bool) []fieldInfo {
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
			Type:        getDisplayType(prop),
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

func extractParams(tool *api.Tool) []fieldInfo {
	return extractFieldsFromSchema(tool.InputSchema, true)
}

func extractOutputSchema(tool *api.Tool) []fieldInfo {
	return extractFieldsFromSchema(tool.OutputSchema, false)
}

// sanitizeTableCell makes cell content safe for a single-line GFM table row.
func sanitizeTableCell(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", "<br>")
	s = strings.ReplaceAll(s, "|", "&#124;")
	return s
}

// formatTable writes a compact GFM markdown table.
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

// toolAnchor returns the GitHub-compatible markdown anchor for a tool name.
func toolAnchor(name string) string {
	return name
}

// categoryAnchor returns the GitHub-compatible markdown anchor for a category name.
func categoryAnchor(name string) string {
	anchor := strings.ToLower(name)
	anchor = strings.ReplaceAll(anchor, " ", "-")
	anchor = strings.ReplaceAll(anchor, "/", "")
	anchor = strings.ReplaceAll(anchor, "(", "")
	anchor = strings.ReplaceAll(anchor, ")", "")
	anchor = strings.ReplaceAll(anchor, "--", "-")
	return strings.Trim(anchor, "-")
}

// firstSentence extracts the first sentence (up to the first period followed by space or end).
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if idx := strings.Index(s, ". "); idx != -1 {
		return s[:idx+1]
	}
	if strings.HasSuffix(s, ".") {
		return s
	}
	return s
}

func generateMarkdown(groups []mcp.ToolGroup, filename string) error {
	var sb strings.Builder

	sb.WriteString("<!-- This file is auto-generated. Do not edit manually. -->\n")
	sb.WriteString("<!-- Run 'make generate-tools-doc' to regenerate. -->\n\n")

	sb.WriteString("# Available Tools\n\n")
	sb.WriteString("This MCP server exposes the following tools for Prometheus/Thanos, Alertmanager, Loki, Tempo, and OpenTelemetry Collector configuration.\n\n")

	// --- Quick-reference table ---
	sb.WriteString("## Quick Reference\n\n")
	sb.WriteString("| Tool | Category | Description |\n")
	sb.WriteString("| :--- | :--- | :--- |\n")
	for _, g := range groups {
		for i := range g.Tools {
			tool := &g.Tools[i].Tool
			paragraphs := strings.Split(strings.TrimSpace(tool.Description), "\n\n")
			desc := firstSentence(paragraphs[0])
			sb.WriteString(fmt.Sprintf("| [`%s`](#%s) | %s %s | %s |\n",
				tool.Name, toolAnchor(tool.Name), g.Icon, g.Name, sanitizeTableCell(desc)))
		}
	}
	sb.WriteString("\n")

	// --- Type conventions note ---
	sb.WriteString("> [!NOTE]\n")
	sb.WriteString("> **Types in the tables** follow JSON Schema: `object` is a JSON object (string keys with JSON values); `object[]` is an array of those objects. Scalar types use their usual names (`string`, `number`, `boolean`, and so on). When a field has no explicit schema type (for example a Go `any` payload), this document shows `object` as shorthand for \"structured JSON,\" not a guarantee that only objects are returned at runtime.\n\n")

	// --- Table of Contents ---
	sb.WriteString("## Table of Contents\n\n")
	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("- **%s [%s](#%s)** (%d tools)\n",
			g.Icon, g.Name, categoryAnchor(g.Name), len(g.Tools)))
		for i := range g.Tools {
			sb.WriteString(fmt.Sprintf("  - [`%s`](#%s)\n", g.Tools[i].Tool.Name, toolAnchor(g.Tools[i].Tool.Name)))
		}
	}
	sb.WriteString("\n---\n\n")

	// --- Tool sections by category ---
	for gi, g := range groups {
		id := categoryAnchor(g.Name)
		sb.WriteString(fmt.Sprintf("<a id=%q></a>\n\n", id))
		sb.WriteString(fmt.Sprintf("## %s %s\n\n", g.Icon, g.Name))

		for ti := range g.Tools {
			tool := &g.Tools[ti].Tool
			sb.WriteString(fmt.Sprintf("### `%s`\n\n", tool.Name))

			paragraphs := strings.Split(strings.TrimSpace(tool.Description), "\n\n")
			mainDesc := strings.ReplaceAll(strings.TrimSpace(paragraphs[0]), "\n", "\n> ")
			sb.WriteString(fmt.Sprintf("> %s\n\n", mainDesc))

			// Usage tips in a collapsible section
			if len(paragraphs) > 1 {
				sb.WriteString("<details>\n<summary><strong>Usage Tips</strong></summary>\n\n")
				for _, para := range paragraphs[1:] {
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
				sb.WriteString("\n</details>\n\n")
			}

			// Parameters
			params := extractParams(tool)
			if len(params) == 0 {
				sb.WriteString("_No parameters._\n\n")
			} else {
				sb.WriteString("**Parameters:**\n\n")

				var requiredRows, optionalRows [][]string
				for _, p := range params {
					row := []string{
						fmt.Sprintf("`%s`", p.Name),
						fmt.Sprintf("`%s`", p.Type),
						p.Description,
					}
					if p.Required {
						requiredRows = append(requiredRows, row)
					} else {
						optionalRows = append(optionalRows, row)
					}
				}

				if len(requiredRows) > 0 {
					sb.WriteString("**Required:**\n\n")
					sb.WriteString(formatTable(
						[]string{"Parameter", "Type", "Description"},
						[]string{"l", "l", "l"},
						requiredRows,
					))
					sb.WriteString("\n")
				}

				if len(optionalRows) > 0 {
					sb.WriteString("<details>\n<summary><strong>Optional Parameters</strong></summary>\n\n")
					sb.WriteString(formatTable(
						[]string{"Parameter", "Type", "Description"},
						[]string{"l", "l", "l"},
						optionalRows,
					))
					sb.WriteString("\n</details>\n\n")
				}

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
				sb.WriteString("<details>\n<summary><strong>Output Schema</strong></summary>\n\n")
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
				sb.WriteString("\n</details>\n\n")
			}

			// Separator between tools (but not after the very last tool)
			isLastTool := gi == len(groups)-1 && ti == len(g.Tools)-1
			if !isLastTool {
				sb.WriteString("---\n\n")
			}
		}
	}

	return os.WriteFile(filename, []byte(sb.String()), 0o644)
}
