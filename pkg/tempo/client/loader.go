package client

import (
	"context"
	"fmt"
	"net/http"
)

// Loader defines the interface for querying Tempo
type Loader interface {
	QueryV2(ctx context.Context, traceID string, opts QueryV2Options) (string, error)
	Search(ctx context.Context, opts SearchOptions) (string, error)
	SearchTagsV2(ctx context.Context, opts SearchTagsV2Options) (string, error)
	SearchTagValuesV2(ctx context.Context, tag string, opts SearchTagValuesV2Options) (string, error)
}

// RealLoader implements Loader using the Tempo HTTP API.
type RealLoader struct {
	client *TempoClient
}

var _ Loader = (*RealLoader)(nil)

const MAX_TRACE_LIMIT = 1000

func NewTempoLoader(httpClient *http.Client, url string) Loader {
	client := NewTempoClient(httpClient, url)

	return &RealLoader{
		client: client,
	}
}

func (r *RealLoader) QueryV2(ctx context.Context, traceID string, opts QueryV2Options) (string, error) {
	return r.client.QueryV2(ctx, traceID, opts)
}

func (r *RealLoader) Search(ctx context.Context, opts SearchOptions) (string, error) {
	if opts.Limit > MAX_TRACE_LIMIT {
		return "", fmt.Errorf("Requested search results limit %d is greater than max limit %d. Please decrease the search results limit.", opts.Limit, MAX_TRACE_LIMIT)
	}
	return r.client.Search(ctx, opts)
}

func (r *RealLoader) SearchTagsV2(ctx context.Context, opts SearchTagsV2Options) (string, error) {
	return r.client.SearchTagsV2(ctx, opts)
}

func (r *RealLoader) SearchTagValuesV2(ctx context.Context, tag string, opts SearchTagValuesV2Options) (string, error) {
	return r.client.SearchTagValuesV2(ctx, tag, opts)
}
