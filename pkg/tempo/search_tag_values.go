package tempo

import (
	"context"
	"fmt"
)

// SearchTagValues lists values for a tag from Tempo (v2 API).
func (t *TempoToolset) SearchTagValues(ctx context.Context, namespace, name, tenant, tag, query, startRFC, endRFC string, limit, maxStaleValues int) (string, error) {
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

	opts := SearchTagValuesV2Options{
		Query:          query,
		Start:          start,
		End:            end,
		Limit:          limit,
		MaxStaleValues: maxStaleValues,
	}

	return client.SearchTagValuesV2(ctx, tag, opts)
}
