package tempo

import (
	"context"
	"fmt"
)

// SearchTags lists tag names from Tempo (v2 API).
func (t *TempoToolset) SearchTags(ctx context.Context, namespace, name, tenant, scope, query, startRFC, endRFC string, limit, maxStaleValues int) (string, error) {
	client, err := t.GetTempoClient(ctx, namespace, name, tenant)
	if err != nil {
		return "", err
	}

	start, err := parseDate(startRFC)
	if err != nil {
		return "", fmt.Errorf("invalid start time: %w", err)
	}

	end, err := parseDate(endRFC)
	if err != nil {
		return "", fmt.Errorf("invalid end time: %w", err)
	}

	opts := SearchTagsV2Options{
		Scope:          scope,
		Query:          query,
		Start:          start,
		End:            end,
		Limit:          limit,
		MaxStaleValues: maxStaleValues,
	}

	return client.SearchTagsV2(ctx, opts)
}
