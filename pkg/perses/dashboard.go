package perses

// PersesDashboardInfo contains summary information about a PersesDashboard.
type PersesDashboardInfo struct {
	Name        string            `json:"name" jsonschema:"description=Name of the PersesDashboard"`
	Namespace   string            `json:"namespace" jsonschema:"description=Namespace where the PersesDashboard is located"`
	Labels      map[string]string `json:"labels,omitempty" jsonschema:"description=Labels attached to the PersesDashboard"`
	Description string            `json:"description,omitempty" jsonschema:"description=Human-readable description of the dashboard and what information it contains (from operator.perses.dev/mcp-help annotation)"`
}
