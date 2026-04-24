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

type Guardrails struct {
	// DisallowExplicitNameLabel prevents queries using explicit {__name__="..."} syntax
	DisallowExplicitNameLabel Guardrail
	// RequireLabelMatcher ensures all vector selectors have at least one non-name label matcher
	RequireLabelMatcher Guardrail
	// DisallowBlanketRegex prevents expensive regex patterns like .* or .+ on any label
	DisallowBlanketRegex Guardrail
	// MaxMetricCardinality sets the maximum allowed series count per metric (0 = disabled)
	MaxMetricCardinality Guardrail
	// MaxLabelCardinality sets the maximum allowed label value count for blanket regex
	// (0 = always disallow regex matcher provided DisallowBlanketRegex is true)
	MaxLabelCardinality Guardrail
	// tsdbAvailable is flag saying whether TSDB is available or not
	tsdbAvailable bool
}

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
	Name  string
	Value any
}

func NewGuardrail(name string, value any) Guardrail {
	return Guardrail{
		Name:  name,
		Value: value,
	}
}

func (g *Guardrails) SetMaxLabelCardinality(v uint64) {
	g.MaxLabelCardinality.Value = v
}

func (g *Guardrails) SetMaxMetricCardinality(v uint64) {
	g.MaxMetricCardinality.Value = v
}

func (g *Guardrails) GetMaxMetricCardinality() uint64 {
	mmc, ok := g.MaxMetricCardinality.Value.(uint64)
	if !ok {
		return 0
	}
	return mmc
}

func (g *Guardrails) GetMaxLabelCardinality() uint64 {
	mlc, ok := g.MaxLabelCardinality.Value.(uint64)
	if !ok {
		return 0
	}
	return mlc
}

func (g *Guardrails) IsLabelMatcherRequired() bool {
	rlm, ok := g.RequireLabelMatcher.Value.(bool)
	if !ok {
		return false
	}
	return rlm
}

func (g *Guardrails) IsExplicitNameLabelDisallowed() bool {
	denl, ok := g.DisallowExplicitNameLabel.Value.(bool)
	if !ok {
		return false
	}
	return denl
}

func (g *Guardrails) IsBlanketRegexDisallowed() bool {
	dbr, ok := g.DisallowBlanketRegex.Value.(bool)
	if !ok {
		return false
	}
	return dbr
}

func (g *Guardrails) IsTSDBAvailable() bool {
	return g.tsdbAvailable
}

func DefaultGuardrails(tsdbAvailable bool) *Guardrails {
	defaultGuardrails := &Guardrails{
		DisallowExplicitNameLabel: NewGuardrail(GuardrailDisallowExplicitNameLabel, true),
		RequireLabelMatcher:       NewGuardrail(GuardrailRequireLabelMatcher, true),
	}
	if tsdbAvailable {
		defaultGuardrails.DisallowBlanketRegex = NewGuardrail(GuardrailDisallowBlanketRegex, true)
		defaultGuardrails.MaxMetricCardinality = NewGuardrail(GuardrailMaxMetricCardinality, uint64(20000))
		defaultGuardrails.MaxLabelCardinality = NewGuardrail(GuardrailMaxLabelCardinality, uint64(500))
	}
	return defaultGuardrails
}

func ParseGuardrails(value string) (*Guardrails, error) {
	value = strings.TrimSpace(value)

	switch strings.ToLower(value) {
	case "none":
		return nil, nil
	case "all", "":
		return DefaultGuardrails(false), nil
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
			g.DisallowExplicitNameLabel = NewGuardrail(GuardrailDisallowExplicitNameLabel, true)
		case GuardrailRequireLabelMatcher:
			g.RequireLabelMatcher = NewGuardrail(GuardrailRequireLabelMatcher, true)
		case GuardrailDisallowBlanketRegex:
			g.DisallowBlanketRegex = NewGuardrail(GuardrailDisallowBlanketRegex, true)
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
	disallowExplicitNameLabel := g.IsExplicitNameLabelDisallowed()
	requireLabelMatcher := g.IsLabelMatcherRequired()
	disallowBlanketRegex := g.IsBlanketRegexDisallowed()
	maxMetricCardinality := g.GetMaxMetricCardinality()
	maxLabelCardinality := g.GetMaxLabelCardinality()

	needsTSDB := g.IsTSDBAvailable() && (maxLabelCardinality > 0 && disallowBlanketRegex) || maxMetricCardinality > 0

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
	if maxMetricCardinality > 0 && g.IsTSDBAvailable() {
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
			if maxLabelCardinality == 0 {
				return false, &GuardrailViolation{
					Guardrail: GuardrailDisallowBlanketRegex,
					Message:   fmt.Sprintf("query uses blanket regex on label %q, which is disallowed", blanketRegexLabels[0]),
				}
			}

			if g.IsTSDBAvailable() {
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
