package perses

import (
	"fmt"
	"log/slog"
)

// ExtractPanels extracts panel information from a dashboard spec.
// If panelIDs is provided, only extracts those specific panels.
func ExtractPanels(dashboardName, dashboardNamespace string, spec map[string]any, panelIDs []string) (panels []*DashboardPanel) {
	panelsMap, ok := spec["panels"].(map[string]any)
	if !ok {
		slog.Debug("No panels found in dashboard spec", "dashboard", dashboardName)
		return panels
	}

	// Build a lookup set for requested panel IDs
	requestedIDs := make(map[string]bool)
	for _, id := range panelIDs {
		requestedIDs[id] = true
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

			panel := &DashboardPanel{
				ID:          panelID,
				Title:       title,
				Description: description,
				Query:       query,
				ChartType:   chartType,
			}

			panels = append(panels, panel)
		}
	}

	slog.Debug("Extracted panels from dashboard",
		"dashboard", dashboardName,
		"namespace", dashboardNamespace,
		"panelCount", len(panels))

	return panels
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

	return kind
}

// extractQueries extracts all PromQL query strings from a panel
func extractQueries(panelMap map[string]any) []string {
	var queries []string

	queriesArray, ok := panelMap["queries"].([]any)
	if !ok {
		return queries
	}

	for _, queryData := range queriesArray {
		queryMap, ok := queryData.(map[string]any)
		if !ok {
			continue
		}

		if q := extractSingleQuery(queryMap); q != "" {
			queries = append(queries, q)
		}
	}

	return queries
}

// extractSingleQuery extracts the PromQL query string from a query spec.
//
//	structure: { spec: { plugin: { spec: { query: "..." } } } }
func extractSingleQuery(queryMap map[string]any) string {
	spec, ok := queryMap["spec"].(map[string]any)
	if !ok {
		return ""
	}

	plugin, ok := spec["plugin"].(map[string]any)
	if !ok {
		return ""
	}

	pluginSpec, ok := plugin["spec"].(map[string]any)
	if !ok {
		return ""
	}

	query, _ := pluginSpec["query"].(string)
	return query
}
