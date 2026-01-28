package tempo

import (
	"context"
	"fmt"
)

// SearchTraces runs a TraceQL search against Tempo.
func (t *TempoToolset) SearchTraces(ctx context.Context, namespace, name, tenant, query string, limit int, startRFC, endRFC string, spss int) (string, error) {
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

	opts := SearchOptions{
		Query: query,
		Limit: limit,
		Start: start,
		End:   end,
		Spss:  spss,
	}

	return client.Search(ctx, opts)
}
