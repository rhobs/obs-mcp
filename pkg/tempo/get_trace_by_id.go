package tempo

import (
	"fmt"

	"github.com/rhobs/obs-mcp/pkg/resultutil"
	tempoclient "github.com/rhobs/obs-mcp/pkg/tempo/client"
	"github.com/rhobs/obs-mcp/pkg/tools"
)

var GetTraceByIdTool = tools.ToolDef{
	Name:        "tempo_get_trace_by_id",
	Description: "Get a trace by trace ID",
	Title:       "Get trace by ID",
	Params: []tools.ParamDef{
		tempoNamespaceParameter,
		tempoNameParameter,
		tempoTenantParameter,
		{
			Name:        "traceid",
			Type:        tools.ParamTypeString,
			Description: "TraceID of the trace",
			Required:    true,
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
	},
	ReadOnly:    true,
	Destructive: false,
	Idempotent:  true,
	OpenWorld:   true,
}

func (t *Toolset) GetTraceByIdHandler(params ToolParams) *resultutil.Result {
	client, err := t.getTempoClient(params)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	args := params.arguments

	traceid := tools.GetString(args, "traceid", "")
	if traceid == "" {
		return resultutil.NewErrorResult(fmt.Errorf("traceid parameter must not be empty"))
	}

	start, err := parseDate(tools.GetString(args, "start", ""))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("invalid start time: %v", err))
	}

	end, err := parseDate(tools.GetString(args, "end", ""))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("invalid end time: %v", err))
	}

	opts := tempoclient.QueryV2Options{
		Start: start,
		End:   end,
	}

	trace, err := client.QueryV2(params.context, traceid, opts)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	return resultutil.NewJSONSuccessResult(trace)
}
