package prometheus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
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

// makeLLMFriendlyError converts Prometheus errors into more descriptive, LLM-friendly messages
func makeLLMFriendlyError(err error, query string) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	lowerMsg := strings.ToLower(errMsg)

	// Check for common error patterns and provide helpful context
	switch {
	case strings.Contains(lowerMsg, "parse error") || strings.Contains(lowerMsg, "bad_data"):
		return fmt.Errorf("the PromQL query '%s' has a syntax error. Error details: %w. "+
			"Please check the query syntax and ensure all metric names, labels, and functions are correctly formatted",
			query, err)

	case strings.Contains(lowerMsg, "unknown function"):
		return fmt.Errorf("the PromQL query '%s' uses an unknown function. Error details: %w. "+
			"Please verify that the function name is correct and supported by Prometheus",
			query, err)

	case strings.Contains(lowerMsg, "timeout") || strings.Contains(lowerMsg, "deadline exceeded"):
		return fmt.Errorf("the query '%s' took too long to execute and timed out. Error details: %w. "+
			"This might happen if the query is too complex, the time range is too large, or the Prometheus server is under heavy load. "+
			"Try reducing the time range, increasing the step size, or simplifying the query",
			query, err)

	case strings.Contains(lowerMsg, "no such host") || strings.Contains(lowerMsg, "connection refused"):
		return fmt.Errorf("cannot connect to the Prometheus server. Error details: %w. "+
			"Please verify that the Prometheus server is running and accessible", err)

	default:
		// Return error with query context for better debugging
		return fmt.Errorf("query '%s' failed: %w", query, err)
	}
}

// checkEmptyResult provides helpful context when a query returns no data
func checkEmptyResult(result any, query string) string {
	var isEmpty bool

	switch v := result.(type) {
	case model.Matrix:
		isEmpty = len(v) == 0
	case model.Vector:
		isEmpty = len(v) == 0
	case *model.Scalar:
		isEmpty = v == nil
	case *model.String:
		isEmpty = v == nil
	default:
		return ""
	}

	if isEmpty {
		return fmt.Sprintf("The query '%s' executed successfully but returned no data. "+
			"This could mean: (1) the metric does not exist, (2) the metric exists but has no data for the specified time range, "+
			"(3) the label filters are too restrictive, or (4) there's no data matching your query conditions. "+
			"You can use the 'list_metrics' tool to see all available metrics, or try adjusting the time range or label filters.",
			query)
	}

	return ""
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
		return nil, makeLLMFriendlyError(err, query)
	}

	response := map[string]any{
		"resultType": result.Type().String(),
		"result":     result,
	}

	// Add warnings from Prometheus
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	// Check for empty results and add guidance separately
	if emptyWarning := checkEmptyResult(result, query); emptyWarning != "" {
		response["emptyResultGuidance"] = emptyWarning
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
		return nil, makeLLMFriendlyError(err, query)
	}

	response := map[string]any{
		"resultType": result.Type().String(),
		"result":     result,
	}

	// Add warnings from Prometheus
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	// Check for empty results and add guidance separately
	if emptyWarning := checkEmptyResult(result, query); emptyWarning != "" {
		response["emptyResultGuidance"] = emptyWarning
	}

	return response, nil
}
