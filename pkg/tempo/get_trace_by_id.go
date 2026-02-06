package tempo

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func GetTraceByIdTool() mcp.Tool {
	return mcp.NewTool(
		"tempo_get_trace_by_id",
		mcp.WithDescription("Get a trace by trace ID"),
		mcp.WithReadOnlyHintAnnotation(true),
		withTempoInstanceParams(),
		mcp.WithString("traceid",
			mcp.Required(),
			mcp.Description("TraceID of the trace"),
		),
		mcp.WithString("start",
			mcp.Description("Start time in RFC 3339 format"),
		),
		mcp.WithString("end",
			mcp.Description("End time in RFC 3339 format"),
		),
	)
}

func (t *TempoToolset) GetTraceByIdHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.getTempoClient(ctx, request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	traceid, err := request.RequireString("traceid")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if traceid == "" {
		return mcp.NewToolResultError("traceid parameter must not be empty"), nil
	}

	start, err := parseDate(request.GetString("start", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid start time: %v", err)), nil
	}

	end, err := parseDate(request.GetString("end", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid end time: %v", err)), nil
	}

	opts := QueryV2Options{
		Start: start,
		End:   end,
	}

	trace, err := client.QueryV2(ctx, traceid, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(trace), nil
}
