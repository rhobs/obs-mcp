package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type PrometheusClient struct {
	client v1.API
}

func NewPrometheusClient(apiConfig api.Config) (*PrometheusClient, error) {
	client, err := api.NewClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %w", err)
	}

	v1api := v1.NewAPI(client)
	return &PrometheusClient{client: v1api}, nil
}

func (p *PrometheusClient) ListMetrics(ctx context.Context) ([]string, error) {
	labelValues, _, err := p.client.LabelValues(ctx, "__name__", []string{}, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		return nil, fmt.Errorf("error fetching metric names: %w", err)
	}

	metrics := make([]string, len(labelValues))
	for i, value := range labelValues {
		metrics[i] = string(value)
	}
	return metrics, nil
}

func (p *PrometheusClient) ExecuteRangeQuery(ctx context.Context, query string, start, end time.Time, step time.Duration, useGuardrails bool) (map[string]interface{}, error) {
	if useGuardrails {
		isSafe, err := isSafeQuery(query)
		if err != nil {
			return nil, fmt.Errorf("failed to parse query: %w", err)
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

	result, warnings, err := p.client.QueryRange(ctx, query, r, v1.WithTimeout(30*time.Second))
	if err != nil {
		return nil, fmt.Errorf("error executing range query: %w", err)
	}

	response := map[string]interface{}{
		"resultType": result.Type().String(),
		"result":     result,
	}

	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	return response, nil
}

func (p *PrometheusClient) ExecuteInstantQuery(ctx context.Context, query string, time time.Time, useGuardrails bool) (map[string]interface{}, error) {
	if useGuardrails {
		isSafe, err := isSafeQuery(query)
		if err != nil {
			return nil, fmt.Errorf("failed to parse query: %w", err)
		}
		if !isSafe {
			return nil, fmt.Errorf("query is not safe")
		}
	}

	result, warnings, err := p.client.Query(ctx, query, time)
	if err != nil {
		return nil, fmt.Errorf("error executing instant query: %w", err)
	}

	response := map[string]interface{}{
		"resultType": result.Type().String(),
		"result":     result,
	}

	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	return response, nil
}
