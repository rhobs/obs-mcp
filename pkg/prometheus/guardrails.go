package prometheus

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

// Guardrail name constants for use with ParseGuardrails
const (
	GuardrailDisallowExplicitNameLabel = "disallow-explicit-name-label"
	GuardrailRequireLabelMatcher       = "require-label-matcher"
	GuardrailDisallowBlanketRegex      = "disallow-blanket-regex"
	GuardrailMaxMetricCardinality      = "max-metric-cardinality"
	GuardrailMaxLabelCardinality       = "max-label-cardinality"
)

// Guardrails provides safety checks for PromQL queries based on configurable rules.
type Guardrails struct {
	// DisallowExplicitNameLabel prevents queries using explicit {__name__="..."} syntax
	DisallowExplicitNameLabel bool
	// RequireLabelMatcher ensures all vector selectors have at least one non-name label matcher
	RequireLabelMatcher bool
	// DisallowBlanketRegex prevents expensive regex patterns like .* or .+ on any label
	DisallowBlanketRegex bool
	// MaxMetricCardinality sets the maximum allowed series count per metric (0 = disabled)
	MaxMetricCardinality uint64
	// MaxLabelCardinality sets the maximum allowed label value count for blanket regex
	// (0 = always disallow regex matcher provided DisallowBlanketRegex is true)
	MaxLabelCardinality uint64
}

// DefaultGuardrails returns a Guardrails instance with all safety checks enabled.
func DefaultGuardrails() *Guardrails {
	return &Guardrails{
		DisallowExplicitNameLabel: true,
		RequireLabelMatcher:       true,
		DisallowBlanketRegex:      true,
		MaxMetricCardinality:      20000,
		MaxLabelCardinality:       500,
	}
}

func ParseGuardrails(value string) (*Guardrails, error) {
	value = strings.TrimSpace(value)

	switch strings.ToLower(value) {
	case "none":
		return nil, nil
	case "all", "":
		return DefaultGuardrails(), nil
	}

	g := &Guardrails{}
	names := strings.SplitSeq(value, ",")
	for name := range names {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			continue
		}

		switch name {
		case GuardrailDisallowExplicitNameLabel:
			g.DisallowExplicitNameLabel = true
		case GuardrailRequireLabelMatcher:
			g.RequireLabelMatcher = true
		case GuardrailDisallowBlanketRegex:
			g.DisallowBlanketRegex = true
		default:
			return nil, fmt.Errorf("unknown guardrail: %q (valid options: %s, %s, %s)",
				name, GuardrailDisallowExplicitNameLabel, GuardrailRequireLabelMatcher,
				GuardrailDisallowBlanketRegex)
		}
	}

	return g, nil
}

// IsSafeQuery analyzes a PromQL query string and returns false if it's
// deemed unsafe or too expensive based on the configured rules.
//
// Non-optional checks (always enforced):
//   - Metric existence validation: Ensures queried metrics exist in TSDB
//
// Optional checks (based on Guardrails configuration):
//   - DisallowExplicitNameLabel: Prevents queries using explicit {__name__="..."} syntax
//   - RequireLabelMatcher: Ensures all vector selectors have at least one non-name label matcher
//   - MaxMetricCardinality: Checks TSDB metric cardinality limits
//   - DisallowBlanketRegex: Prevents expensive regex patterns like .* or .+
//   - MaxLabelCardinality: Checks TSDB label cardinality for blanket regex
//
// Returns (false, error) if the query is invalid or violates any rule.
// The error message explains which rule was violated.
// Returns (true, nil) if the query is valid and passes all rules.
//
//nolint:gocyclo // complex validation logic, refactoring would reduce readability
func (g *Guardrails) IsSafeQuery(ctx context.Context, query string, client v1.API) (bool, error) {
	// Always require client and context for TSDB checks (including metric existence validation)
	if client == nil || ctx == nil {
		return false, fmt.Errorf("cannot validate query safety: Prometheus client or context is not available. " +
			"This is an internal configuration issue - please ensure the Prometheus client is properly initialized")
	}

	// NON-OPTIONAL CHECK: Parse and validate query syntax
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return false, fmt.Errorf("the PromQL query '%s' has a syntax error. Error details: %w. "+
			"Please check the query syntax and ensure all metric names, labels, and functions are correct, and properly formatted",
			query, err)
	}

	// Early check: validate that queried metrics exist in TSDB
	// Also fetch TSDB stats once for all subsequent cardinality checks
	metricNames, err := ExtractMetricNames(query)
	if err != nil {
		return false, fmt.Errorf("failed to analyze the query '%s' to extract metric names. Error details: %w. "+
			"This might indicate an issue with the query structure",
			query, err)
	}

	var seriesCountByMetric map[string]uint64
	var labelValueCountByLabel map[string]uint64

	// Query TSDB once and reuse for all checks
	tsdbResult, err := client.TSDB(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve metrics information from Prometheus TSDB. Error details: %w. "+
			"Please verify that the Prometheus server is accessible and responding correctly",
			err)
	}

	seriesCountByMetric = make(map[string]uint64)
	for _, stat := range tsdbResult.SeriesCountByMetricName {
		seriesCountByMetric[stat.Name] = stat.Value
	}

	labelValueCountByLabel = make(map[string]uint64)
	for _, stat := range tsdbResult.LabelValueCountByLabelName {
		labelValueCountByLabel[stat.Name] = stat.Value
	}

	// NON-OPTIONAL CHECK: Validate that all queried metrics exist in TSDB
	// This check always runs regardless of guardrail configuration
	if len(metricNames) > 0 {
		for _, metricName := range metricNames {
			count, exists := seriesCountByMetric[metricName]
			if !exists || count == 0 {
				return false, fmt.Errorf("the metric '%s' does not exist in Prometheus. "+
					"You can use the 'list_metrics' tool to see all available metrics, or check if the metric name is spelled correctly",
					metricName)
			}
		}
	}

	// NON-OPTIONAL CHECK: Validate that all queried labels exist in TSDB
	labelNames, err := ExtractLabelNames(query)
	if err != nil {
		return false, fmt.Errorf("failed to analyze the query '%s' to extract label names. Error details: %w. "+
			"This might indicate an issue with the query structure",
			query, err)
	}

	if len(labelNames) > 0 {
		for _, labelName := range labelNames {
			if _, exists := labelValueCountByLabel[labelName]; !exists {
				return false, fmt.Errorf("the label '%s' used in query '%s' does not exist in Prometheus. "+
					"Please check if the label name is spelled correctly. "+
					"Common labels include 'job', 'instance', etc. You may need to verify which labels are available for your metrics",
					labelName, query)
			}
		}
	}

	var unsafeReason error

	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		vs, ok := node.(*parser.VectorSelector)
		if !ok {
			return nil
		}

		// Check for explicit __name__ label query
		if g.DisallowExplicitNameLabel && vs.Name == "" {
			for _, m := range vs.LabelMatchers {
				if m.Name == labels.MetricName {
					unsafeReason = fmt.Errorf("the query '%s' uses explicit {__name__=\"...\"} syntax, which is not allowed. "+
						"Please use the direct metric name syntax instead (e.g., use 'metric_name{label=\"value\"}' instead of '{__name__=\"metric_name\",label=\"value\"}')",
						query)
					return unsafeReason
				}
			}
		}

		// All vector selectors must have at least one non-name label matcher
		if g.RequireLabelMatcher {
			hasNonNameMatcher := false
			for _, m := range vs.LabelMatchers {
				if m.Name != labels.MetricName {
					hasNonNameMatcher = true
					break
				}
			}
			if !hasNonNameMatcher {
				unsafeReason = fmt.Errorf("the query for metric '%s' does not have any label filters, which is required for safety. "+
					"Queries without label matchers can be very expensive as they return all series for a metric. "+
					"Please add at least one label filter (e.g., '%s{job=\"...\"}' or '%s{instance=\"...\"}')",
					vs.Name, vs.Name, vs.Name)
				return unsafeReason
			}
		}

		return nil
	})

	if unsafeReason != nil {
		return false, unsafeReason
	}

	// Check metric cardinality (reuse TSDB data fetched earlier)
	if g.MaxMetricCardinality > 0 && len(metricNames) > 0 {
		for _, metricName := range metricNames {
			if count, exists := seriesCountByMetric[metricName]; exists {
				if count > g.MaxMetricCardinality {
					return false, fmt.Errorf("the metric '%s' has %d time series, which exceeds the maximum allowed limit of %d. "+
						"This metric has too many unique label combinations, making queries expensive. "+
						"Please add more specific label filters to reduce the number of series returned, or use aggregation functions to reduce cardinality",
						metricName, count, g.MaxMetricCardinality)
				}
			}
		}
	}

	// Check blanket regex patterns (reuse TSDB data fetched earlier)
	if g.DisallowBlanketRegex {
		blanketRegexLabels, err := ExtractBlanketRegexLabels(query)
		if err != nil {
			return false, fmt.Errorf("failed to analyze the query '%s' for regex patterns. Error details: %w. "+
				"This might indicate an issue with the query structure",
				query, err)
		}

		if len(blanketRegexLabels) > 0 {
			// If MaxLabelCardinality is 0, always disallow blanket regex
			if g.MaxLabelCardinality == 0 {
				return false, fmt.Errorf("the query '%s' uses a blanket regex pattern (=~ \".*\" or =~ \".+\") on label '%s', which is not allowed. "+
					"Blanket regex patterns match all values and can be extremely expensive. "+
					"Please use more specific label matchers or exact value matches instead",
					query, blanketRegexLabels[0])
			}

			// Check label cardinality for blanket regex using already-fetched TSDB data
			for _, labelName := range blanketRegexLabels {
				if count, exists := labelValueCountByLabel[labelName]; exists {
					if count > g.MaxLabelCardinality {
						return false, fmt.Errorf("the query '%s' uses a blanket regex pattern on label '%s', which has %d unique values. "+
							"This exceeds the maximum allowed limit of %d for regex patterns. "+
							"Blanket regex on high-cardinality labels is very expensive. "+
							"Please use specific label values or more restrictive regex patterns instead",
							query, labelName, count, g.MaxLabelCardinality)
					}
				}
			}
		}
	}

	return true, nil
}

// MakeLLMFriendlyError converts Prometheus execution errors into more descriptive, LLM-friendly messages.
// Note: Parse errors are handled in IsSafeQuery where the query is parsed.
func MakeLLMFriendlyError(err error, query string) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	lowerMsg := strings.ToLower(errMsg)

	// Check for common error patterns and provide helpful context
	switch {
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

func ExtractMetricNames(query string) ([]string, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	metricNames := make(map[string]bool)
	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			if vs.Name != "" {
				metricNames[vs.Name] = true
			}
			// Also check for __name__ label matchers
			for _, m := range vs.LabelMatchers {
				if m.Name == labels.MetricName && m.Type == labels.MatchEqual {
					metricNames[m.Value] = true
				}
			}
		}
		return nil
	})

	result := make([]string, 0, len(metricNames))
	for name := range metricNames {
		result = append(result, name)
	}
	return result, nil
}

// ExtractLabelNames extracts all label names used in the query (excluding __name__).
func ExtractLabelNames(query string) ([]string, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	labelNames := make(map[string]bool)
	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			for _, m := range vs.LabelMatchers {
				// Skip __name__ as it's the metric name, not a label
				if m.Name != labels.MetricName {
					labelNames[m.Name] = true
				}
			}
		}
		return nil
	})

	result := make([]string, 0, len(labelNames))
	for name := range labelNames {
		result = append(result, name)
	}
	return result, nil
}

// ExtractBlanketRegexLabels extracts label names that use blanket regex patterns (.* or .+).
func ExtractBlanketRegexLabels(query string) ([]string, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	labelNames := make(map[string]bool)
	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			for _, m := range vs.LabelMatchers {
				isRegex := m.Type == labels.MatchRegexp || m.Type == labels.MatchNotRegexp
				if isRegex && (m.Value == ".*" || m.Value == ".+") {
					labelNames[m.Name] = true
				}
			}
		}
		return nil
	})

	result := make([]string, 0, len(labelNames))
	for name := range labelNames {
		result = append(result, name)
	}
	return result, nil
}
