package perses

// DashboardInfo contains metadata about a Perses Dashboard.
type DashboardInfo struct {
	Name        string            `json:"name" jsonschema:"description=Name of the Dashboard"`
	Namespace   string            `json:"namespace" jsonschema:"description=Namespace where the Dashboard is located"`
	Labels      map[string]string `json:"labels,omitempty" jsonschema:"description=Labels attached to the Dashboard"`
	Description string            `json:"description,omitempty" jsonschema:"description=Human-readable description of the dashboard and what information it contains (from operator.perses.dev/mcp-help annotation)"`
}

// DashboardPanel is an intermediary representation of a dashboard panel's metadata and query information.
// We maintain this alongside DashboardWidget to separate concerns between data extraction and UI rendering.
// More importantly, this helps curb the blast radius of changes if the DashboardWidget interface evolves, which we expect it will (during the genie-plugin to genie-web-client migration).
type DashboardPanel struct {
	// Needed to build context for identification and selection
	ID          string `json:"id" jsonschema:"description=Unique identifier for the panel in format 'panelName' or 'panelName-N' where N is the query index for multi-query panels"`
	Title       string `json:"title,omitempty" jsonschema:"description=Human-readable title of the panel extracted from panel.display.name"`
	Description string `json:"description,omitempty" jsonschema:"description=Description of what the panel displays extracted from panel.display.description"`
	Query       string `json:"query" jsonschema:"description=PromQL query string for fetching data"`
	ChartType   string `json:"chartType,omitempty" jsonschema:"description=Type of chart to render (TimeSeriesChart, PieChart, StatChart, Table, etc)"`

	// Needed for UI rendering (only populated when fullDetails=true in ExtractPanels)
	Duration string         `json:"duration,omitempty" jsonschema:"description=Time duration for the query (e.g. 1h, 24h, 7d), extracted from dashboard spec or defaults to 1h"`
	Start    string         `json:"start,omitempty" jsonschema:"description=Optional explicit start time as RFC3339 or Unix timestamp"`
	End      string         `json:"end,omitempty" jsonschema:"description=Optional explicit end time as RFC3339 or Unix timestamp"`
	Step     string         `json:"step,omitempty" jsonschema:"description=Query resolution step width (e.g. 15s, 1m, 5m), extracted from query spec if available"`
	Position *PanelPosition `json:"position,omitempty" jsonschema:"description=Layout position information extracted from dashboard layout spec (only when fullDetails=true)"`
}

// PanelPosition defines the layout position of a panel in a 24-column grid system.
type PanelPosition struct {
	X int `json:"x" jsonschema:"description=X coordinate in 24-column grid"`
	Y int `json:"y" jsonschema:"description=Y coordinate in grid"`
	W int `json:"w" jsonschema:"description=Width in grid units (out of 24 columns)"`
	H int `json:"h" jsonschema:"description=Height in grid units"`
}

// DashboardWidget represents a dashboard widget in the format expected by genie-plugin UI.
// This matches the DashboardWidget interface from jhadvig/genie-plugin.
type DashboardWidget struct {
	ID            string               `json:"id" jsonschema:"description=Unique identifier for the widget"`
	ComponentType string               `json:"componentType" jsonschema:"description=Type of Perses component to render (PersesTimeSeries, PersesPieChart, PersesTable)"`
	Position      PanelPosition        `json:"position" jsonschema:"description=Layout position in 24-column grid (optional, included when available from dashboard layout)"`
	Props         DashboardWidgetProps `json:"props" jsonschema:"description=Properties passed to the Perses component"`
	Breakpoint    string               `json:"breakpoint" jsonschema:"description=Responsive grid breakpoint (xl/lg/md/sm) inferred from panel width, defaults to lg if position unavailable"`
}

// DashboardWidgetProps contains the properties passed to Perses components.
type DashboardWidgetProps struct {
	Query    string `json:"query" jsonschema:"description=PromQL query string"`
	Duration string `json:"duration" jsonschema:"description=Time duration for the query (e.g. 1h, 24h), defaults to 1h if not specified in dashboard"`
	Start    string `json:"start,omitempty" jsonschema:"description=Optional explicit start time as RFC3339 or Unix timestamp"`
	End      string `json:"end,omitempty" jsonschema:"description=Optional explicit end time as RFC3339 or Unix timestamp"`
	Step     string `json:"step" jsonschema:"description=Query resolution step width (e.g. 15s, 1m, 5m), defaults to 15s if not specified in dashboard"`
}
