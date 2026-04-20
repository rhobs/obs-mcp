# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- MCPChecker eval framework for automated tool verification ([#34](https://github.com/rhobs/obs-mcp/pull/34), [#66](https://github.com/rhobs/obs-mcp/pull/66))
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

- Migrate to `modelcontextprotocol/go-sdk` ([#53](https://github.com/rhobs/obs-mcp/pull/53))
- Summarize range query results by default ([#37](https://github.com/rhobs/obs-mcp/pull/37))
- Improved `list_metrics` prompt for better metric discovery ([#44](https://github.com/rhobs/obs-mcp/pull/44))
- Hardened Containerfile for robustness and faster builds ([#64](https://github.com/rhobs/obs-mcp/pull/64))
- Bumped sigstore/cosign-installer GitHub Action ([#61](https://github.com/rhobs/obs-mcp/pull/61))
