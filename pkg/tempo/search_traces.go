package tempo

import (
	"fmt"

	"github.com/rhobs/obs-mcp/pkg/resultutil"
	tempoclient "github.com/rhobs/obs-mcp/pkg/tempo/client"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

var SearchTracesTool = tools.ToolDef{
	Name: "tempo_search_traces",
	Description: `Search for distributed traces in Tempo using TraceQL.
Use this tool to find traces matching specific criteria such as service name, HTTP status code, duration, or other span or resource attributes.`,
	Title: "Search traces",
	Params: []tools.ParamDef{
		tempoNamespaceParameter,
		tempoNameParameter,
		tempoTenantParameter,
		{
			Name: "query",
			Type: tools.ParamTypeString,
			Description: `A TraceQL query expression. Examples:
all traces: {}
by service: { resource.service.name="frontend" }
by status: { span.http.response.status_code=500 }
by duration: { duration>1s }
combined conditions: { resource.service.name="frontend" && span.http.response.status_code>=400 }`,
			Required: true,
		},
		{
			Name:        "limit",
			Type:        tools.ParamTypeNumber,
			Description: "Maximum number of traces to return. Defaults to the server-side limit if not specified.",
		},
		{
			Name: "start",
			Type: tools.ParamTypeString,
			Description: `Start of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".
Use "NOW" for current time.
Both start and end should be provided to search the full time range; if omitted, only a small window of recent data is searched.`,
		},
		{
			Name: "end",
			Type: tools.ParamTypeString,
			Description: `End of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".
Use "NOW" for current time.
Both start and end should be provided to search the full time range; if omitted, only a small window of recent data is searched.`,
		},
		{
			Name:        "spss",
			Type:        tools.ParamTypeNumber,
			Description: "Maximum number of matching spans to return per trace.",
		},
	},
	ReadOnly:    true,
	Destructive: false,
	Idempotent:  true,
	OpenWorld:   true,
}

func (t *Toolset) SearchTracesHandler(params ToolParams) *resultutil.Result {
	client, err := t.getTempoClient(params)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	args := params.arguments

	query := tools.GetString(args, "query", "")
	if query == "" {
		return resultutil.NewErrorResult(fmt.Errorf("query parameter must not be empty"))
	}

	start, err := parseDate(tools.GetString(args, "start", ""))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("invalid start time: %v", err))
	}

	end, err := parseDate(tools.GetString(args, "end", ""))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("invalid end time: %v", err))
	}

	limit := tools.GetInt(args, "limit", 0)
	spss := tools.GetInt(args, "spss", 0)

	opts := tempoclient.SearchOptions{
		Query: query,
		Limit: limit,
		Start: start,
		End:   end,
		Spss:  spss,
	}

	trace, err := client.Search(params.context, opts)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	return resultutil.NewJSONSuccessResult(trace)
}
