# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Bump `kubernetes-mcp-server` to v0.0.65 and adapt toolset `GetTools` to the new `api.FilteringProvider` signature (replaces removed `api.Openshift` from upstream [PR #1196](https://github.com/containers/kubernetes-mcp-server/pull/1196))

## [v0.5.0] - 2026-07-06

### Added

- CI: skip lint, unit, and e2e workflows for doc-only changes ([#132](https://github.com/rhobs/obs-mcp/pull/132))
- CI: add multitenancy tests for tracing, split files ([#136](https://github.com/rhobs/obs-mcp/pull/136))
- Evals: add suite label to loki eval tasks ([#129](https://github.com/rhobs/obs-mcp/pull/129))
- Skills: mcpchecker-results-triage for evaluating and triaging eval results ([#149](https://github.com/rhobs/obs-mcp/pull/149))

### Changed

- Namespace all toolset names with `observability/` prefix for clarity in user configuration: `metrics` → `observability/metrics`, `traces` → `observability/traces`, `logs` → `observability/logs`, `otelcol` → `observability/otelcol` ([#146](https://github.com/rhobs/obs-mcp/pull/146))
- `tempo_search_traces`: establish latency baseline instead of guessing ([#143](https://github.com/rhobs/obs-mcp/pull/143))
- Bump Go dependencies ([#151](https://github.com/rhobs/obs-mcp/pull/151))
- Evals: keep latest results in git repo, remove llmJudge reason ([#149](https://github.com/rhobs/obs-mcp/pull/149))

### Fixed

- Fix stdin mode ([#148](https://github.com/rhobs/obs-mcp/pull/148))
- Evals: fix false positives ([#149](https://github.com/rhobs/obs-mcp/pull/149))

## [v0.4.0] - 2026-06-17

### Added

- `logs` toolset for [Loki](https://grafana.com/oss/loki/) log management: `loki_list_instances`, `loki_label_names`, `loki_label_values`, and `loki_query_range` ([#107](https://github.com/rhobs/obs-mcp/pull/107))
- Static `--tempo-url` flag for configuring Tempo endpoint without route discovery ([#124](https://github.com/rhobs/obs-mcp/pull/124))
- `make test-e2e-run` target for running the server locally against cluster backends with automatic port-forwarding ([#124](https://github.com/rhobs/obs-mcp/pull/124))
- Loki e2e tests and mcpchecker eval tasks for log query validation ([#107](https://github.com/rhobs/obs-mcp/pull/107))
- `andreasgerstmayr` added to OWNERS_ALIASES ([#119](https://github.com/rhobs/obs-mcp/pull/119))

### Changed

- Extract authentication code into a shared `pkg/auth` package ([#113](https://github.com/rhobs/obs-mcp/pull/113))
- Improve TOOLS.md generation with category grouping, table of contents, collapsible sections, and quick-reference table ([#125](https://github.com/rhobs/obs-mcp/pull/125))
- Add `category` label to traces and otelcol eval tasks ([#118](https://github.com/rhobs/obs-mcp/pull/118))
- Improve port-forwarding failure detection in e2e setup ([#124](https://github.com/rhobs/obs-mcp/pull/124))
- Bump Go to 1.26.3 and upgrade module dependencies ([#127](https://github.com/rhobs/obs-mcp/pull/127))

### Fixed

- Fix broken references, typos, and stale content across markdown documentation (README, TESTING, DEPLOYMENT, RELEASE, evals) ([#126](https://github.com/rhobs/obs-mcp/pull/126))

## [v0.3.0] - 2026-06-05

### Added

- `otelcol` toolset for [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) configuration assistance: `otelcol_list_components`, `otelcol_get_component_schema`, `otelcol_validate_config`, and `otelcol_get_versions` ([#77](https://github.com/rhobs/obs-mcp/pull/77))
- Comprehensive unit and e2e tests for the Tempo toolset ([#105](https://github.com/rhobs/obs-mcp/pull/105))
- otelcol smoke test in the e2e suite ([#114](https://github.com/rhobs/obs-mcp/pull/114))
- `make update-go-deps` Makefile target for Go dependency maintenance ([#115](https://github.com/rhobs/obs-mcp/pull/115))

### Changed

- Rename CLI flag `-tempo.use-route` to `-traces.use-route` ([#111](https://github.com/rhobs/obs-mcp/pull/111))
- Consolidate e2e cluster setup in `hack/e2e/setup.sh` and unify Kubernetes/OpenShift deployment manifests ([#112](https://github.com/rhobs/obs-mcp/pull/112))
- Bump Go to 1.26 and upgrade module dependencies ([#115](https://github.com/rhobs/obs-mcp/pull/115))
- Bump Kind to v0.32.0 for Kubernetes 1.36 e2e tests ([#116](https://github.com/rhobs/obs-mcp/pull/116))

### Fixed

- Flatten `AdditionalFields` directly into tool `Meta` metadata ([#110](https://github.com/rhobs/obs-mcp/pull/110))

## [v0.2.0] - 2026-05-14

### Added

- Traces toolset for [Grafana Tempo](https://grafana.com/docs/tempo/latest/): `tempo_list_instances`, `tempo_get_trace_by_id`, `tempo_search_traces`, `tempo_search_tags`, and `tempo_search_tag_values` ([#29](https://github.com/rhobs/obs-mcp/pull/29))

### Changed

- Bump `prometheus/prometheus` and adapt PromQL parsing and metric-name label checks for the updated parser API ([#98](https://github.com/rhobs/obs-mcp/pull/98))
- Improve auto-generated `TOOLS.md`: GFM-safe tables, compact rows, types legend (`[!NOTE]`), and clearer intro for all backends

### Fixed

- Bump `github.com/go-openapi/runtime` to v0.30.0 (and related `go-openapi/*` modules via `go mod tidy`) so release builds no longer fail when resolving `github.com/go-openapi/testify/v2` test dependencies.

## [v0.1.4] - 2026-05-06

### Added

- Unify guardrails configuration with `!prefix` exclusion and allow disabling all TSDB guardrails at once ([#86](https://github.com/rhobs/obs-mcp/pull/86))

### Fixed

- Validate `max-metric-cardinality = 0` in toolset config and improve `--help` messages for guardrails ([#86](https://github.com/rhobs/obs-mcp/pull/86))

### Changed

- Bump kubernetes-mcp-server dependency ([#94](https://github.com/rhobs/obs-mcp/pull/94))
- Document eval sync process with openshift-mcp-server ([#87](https://github.com/rhobs/obs-mcp/pull/87))

## [v0.1.3] - 2026-04-28

### Fixed

- Strip Bearer prefix from authorization header in `readTokenFromCtx` to prevent duplicate scheme ([#82](https://github.com/rhobs/obs-mcp/pull/82))

### Added

- Enable CodeRabbit for automated PR reviews ([#83](https://github.com/rhobs/obs-mcp/pull/83))

## [v0.1.2] - 2026-04-28

### Fixed

- Fix Prometheus client to get bearer token from request context ([#78](https://github.com/rhobs/obs-mcp/pull/78))

### Changed

- Sync mcpchecker eval task definitions with openshift-mcp-server and bump mcpchecker to v0.0.16 ([#79](https://github.com/rhobs/obs-mcp/pull/79))

## [v0.1.1] - 2026-04-23

### Fixed

- Rename the toolset config key consistently everywhere ([#75](https://github.com/rhobs/obs-mcp/pull/75))
- Strengthen mcpchecker eval judge checks with tool-derived values and OpenShift compatibility fixes ([#71](https://github.com/rhobs/obs-mcp/pull/71))

## [v0.1.0] - 2026-04-22

> Note: Some entries lack PR numbers because they were developed in the original [monorepo](https://github.com/jhadvig/genie-plugin) before migration to [rhobs/obs-mcp](https://github.com/rhobs/obs-mcp).

### Added

- MCP server exposing Prometheus and Alertmanager as tools via Model Context Protocol
- Tools: `list_metrics`, `execute_instant_query`, `execute_range_query`, `get_label_names`, `get_label_values`, `get_series`, `get_alerts`, `get_silences` ([#15](https://github.com/rhobs/obs-mcp/pull/15), [#25](https://github.com/rhobs/obs-mcp/pull/25))
- `show_timeseries` visualization tool for UI clients ([#36](https://github.com/rhobs/obs-mcp/pull/36))
- Authentication modes: `kubeconfig`, `serviceaccount`, `header` ([#10](https://github.com/rhobs/obs-mcp/pull/10), [#14](https://github.com/rhobs/obs-mcp/pull/14))
- PromQL safety guardrails: disallow explicit name label, require label matcher, disallow blanket regex, TSDB cardinality checks
- Range query result summarization with optional full response flag ([#37](https://github.com/rhobs/obs-mcp/pull/37))
- Metric existence validation before query execution ([#8](https://github.com/rhobs/obs-mcp/pull/8))
- Auto-discovery of Prometheus/Thanos and Alertmanager routes in kubeconfig mode ([#11](https://github.com/rhobs/obs-mcp/pull/11))
- `--metrics-backend` flag for controlling route discovery ([#11](https://github.com/rhobs/obs-mcp/pull/11))
- Structured `slog` logging with `--log-level` flag
- Kubernetes deployment manifests with RBAC
- GoReleaser-based release pipeline with cosign artifact signing ([#54](https://github.com/rhobs/obs-mcp/pull/54))
- MCP Inspector compose setup for local testing with Docker and Podman ([#56](https://github.com/rhobs/obs-mcp/pull/56))
- MCPChecker eval framework for automated tool verification ([#34](https://github.com/rhobs/obs-mcp/pull/34), [#66](https://github.com/rhobs/obs-mcp/pull/66), [#69](https://github.com/rhobs/obs-mcp/pull/69))
- Dependabot for Go modules and GitHub Actions ([#60](https://github.com/rhobs/obs-mcp/pull/60))

### Fixed

- Validate and sanitize `name_regex` input to prevent PromQL matcher injection ([#58](https://github.com/rhobs/obs-mcp/pull/58))
- Use service-ca file for TLS in prometheus client ([#50](https://github.com/rhobs/obs-mcp/pull/50))
- Use empty map when no labels are present in summary ([#52](https://github.com/rhobs/obs-mcp/pull/52))
- Use configured transport for alertmanager client ([#38](https://github.com/rhobs/obs-mcp/pull/38))
- Fail fast on missing URLs for non-kubeconfig modes ([#41](https://github.com/rhobs/obs-mcp/pull/41))
- Detect and log actual backend type in prometheus loader ([#42](https://github.com/rhobs/obs-mcp/pull/42))
- Relaxed range query params validation to accept flexible time formats ([#55](https://github.com/rhobs/obs-mcp/pull/55))
- Propagate range query summary changes to toolset properly ([#65](https://github.com/rhobs/obs-mcp/pull/65))

### Changed

- Rename toolset registration from `observability` to `metrics` ([#68](https://github.com/rhobs/obs-mcp/pull/68), [#72](https://github.com/rhobs/obs-mcp/pull/72))
- Migrate to `modelcontextprotocol/go-sdk` ([#53](https://github.com/rhobs/obs-mcp/pull/53))
- Summarize range query results by default ([#37](https://github.com/rhobs/obs-mcp/pull/37))
- Improved `list_metrics` prompt for better metric discovery ([#44](https://github.com/rhobs/obs-mcp/pull/44))
- Hardened Containerfile for robustness and faster builds ([#64](https://github.com/rhobs/obs-mcp/pull/64))
- Bumped sigstore/cosign-installer GitHub Action ([#61](https://github.com/rhobs/obs-mcp/pull/61))
