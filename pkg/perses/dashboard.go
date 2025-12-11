package perses

// DashboardInfo contains metadata about a Perses Dashboard.
type DashboardInfo struct {
	Name        string            `json:"name" jsonschema:"Name of the Dashboard"`
	Namespace   string            `json:"namespace" jsonschema:"Namespace where the Dashboard is located"`
	Labels      map[string]string `json:"labels,omitempty" jsonschema:"Labels attached to the Dashboard"`
	Description string            `json:"description,omitempty" jsonschema:"Human-readable description of the dashboard"`
}

// DashboardPanel represents a panel extracted from a Perses dashboard, containing
// the metadata and query information needed for the LLM to select relevant panels
// and pass their queries to show_timeseries for visualization.
type DashboardPanel struct {
	ID          string `json:"id" jsonschema:"Unique identifier for the panel"`
	Title       string `json:"title,omitempty" jsonschema:"Human-readable title of the panel"`
	Description string `json:"description,omitempty" jsonschema:"Description of what the panel displays"`
	Query       string `json:"query" jsonschema:"PromQL query string for fetching data"`
	ChartType   string `json:"chartType,omitempty" jsonschema:"Type of chart to render (TimeSeriesChart, PieChart, StatChart, Table, etc)"`
}
