package tempo

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func SearchTracesTool() mcp.Tool {
	return mcp.NewTool(
		"tempo_search_traces",
		mcp.WithDescription("Search for traces in Tempo"),
		mcp.WithReadOnlyHintAnnotation(true),
		withTempoInstanceParams(),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query in the TraceQL query language"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum search results"),
		),
		mcp.WithString("start",
			mcp.Description("Start time in RFC 3339 format"),
		),
		mcp.WithString("end",
			mcp.Description("End time in RFC 3339 format"),
		),
		mcp.WithNumber("spss",
			mcp.Description("Spans per span-set limit"),
		),
	)
}

func (t *TempoToolset) SearchTracesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.getTempoClient(ctx, request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	start, err := parseDate(request.GetString("start", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid start time: %v", err)), nil
	}

	end, err := parseDate(request.GetString("end", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid end time: %v", err)), nil
	}

	opts := SearchOptions{
		Query: query,
		Limit: request.GetInt("limit", 0),
		Start: start,
		End:   end,
		Spss:  request.GetInt("spss", 0),
	}

	trace, err := client.Search(ctx, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(trace), nil
}
