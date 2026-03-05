package tempo

import (
	"fmt"

	"github.com/rhobs/obs-mcp/pkg/resultutil"
	tempoclient "github.com/rhobs/obs-mcp/pkg/tempo/client"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

var SearchTracesTool = tools.ToolDef{
	Name:        "tempo_search_traces",
	Description: "Search for traces in Tempo",
	Title:       "Search traces",
	Params: []tools.ParamDef{
		tempoNamespaceParameter,
		tempoNameParameter,
		tempoTenantParameter,
		{
			Name:        "query",
			Type:        tools.ParamTypeString,
			Description: "Search query in the TraceQL query language",
			Required:    true,
		},
		{
			Name:        "limit",
			Type:        tools.ParamTypeNumber,
			Description: "Maximum search results",
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
			Name:        "spss",
			Type:        tools.ParamTypeNumber,
			Description: "Spans per span-set limit",
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
