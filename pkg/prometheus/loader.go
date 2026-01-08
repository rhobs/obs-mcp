package prometheus

import (
	"context"
	"fmt"
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
// If nil is passed, creates an empty Guardrails (metric existence check always runs).
func (p *RealLoader) WithGuardrails(g *Guardrails) *RealLoader {
	if g == nil {
		// Even with no optional guardrails, ensure we have an instance for existence checks
		p.guardrails = &Guardrails{}
	} else {
		p.guardrails = g
	}
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

// checkEmptyResult provides helpful context when a query returns no data after validation passes
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
		return fmt.Sprintf("The query '%s' returned no data. "+
			"The metric and labels you specified exist in Prometheus, but no time series match your specific label filter combination. "+
			"This could mean: (1) the label values you're filtering for don't exist, (2) the combination of label filters is too restrictive, "+
			"or (3) there's no data for the specified time range with these exact label values. "+
			"Try using less restrictive label filters, checking label values, or adjusting the time range.",
			query)
	}

	return ""
}

func (p *RealLoader) ExecuteRangeQuery(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]any, error) {
	// Always validate query (including non-optional metric existence check)
	isSafe, err := p.guardrails.IsSafeQuery(ctx, query, p.client)
	if err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}
	if !isSafe {
		return nil, fmt.Errorf("query is not safe")
	}

	r := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}

	result, warnings, err := p.client.QueryRange(ctx, query, r, v1.WithTimeout(DefaultQueryTimeout))
	if err != nil {
		return nil, MakeLLMFriendlyError(err, query)
	}

	response := map[string]any{
		"resultType": result.Type().String(),
		"result":     result,
	}

	// Add warnings from Prometheus
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	// Check for empty results and add guidance
	if emptyWarning := checkEmptyResult(result, query); emptyWarning != "" {
		response["emptyResultGuidance"] = emptyWarning
	}

	return response, nil
}

func (p *RealLoader) ExecuteInstantQuery(ctx context.Context, query string, ts time.Time) (map[string]any, error) {
	// Always validate query (including non-optional metric existence check)
	isSafe, err := p.guardrails.IsSafeQuery(ctx, query, p.client)
	if err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}
	if !isSafe {
		return nil, fmt.Errorf("query is not safe")
	}

	result, warnings, err := p.client.Query(ctx, query, ts)
	if err != nil {
		return nil, MakeLLMFriendlyError(err, query)
	}

	response := map[string]any{
		"resultType": result.Type().String(),
		"result":     result,
	}

	// Add warnings from Prometheus
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	// Check for empty results and add guidance
	if emptyWarning := checkEmptyResult(result, query); emptyWarning != "" {
		response["emptyResultGuidance"] = emptyWarning
	}

	return response, nil
}
