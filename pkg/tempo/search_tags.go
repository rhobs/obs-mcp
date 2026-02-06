package tempo

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func SearchTagsTool() mcp.Tool {
	return mcp.NewTool(
		"tempo_search_tags",
		mcp.WithDescription("Search for tag names in Tempo"),
		mcp.WithReadOnlyHintAnnotation(true),
		withTempoInstanceParams(),
		mcp.WithString("scope",
			mcp.Description("Scope to filter tags: resource, span, intrinsic, event, link, or instrumentation"),
		),
		mcp.WithString("query",
			mcp.Description("TraceQL query for filtering tag names"),
		),
		mcp.WithString("start",
			mcp.Description("Start time in RFC 3339 format"),
		),
		mcp.WithString("end",
			mcp.Description("End time in RFC 3339 format"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of tag names per scope"),
		),
		mcp.WithNumber("maxStaleValues",
			mcp.Description("Search termination threshold for stale values"),
		),
	)
}

func (t *TempoToolset) SearchTagsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := t.getTempoClient(ctx, request)
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

	opts := SearchTagsV2Options{
		Scope:          request.GetString("scope", ""),
		Query:          request.GetString("query", ""),
		Start:          start,
		End:            end,
		Limit:          request.GetInt("limit", 0),
		MaxStaleValues: request.GetInt("maxStaleValues", 0),
	}

	result, err := client.SearchTagsV2(ctx, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}
