package tempo

import (
	"context"
	"fmt"
)

// GetTraceByID loads a trace from Tempo (query API v2).
func (t *TempoToolset) GetTraceByID(ctx context.Context, namespace, name, tenant, traceID, startRFC, endRFC string) (string, error) {
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

	opts := QueryV2Options{
		Start: start,
		End:   end,
	}

	return client.QueryV2(ctx, traceID, opts)
}
