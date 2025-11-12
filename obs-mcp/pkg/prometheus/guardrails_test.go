package prometheus

import "testing"

func TestIsSafeQuery(t *testing.T) {
	tests := map[string]bool{
		// Rule 1: __name__ queries
		`{__name__="http_requests_total"}`:      false,
		`sum({__name__="http_requests_total"})`: false,

		// Rule 2: Vector selector without non-name label matchers
		`http_requests_total`:                          false,
		`some_very_high_cardinality_metric`:            false,
		`sum(http_requests_total)`:                     false, // No label matchers
		`rate(http_requests_total[5m])`:                false, // No label matchers
		`sum by (job) (rate(http_requests_total[5m]))`: false, // No label matchers

		// Rule 3: Expensive regex
		`http_requests_total{pod=~"web-.*"}`:  true,  // Has selective matcher "web-"
		`rate(my_metric{instance!~".+"}[5m])`: false, // Pure wildcard, no selective matcher
		`http_requests_total{pod=~".*"}`:      false, // Pure wildcard, no selective matcher
		`http_requests_total{job=~"api-.*"}`:  true,  // Has selective matcher "api-"

		// Complex failure cases combining rules
		`avg(rate(http_requests_total[5m]))`:                                              false, // No label matchers
		`sum by (status) (rate(http_requests_total[5m]))`:                                 false, // No label matchers
		`histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`:        false, // No label matchers
		`avg(http_requests_total{pod=~".*"})`:                                             false, // Pure wildcard regex
		`sum by (job) (rate(http_requests_total{instance=~".+"}[5m]))`:                    false, // Pure wildcard regex
		`histogram_quantile(0.99, sum(rate(http_latency_bucket{pod=~".*"}[5m])) by (le))`: false, // Pure wildcard regex
		`rate(http_requests_total{job="api"}[5m]) / rate(http_requests_total[5m])`:        false, // Second selector has no labels
		`avg_over_time(cpu_usage[10m])`:                                                   false, // No label matchers

		`http_requests_total{job="api"}`:                          true,
		`http_requests_total{pod="web-1"}`:                        true,
		`sum by (job) (rate(http_requests_total{job="api"}[5m]))`: true,
		`histogram_quantile(0.9, sum(rate(http_request_duration_seconds_bucket{job="api"}[5m])) by (le, job))`: true,
		`up{job="prometheus"} == 0`:                true,
		`rate(http_requests_total{job="api"}[5m])`: true,
		`sum(http_requests_total{job="api"})`:      true,

		// Safe regex on high-cardinality labels
		`http_requests_total{pod=~"web-1|web-2"}`: true,
		`http_requests_total{pod=~"web-[0-9]+"}`:  true,
		`http_requests_total{job=~"api-1|api-2"}`: true,

		// Complex safe cases with aggregations and functions
		`avg(rate(http_requests_total{job="api"}[5m]))`:                                                     true,
		`sum by (status) (rate(http_requests_total{job="api"}[5m]))`:                                        true,
		`histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{job="api"}[5m]))`:               true,
		`avg by (instance) (rate(http_requests_total{job="api",status=~"5.."}[5m]))`:                        true,
		`sum(rate(http_requests_total{job="api"}[5m])) / sum(rate(http_requests_total{job="backend"}[5m]))`: true,
		`max_over_time(cpu_usage{instance="server-1"}[10m])`:                                                true,
		`topk(5, rate(http_requests_total{environment="production"}[5m]))`:                                  true,
		`histogram_quantile(0.99, sum by (le) (rate(http_latency_bucket{service=~"api-.*"}[5m])))`:          true,
		`avg(irate(node_cpu_seconds_total{mode!="idle",instance="localhost"}[5m])) by (cpu)`:                true,
	}

	for query, expectedSafe := range tests {
		t.Run(query, func(t *testing.T) {
			safe, err := isSafeQuery(query)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if safe != expectedSafe {
				t.Errorf("isSafeQuery(%q) = %v, want %v", query, safe, expectedSafe)
			}
		})
	}
}
