package tempo

import (
	"fmt"

	"github.com/rhobs/obs-mcp/pkg/resultutil"
	tempoclient "github.com/rhobs/obs-mcp/pkg/tempo/client"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

var SearchTagsTool = tools.ToolDef{
	Name:        "tempo_search_tags",
	Description: "Search for tag names in Tempo",
	Title:       "Search tags",
	Params: []tools.ParamDef{
		tempoNamespaceParameter,
		tempoNameParameter,
		tempoTenantParameter,
		{
			Name:        "scope",
			Type:        tools.ParamTypeString,
			Description: "Scope to filter tags: resource, span, intrinsic, event, link, or instrumentation",
		},
		{
			Name:        "query",
			Type:        tools.ParamTypeString,
			Description: "TraceQL query for filtering tag names",
		},
		{
			Name:        "start",
			Type:        tools.ParamTypeString,
			Description: "Start time in RFC 3339 format",
		},
		{
			Name:        "end",
			Type:        tools.ParamTypeString,
			Description: "End time in RFC 3339 format",
		},
		{
			Name:        "limit",
			Type:        tools.ParamTypeNumber,
			Description: "Maximum number of tag names per scope",
		},
		{
			Name:        "maxStaleValues",
			Type:        tools.ParamTypeNumber,
			Description: "Search termination threshold for stale values",
		},
	},
	ReadOnly:    true,
	Destructive: false,
	Idempotent:  true,
	OpenWorld:   true,
}

func (t *Toolset) SearchTagsHandler(params ToolParams) *resultutil.Result {
	client, err := t.getTempoClient(params)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	args := params.arguments

	start, err := parseDate(tools.GetString(args, "start", ""))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("invalid start time: %v", err))
	}

	end, err := parseDate(tools.GetString(args, "end", ""))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("invalid end time: %v", err))
	}

	scope := tools.GetString(args, "scope", "")
	query := tools.GetString(args, "query", "")
	limit := tools.GetInt(args, "limit", 0)
	maxStaleValues := tools.GetInt(args, "maxStaleValues", 0)

	opts := tempoclient.SearchTagsV2Options{
		Scope:          scope,
		Query:          query,
		Start:          start,
		End:            end,
		Limit:          limit,
		MaxStaleValues: maxStaleValues,
	}

	result, err := client.SearchTagsV2(params.context, opts)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	return resultutil.NewJSONSuccessResult(result)
}
