package tempo

import (
	"fmt"

	"github.com/rhobs/obs-mcp/pkg/resultutil"
	tempoclient "github.com/rhobs/obs-mcp/pkg/tempo/client"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

var SearchTagValuesTool = tools.ToolDef{
	Name: "tempo_search_tag_values",
	Description: `List the known values for a specific tag (attribute key) in Tempo.
Use this tool to discover what values exist for a given tag, e.g. to find all service names (values of "resource.service.name") or all HTTP methods (values of "span.http.request.method").
This is useful for building accurate TraceQL queries with tempo_search_traces.`,
	Title: "Search tag values",
	Params: []tools.ParamDef{
		tempoNamespaceParameter,
		tempoNameParameter,
		tempoTenantParameter,
		{
			Name: "tag",
			Type: tools.ParamTypeString,
			Description: `The fully qualified tag name to get values for, including its scope prefix, e.g. "resource.service.name" or "span.http.response.status_code".
Use tempo_search_tags to discover available tag names.`,
			Required: true,
		},
		{
			Name: "query",
			Type: tools.ParamTypeString,
			Description: `Optional TraceQL query to filter which traces are considered when listing values,
e.g. '{ resource.service.name="payment-service" }' to only show tag values from the 'payment-service' service.`,
		},
		{
			Name:        "start",
			Type:        tools.ParamTypeString,
			Description: `Optional start of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing values.`,
		},
		{
			Name:        "end",
			Type:        tools.ParamTypeString,
			Description: `Optional end of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing values.`,
		},
		{
			Name:        "limit",
			Type:        tools.ParamTypeNumber,
			Description: "Maximum number of tag values to return.",
		},
		{
			Name:        "maxStaleValues",
			Type:        tools.ParamTypeNumber,
			Description: "Maximum number of consecutive blocks without new values before the search stops early. Higher values are more thorough but slower.",
		},
	},
	ReadOnly:    true,
	Destructive: false,
	Idempotent:  true,
	OpenWorld:   true,
}

func (t *Toolset) SearchTagValuesHandler(params ToolParams) *resultutil.Result {
	client, err := t.getTempoClient(params)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	args := params.arguments

	tag := tools.GetString(args, "tag", "")
	if tag == "" {
		return resultutil.NewErrorResult(fmt.Errorf("tag parameter must not be empty"))
	}

	start, err := parseDate(tools.GetString(args, "start", ""))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("invalid start time: %v", err))
	}

	end, err := parseDate(tools.GetString(args, "end", ""))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("invalid end time: %v", err))
	}

	query := tools.GetString(args, "query", "")
	limit := tools.GetInt(args, "limit", 0)
	maxStaleValues := tools.GetInt(args, "maxStaleValues", 0)

	opts := tempoclient.SearchTagValuesV2Options{
		Query:          query,
		Start:          start,
		End:            end,
		Limit:          limit,
		MaxStaleValues: maxStaleValues,
	}

	result, err := client.SearchTagValuesV2(params.context, tag, opts)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	return resultutil.NewJSONSuccessResult(result)
}
