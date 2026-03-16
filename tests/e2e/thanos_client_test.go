//go:build e2e && !openshift

package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/blang/semver/v4"
)

var (
	isThanos               bool
	thanosVer              string
	thanosHasTSDBSupported bool
)

// tsdbSupportedFrom is the first Thanos release that added /api/v1/status/tsdb
// to Thanos Querier (https://github.com/thanos-io/thanos/pull/8484).
var tsdbSupportedFrom = semver.MustParse("0.40.0")

// setupThanosDetection detects the backend type and version and logs the result.
// Called once from TestMain after mcpClient is initialised.
func setupThanosDetection() {
	isThanos, thanosVer = detectThanos()
	thanosHasTSDBSupported = thanosSupportsStatusTSDB(thanosVer)
	logBackend()
}

// logBackend prints a one-line summary of the detected backend to stderr.
func logBackend() {
	switch {
	case !isThanos:
		fmt.Fprintln(os.Stderr, "=== backend: prometheus")
	case thanosHasTSDBSupported:
		fmt.Fprintf(os.Stderr, "=== backend: thanos %s (supports /api/v1/status/tsdb)\n", thanosVer)
	default:
		fmt.Fprintf(os.Stderr, "=== backend: thanos %s (no /api/v1/status/tsdb, supported from v%s)\n", thanosVer, tsdbSupportedFrom)
	}
}

// detectThanos checks whether the backend is Thanos Querier by looking for
// the thanos_build_info metric, and returns the version if found.
func detectThanos() (bool, string) {
	// Use component="query" to distinguish Thanos Querier from the Thanos
	// sidecar that Prometheus scrapes (which has component="sidecar").
	// get_series uses /api/v1/series (metadata, no cardinality guardrails).
	resp, err := mcpClient.callToolRaw(0, "get_series", map[string]any{
		"matches": `thanos_build_info{component="query"}`,
	})
	if err != nil || resp == nil {
		return false, ""
	}
	sc, _ := resp["structuredContent"].(map[string]any)
	series, _ := sc["series"].([]any)
	if len(series) == 0 {
		return false, ""
	}
	// Extract version from the first series' labels.
	labels, _ := series[0].(map[string]any)
	rawVer, _ := labels["version"].(string)
	v, err := semver.ParseTolerant(rawVer)
	if err != nil {
		return true, ""
	}
	return true, "v" + v.String()
}

// skipIfThanosLacksTSDB skips the test when the backend is Thanos < v0.40.0,
// which does not support /api/v1/status/tsdb required by cardinality guardrails.
func skipIfThanosLacksTSDB(t *testing.T) {
	t.Helper()
	if !isThanos || thanosHasTSDBSupported {
		return
	}
	t.Skipf("Thanos %s does not support /api/v1/status/tsdb (supported from v%s); run obs-mcp with --guardrails require-label-matcher,disallow-blanket-regex to disable cardinality checks", thanosVer, tsdbSupportedFrom)
}

// thanosSupportsStatusTSDB returns true when the Thanos version is >= tsdbSupportedFrom.
// An empty or unparseable version string is treated as unsupported.
func thanosSupportsStatusTSDB(version string) bool {
	v, err := semver.ParseTolerant(strings.TrimPrefix(version, "v"))
	if err != nil {
		return false
	}
	return v.GTE(tsdbSupportedFrom)
}
