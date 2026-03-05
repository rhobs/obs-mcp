package tempo

import (
	"fmt"

	"github.com/rhobs/obs-mcp/pkg/resultutil"
	tempoclient "github.com/rhobs/obs-mcp/pkg/tempo/client"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

var SearchTagValuesTool = tools.ToolDef{
	Name:        "tempo_search_tag_values",
	Description: "Search for tag values in Tempo",
	Title:       "Search tag values",
	Params: []tools.ParamDef{
		tempoNamespaceParameter,
		tempoNameParameter,
		tempoTenantParameter,
		{
			Name:        "tag",
			Type:        tools.ParamTypeString,
			Description: "The tag name to get values for",
			Required:    true,
		},
		{
			Name:        "query",
			Type:        tools.ParamTypeString,
			Description: "TraceQL query for filtering tag values",
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
			Description: "Maximum number of tag values to return",
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
