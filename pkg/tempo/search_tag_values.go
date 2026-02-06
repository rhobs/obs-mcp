package tempo

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func SearchTagValuesTool() mcp.Tool {
	return mcp.NewTool(
		"tempo_search_tag_values",
		mcp.WithDescription("Search for tag values in Tempo"),
		mcp.WithReadOnlyHintAnnotation(true),
		withTempoInstanceParams(),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.Description("The tag name to get values for"),
		),
		mcp.WithString("query",
			mcp.Description("TraceQL query for filtering tag values"),
		),
		mcp.WithString("start",
			mcp.Description("Start time in RFC 3339 format"),
		),
		mcp.WithString("end",
			mcp.Description("End time in RFC 3339 format"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of tag values to return"),
		),
		mcp.WithNumber("maxStaleValues",
			mcp.Description("Search termination threshold for stale values"),
		),
	)
}

func (t *TempoToolset) SearchTagValuesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.getTempoClient(ctx, request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tag, err := request.RequireString("tag")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if tag == "" {
		return mcp.NewToolResultError("tag parameter must not be empty"), nil
	}

	start, err := parseDate(request.GetString("start", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid start time: %v", err)), nil
	}

	end, err := parseDate(request.GetString("end", ""))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid end time: %v", err)), nil
	}

	opts := SearchTagValuesV2Options{
		Query:          request.GetString("query", ""),
		Start:          start,
		End:            end,
		Limit:          request.GetInt("limit", 0),
		MaxStaleValues: request.GetInt("maxStaleValues", 0),
	}

	result, err := client.SearchTagValuesV2(ctx, tag, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}
