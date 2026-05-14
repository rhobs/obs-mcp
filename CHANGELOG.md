# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.2.0] - 2026-05-14

### Added

- Traces toolset for [Grafana Tempo](https://grafana.com/docs/tempo/latest/): `tempo_list_instances`, `tempo_get_trace_by_id`, `tempo_search_traces`, `tempo_search_tags`, and `tempo_search_tag_values` ([#29](https://github.com/rhobs/obs-mcp/pull/29))

### Changed

- Bump `prometheus/prometheus` and adapt PromQL parsing and metric-name label checks for the updated parser API ([#98](https://github.com/rhobs/obs-mcp/pull/98))
- Improve auto-generated `TOOLS.md`: GFM-safe tables, compact rows, types legend (`[!NOTE]`), and clearer intro for all backends

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
