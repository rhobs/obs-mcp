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

var (
	// guard rails with their default values
	disallowExplicitNameLabel = &Guardrail{Name: GuardrailDisallowExplicitNameLabel, RequireTSDBEndpoint: false, Value: true}
	requireLabelMatcher       = &Guardrail{Name: GuardrailRequireLabelMatcher, RequireTSDBEndpoint: false, Value: true}
	disallowBlanketRegex      = &Guardrail{Name: GuardrailDisallowBlanketRegex, RequireTSDBEndpoint: true, Value: true}
	maxMetricCardinality      = &Guardrail{Name: GuardrailMaxMetricCardinality, RequireTSDBEndpoint: true, Value: 20000}
	maxLabelCardinality       = &Guardrail{Name: GuardrailMaxLabelCardinality, RequireTSDBEndpoint: true, Value: 500}
)

// GuardrailViolation is returned when a query violates a specific guardrail rule.
// It carries the guardrail name for structured logging.
type GuardrailViolation struct {
	Guardrail string
	Message   string
}

func (e *GuardrailViolation) Error() string {
	return e.Message
}

type Guardrail struct {
	Name                string
	RequireTSDBEndpoint bool
	Value               any
}

type Guardrails map[string]*Guardrail

func (g Guardrails) SetMaxLabelCardinality(v uint64) {
	g[GuardrailMaxLabelCardinality].Value = v
}

func (g Guardrails) SetMaxMetricCardinality(v uint64) {
	g[GuardrailMaxMetricCardinality].Value = v
}

// Helper methods for accessing guardrail values
func (g Guardrails) IsEnabled(name string) bool {
	if guardrail, ok := g[name]; ok {
		if boolVal, ok := guardrail.Value.(bool); ok {
			return boolVal
		}
	}
	return false
}

func (g Guardrails) GetUint64Value(name string) (uint64, bool) {
	if guardrail, ok := g[name]; ok {
		if val, ok := guardrail.Value.(uint64); ok {
			return val, true
		}
	}
	return 0, false
}

func (g Guardrails) RequiresTSDB(name string) bool {
	if guardrail, ok := g[name]; ok {
		return guardrail.RequireTSDBEndpoint
	}
	return false
}

func (g Guardrails) RequiresAnyTSDB() bool {
	return g.RequiresTSDB(GuardrailDisallowBlanketRegex) ||
		g.RequiresTSDB(GuardrailMaxMetricCardinality) ||
		g.RequiresTSDB(GuardrailMaxLabelCardinality)
}

func DefaultGuardrails() Guardrails {
	return Guardrails{
		GuardrailDisallowExplicitNameLabel: disallowExplicitNameLabel,
		GuardrailRequireLabelMatcher:       requireLabelMatcher,
		GuardrailDisallowBlanketRegex:      disallowBlanketRegex,
		GuardrailMaxMetricCardinality:      maxMetricCardinality,
		GuardrailMaxLabelCardinality:       maxLabelCardinality,
	}
}

func ParseGuardrails(value string) (Guardrails, error) {
	value = strings.TrimSpace(value)

	switch strings.ToLower(value) {
	case "none":
		return nil, nil
	case "all", "":
		return DefaultGuardrails(), nil
	}

	g := Guardrails{}
	names := strings.SplitSeq(value, ",")
	for name := range names {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			continue
		}

		switch name {
		case GuardrailDisallowExplicitNameLabel:
			g[GuardrailDisallowExplicitNameLabel] = disallowExplicitNameLabel
		case GuardrailRequireLabelMatcher:
			g[GuardrailRequireLabelMatcher] = requireLabelMatcher
		case GuardrailDisallowBlanketRegex:
			g[GuardrailDisallowBlanketRegex] = disallowBlanketRegex
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
// If client is provided and MaxMetricCardinality is set, it checks TSDB metric cardinality.
// If client is provided and MaxLabelCardinality is set, it checks TSDB label cardinality for blanket regex.
//
// Returns (false, error) if the query is invalid or violates a guardrail rule.
// The error message explains which rule was violated.
// Returns (true, nil) if the query is valid and passes all rules.
//
//nolint:gocyclo // complex validation logic, refactoring would reduce readability
func (g Guardrails) IsSafeQuery(ctx context.Context, query string, client v1.API) (bool, error) {
	disallowExplicitNameLabel := g.IsEnabled(GuardrailDisallowExplicitNameLabel)
	requireLabelMatcher := g.IsEnabled(GuardrailRequireLabelMatcher)
	disallowBlanketRegex := g.IsEnabled(GuardrailDisallowBlanketRegex)
	maxMetricCardinality, hasMaxMetricCard := g.GetUint64Value(GuardrailMaxMetricCardinality)
	maxLabelCardinality, hasMaxLabelCard := g.GetUint64Value(GuardrailMaxLabelCardinality)
	maxLabelCardinalitySet := hasMaxLabelCard && maxLabelCardinality > 0
	maxMetricCardinalitySet := (hasMaxMetricCard && maxMetricCardinality > 0)

	requiresTSDB := g.RequiresAnyTSDB()
	needsTSDB := requiresTSDB && (maxLabelCardinalitySet && disallowBlanketRegex) || maxMetricCardinalitySet

	if needsTSDB && (client == nil || ctx == nil) {
		return false, fmt.Errorf("cannot verify cardinality without TSDB client")
	}

	expr, err := parser.ParseExpr(query)
	if err != nil {
		return false, fmt.Errorf("failed to parse query: %w", err)
	}

	var unsafeReason error

	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		vs, ok := node.(*parser.VectorSelector)
		if !ok {
			return nil
		}

		// Check for explicit __name__ label query
		if disallowExplicitNameLabel && vs.Name == "" {
			for _, m := range vs.LabelMatchers {
				if m.Name == labels.MetricName {
					unsafeReason = &GuardrailViolation{
						Guardrail: GuardrailDisallowExplicitNameLabel,
						Message:   "query uses explicit __name__ label matcher, which is disallowed",
					}
					return unsafeReason
				}
			}
		}

		// All vector selectors must have at least one non-name label matcher
		if requireLabelMatcher {
			hasNonNameMatcher := false
			for _, m := range vs.LabelMatchers {
				if m.Name != labels.MetricName {
					hasNonNameMatcher = true
					break
				}
			}
			if !hasNonNameMatcher {
				unsafeReason = &GuardrailViolation{
					Guardrail: GuardrailRequireLabelMatcher,
					Message:   fmt.Sprintf("query for metric %q does not have any label matchers, which is required", vs.Name),
				}
				return unsafeReason
			}
		}

		return nil
	})

	if unsafeReason != nil {
		return false, unsafeReason
	}

	// Check metric cardinality
	if maxMetricCardinalitySet && requiresTSDB {
		metricNames, err := ExtractMetricNames(query)
		if err != nil {
			return false, fmt.Errorf("failed to extract metric names: %w", err)
		}

		if len(metricNames) > 0 {
			tsdbResult, err := client.TSDB(ctx)
			if err != nil {
				return false, fmt.Errorf(
					"cannot enforce max-metric-cardinality guardrail: TSDB stats endpoint is unavailable on this backend "+
						"(Thanos Querier < v0.40.0 does not implement /api/v1/status/tsdb); "+
						"disable this guardrail with --guardrails require-label-matcher,disallow-blanket-regex: %w", err)
			}

			seriesCountByMetric := make(map[string]uint64)
			for _, stat := range tsdbResult.SeriesCountByMetricName {
				seriesCountByMetric[stat.Name] = stat.Value
			}

			for _, metricName := range metricNames {
				if count, exists := seriesCountByMetric[metricName]; exists {
					if count > maxMetricCardinality {
						return false, &GuardrailViolation{
							Guardrail: GuardrailMaxMetricCardinality,
							Message:   fmt.Sprintf("metric %q has cardinality %d, which exceeds maximum allowed %d", metricName, count, maxMetricCardinality),
						}
					}
				}
			}
		}
	}

	// Check blanket regex patterns
	if disallowBlanketRegex {
		blanketRegexLabels, err := ExtractBlanketRegexLabels(query)
		if err != nil {
			return false, fmt.Errorf("failed to extract blanket regex labels: %w", err)
		}

		if len(blanketRegexLabels) > 0 {
			// If MaxLabelCardinality is 0, always disallow blanket regex
			if !hasMaxLabelCard || maxLabelCardinality == 0 {
				return false, &GuardrailViolation{
					Guardrail: GuardrailDisallowBlanketRegex,
					Message:   fmt.Sprintf("query uses blanket regex on label %q, which is disallowed", blanketRegexLabels[0]),
				}
			}

			if requiresTSDB {
				// Check TSDB label cardinality for blanket regex
				tsdbResult, err := client.TSDB(ctx)
				if err != nil {
					return false, fmt.Errorf(
						"cannot enforce max-label-cardinality guardrail: TSDB stats endpoint is unavailable on this backend "+
							"(Thanos Querier < v0.40.0 does not implement /api/v1/status/tsdb); "+
							"disable this guardrail with --guardrails require-label-matcher,disallow-blanket-regex: %w", err)
				}

				labelValueCountByLabel := make(map[string]uint64)
				for _, stat := range tsdbResult.LabelValueCountByLabelName {
					labelValueCountByLabel[stat.Name] = stat.Value
				}

				for _, labelName := range blanketRegexLabels {
					if count, exists := labelValueCountByLabel[labelName]; exists {
						if count > maxLabelCardinality {
							return false, &GuardrailViolation{
								Guardrail: GuardrailMaxLabelCardinality,
								Message:   fmt.Sprintf("label %q has cardinality %d, which exceeds maximum allowed %d for blanket regex", labelName, count, maxLabelCardinality),
							}
						}
					}
				}
			}
		}
	}

	return true, nil
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
