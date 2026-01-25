package perses

import (
	"fmt"
	"log/slog"
)

// ExtractPanels extracts panel information from a dashboard spec.
// If fullDetails is true, includes position, step, and duration for UI rendering.
// If panelIDs is provided, only extracts those specific panels.
// TODO: Sometimes, the dashboard description may be present in a dedicated panel rather than the dashboard metadata. Consider extracting that as well.
func ExtractPanels(dashboardName, dashboardNamespace string, spec map[string]any, fullDetails bool, panelIDs []string) ([]DashboardPanel, error) {
	var panels []DashboardPanel

	panelsMap, ok := spec["panels"].(map[string]any)
	if !ok {
		slog.Debug("No panels found in dashboard spec", "dashboard", dashboardName)
		return panels, nil
	}

	// Build a lookup set for requested panel IDs
	requestedIDs := make(map[string]bool)
	for _, id := range panelIDs {
		requestedIDs[id] = true
	}

	// Extract additional details only if needed
	var defaultDuration string
	var layoutMap map[string]PanelPosition
	if fullDetails {
		defaultDuration = extractDefaultDuration(spec)
		layoutMap = extractLayoutPositions(spec)
	}

	// Process each panel
	for panelName, panelData := range panelsMap {
		panelMap, ok := panelData.(map[string]any)
		if !ok {
			continue
		}

		// Extract the spec from the Panel wrapper
		// Structure: { kind: "Panel", spec: { display, plugin, queries } }
		spec, ok := panelMap["spec"].(map[string]any)
		if !ok {
			// Try without wrapper for backwards compatibility
			spec = panelMap
		}

		// Get basic panel info
		title, description := extractDisplayInfo(spec)
		chartType := extractChartType(spec)
		queries := extractQueries(spec)

		// Create a panel for each query
		for i, query := range queries {
			panelID := panelName
			if len(queries) > 1 {
				panelID = fmt.Sprintf("%s-%d", panelName, i)
			}

			// Skip if filtering and this panel wasn't requested
			if len(requestedIDs) > 0 && !requestedIDs[panelID] {
				continue
			}

			panel := DashboardPanel{
				ID:          panelID,
				Title:       title,
				Description: description,
				Query:       query.Query,
				ChartType:   chartType,
			}

			// Add full details only if requested
			if fullDetails {
				panel.Duration = defaultDuration
				panel.Step = query.Step
				if pos, ok := layoutMap[panelName]; ok {
					panel.Position = &pos
				}
			}

			panels = append(panels, panel)
		}
	}

	slog.Debug("Extracted panels from dashboard",
		"dashboard", dashboardName,
		"namespace", dashboardNamespace,
		"fullDetails", fullDetails,
		"panelCount", len(panels))

	return panels, nil
}

// panelQuery represents a query extracted from a panel
type panelQuery struct {
	Query string
	Step  string
}

// extractDisplayInfo extracts title and description from a panel's display section
func extractDisplayInfo(panelMap map[string]any) (title, description string) {
	display, ok := panelMap["display"].(map[string]any)
	if !ok {
		return "", ""
	}

	if name, ok := display["name"].(string); ok {
		title = name
	}
	if desc, ok := display["description"].(string); ok {
		description = desc
	}
	return title, description
}

// extractChartType extracts and maps the chart type from a panel's plugin section
func extractChartType(panelMap map[string]any) string {
	plugin, ok := panelMap["plugin"].(map[string]any)
	if !ok {
		return ""
	}

	kind, ok := plugin["kind"].(string)
	if !ok {
		return ""
	}

	return mapKindToChartType(kind)
}

// extractQueries extracts all queries from a panel
func extractQueries(panelMap map[string]any) []panelQuery {
	var queries []panelQuery

	queriesArray, ok := panelMap["queries"].([]any)
	if !ok {
		return queries
	}

	for _, queryData := range queriesArray {
		queryMap, ok := queryData.(map[string]any)
		if !ok {
			continue
		}

		if pq := extractSingleQuery(queryMap); pq.Query != "" {
			queries = append(queries, pq)
		}
	}

	return queries
}

// extractSingleQuery extracts query and step from a  query spec.
//
//	structure: { spec: { plugin: { spec: { query: "...", step: "..." } } } }
func extractSingleQuery(queryMap map[string]any) panelQuery {
	spec, ok := queryMap["spec"].(map[string]any)
	if !ok {
		return panelQuery{}
	}

	plugin, ok := spec["plugin"].(map[string]any)
	if !ok {
		return panelQuery{}
	}

	pluginSpec, ok := plugin["spec"].(map[string]any)
	if !ok {
		return panelQuery{}
	}

	pq := panelQuery{}
	if query, ok := pluginSpec["query"].(string); ok {
		pq.Query = query
	}
	if step, ok := pluginSpec["step"].(string); ok {
		pq.Step = step
	}

	return pq
}

// extractDefaultDuration extracts the default duration from dashboard spec
func extractDefaultDuration(spec map[string]any) string {
	if duration, ok := spec["duration"].(string); ok {
		return duration
	}
	return "1h"
}

// extractLayoutPositions extracts panel positions from the dashboard layout
func extractLayoutPositions(spec map[string]any) map[string]PanelPosition {
	positions := make(map[string]PanelPosition)

	layouts, ok := spec["layouts"].([]any)
	if !ok {
		return positions
	}

	for _, layoutData := range layouts {
		layoutMap, ok := layoutData.(map[string]any)
		if !ok {
			continue
		}

		layoutSpec, ok := layoutMap["spec"].(map[string]any)
		if !ok {
			continue
		}

		items, ok := layoutSpec["items"].([]any)
		if !ok {
			continue
		}

		for _, itemData := range items {
			itemMap, ok := itemData.(map[string]any)
			if !ok {
				continue
			}

			x, xOk := getInt(itemMap["x"])
			y, yOk := getInt(itemMap["y"])
			w, wOk := getInt(itemMap["width"])
			h, hOk := getInt(itemMap["height"])

			if !xOk || !yOk || !wOk || !hOk {
				continue
			}

			// Extract panel reference from content.$ref
			content, ok := itemMap["content"].(map[string]any)
			if !ok {
				continue
			}

			ref, ok := content["$ref"].(string)
			if !ok {
				continue
			}

			// - x: 4
			//	 "y": 1
			//	 width: 4
			//	 height: 3
			//	 content:
			//		 $ref: "#/spec/panels/0_1"
			panelName := extractPanelNameFromRef(ref)
			if panelName != "" {
				positions[panelName] = PanelPosition{X: x, Y: y, W: w, H: h}
			}
		}
	}

	return positions
}

// extractPanelNameFromRef extracts panel name from a JSON reference
// Example: "#/spec/panels/panelName" -> "panelName"
func extractPanelNameFromRef(ref string) string {
	const prefix = "#/spec/panels/"
	if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
		return ref[len(prefix):]
	}
	return ""
}

// mapKindToChartType maps  plugin kinds to UI chart types
func mapKindToChartType(persesKind string) string {
	switch persesKind {
	case "TimeSeriesChart", "BarChart":
		return "TimeSeriesChart"
	case "StatChart", "GaugeChart":
		return "PieChart"
	case "Table":
		return "Table"
	default:
		return persesKind
	}
}

func getInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	default:
		return 0, false
	}
}
