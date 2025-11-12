package prometheus

import (
	"fmt"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

// isSafeQuery analyzes a PromQL query string and returns false if it's
// deemed unsafe or too expensive based on a set of rules.
//
// Returns (false, error) only if the query syntax is invalid.
// Returns (false, nil) if the query is valid but violates a rule.
// Returns (true, nil) if the query is valid and passes all rules.
func isSafeQuery(query string) (bool, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return false, fmt.Errorf("failed to parse query: %w", err)
	}

	foundUnsafe := false
	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		switch n := node.(type) {
		case *parser.VectorSelector:
			hasNonNameMatcher := false

			for _, m := range n.LabelMatchers {
				// Rule 1: Check for explicit __name__ label query
				if m.Name == labels.MetricName && n.Name == "" {
					foundUnsafe = true
					return fmt.Errorf("unsafe")
				}

				if m.Name != labels.MetricName {
					hasNonNameMatcher = true
				}

				// Rule 3: Check for expensive regex matchers on *any* label i.e blanket matchers
				isRegex := m.Type == labels.MatchRegexp || m.Type == labels.MatchNotRegexp
				if isRegex {
					if m.Value == ".*" || m.Value == ".+" {
						foundUnsafe = true
						return fmt.Errorf("unsafe")
					}
				}
			}

			// Rule 2: All vector selectors must have at least one non-name label matcher
			if !hasNonNameMatcher {
				foundUnsafe = true
				return fmt.Errorf("unsafe")
			}
		}
		return nil
	})

	if foundUnsafe {
		return false, nil
	}

	return true, nil
}
