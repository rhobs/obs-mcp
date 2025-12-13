package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const (
	// ListMetricsTimeRange is the time range used when listing metrics
	ListMetricsTimeRange = 1 * time.Hour
	// DefaultQueryTimeout is the default timeout for Prometheus queries
	DefaultQueryTimeout = 30 * time.Second
)

// Loader defines the interface for querying Prometheus
type Loader interface {
	ListMetrics(ctx context.Context) ([]string, error)
	ExecuteRangeQuery(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]any, error)
	ExecuteInstantQuery(ctx context.Context, query string, time time.Time) (map[string]any, error)
}

// PrometheusClient implements PromClient
type RealLoader struct {
	client     v1.API
	guardrails *Guardrails
}

// Ensure PrometheusClient implements PromClient at compile time
var _ Loader = (*RealLoader)(nil)

func NewPrometheusClient(apiConfig api.Config) (*RealLoader, error) {
	client, err := api.NewClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %w", err)
	}

	v1api := v1.NewAPI(client)
	return &RealLoader{
		client:     v1api,
		guardrails: DefaultGuardrails(),
	}, nil
}

// WithGuardrails sets a custom Guardrails configuration for the client.
func (p *RealLoader) WithGuardrails(g *Guardrails) *RealLoader {
	p.guardrails = g
	return p
}

func (p *RealLoader) ListMetrics(ctx context.Context) ([]string, error) {
	labelValues, _, err := p.client.LabelValues(ctx, "__name__", []string{}, time.Now().Add(-ListMetricsTimeRange), time.Now())
	if err != nil {
		return nil, fmt.Errorf("error fetching metric names: %w", err)
	}

	metrics := make([]string, len(labelValues))
	for i, value := range labelValues {
		metrics[i] = string(value)
	}
	return metrics, nil
}

func (p *RealLoader) ExecuteRangeQuery(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]any, error) {
	if p.guardrails != nil {
		isSafe, err := p.guardrails.IsSafeQuery(ctx, query, p.client)
		if err != nil {
			return nil, fmt.Errorf("query validation failed: %w", err)
		}
		if !isSafe {
			return nil, fmt.Errorf("query is not safe")
		}
	}

	r := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}

	result, warnings, err := p.client.QueryRange(ctx, query, r, v1.WithTimeout(DefaultQueryTimeout))
	if err != nil {
		return nil, fmt.Errorf("error executing range query: %w", err)
	}

	response := map[string]any{
		"resultType": result.Type().String(),
		"result":     result,
	}

	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	return response, nil
}

func (p *RealLoader) ExecuteInstantQuery(ctx context.Context, query string, ts time.Time) (map[string]any, error) {
	if p.guardrails != nil {
		isSafe, err := p.guardrails.IsSafeQuery(ctx, query, p.client)
		if err != nil {
			return nil, fmt.Errorf("query validation failed: %w", err)
		}
		if !isSafe {
			return nil, fmt.Errorf("query is not safe")
		}
	}

	result, warnings, err := p.client.Query(ctx, query, ts)
	if err != nil {
		return nil, fmt.Errorf("error executing instant query: %w", err)
	}

	response := map[string]any{
		"resultType": result.Type().String(),
		"result":     result,
	}

	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	return response, nil
}
