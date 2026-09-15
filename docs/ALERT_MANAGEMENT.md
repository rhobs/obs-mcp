# Alert management toolset

The `observability/alert-management` toolset lets MCP clients list, preview, create, update, and delete OpenShift alert rules through the [monitoring-plugin management API](https://github.com/openshift/monitoring-plugin/blob/main/docs/alert-management.md). It forwards the caller's bearer token. It does not change that API.

This toolset is **opt-in** on the standalone binary (default `--toolsets` is `observability/metrics` only). Sample ConfigMaps under `manifests/` enable it. List, preview, create, update, and delete are always registered; the caller's RBAC on the management API gates writes (HTTP 403).

Per-tool schemas live in [TOOLS.md](../TOOLS.md#alert-management). Authentication and URL setup are in [DEPLOYMENT.md](DEPLOYMENT.md).

## Enable the toolset

Include it in `--toolsets` and point at the monitoring-plugin management API (plugin backend, typically port **9443**, not the Alertmanager proxy).

```shell
go run ./cmd/obs-mcp/ \
  --listen 127.0.0.1:9100 \
  --auth-mode kubeconfig \
  --insecure \
  --toolsets observability/metrics,observability/alert-management \
  --alert-mgmt-api-url http://localhost:9443
```

| Setting | Flag / env | Notes |
|---|---|---|
| Enable tools | `--toolsets` … `observability/alert-management` | Required |
| Management API URL | `--alert-mgmt-api-url` / `ALERT_MGMT_API_URL` | Required in `header` mode. In `kubeconfig` mode, defaults to `http://localhost:9443` if unset |
| Auth | Same `--auth-mode` as other backends | Token is forwarded to the management API; writes follow that user's RBAC |

TOML (openshift-mcp-server ConfigMap):

```toml
toolsets = ["observability/metrics", "observability/alert-management"]

[toolset_configs."observability/alert-management"]
auth_mode = "kubeconfig"
management_api_url = "https://monitoring-plugin.openshift-monitoring.svc:9443"
```

Use the URL that actually serves `/api/v1/alerting/rules` on your cluster.

## Tools

| Tool | What it does |
|---|---|
| `list_alert_rules` | List rules (id, expr, severity, namespace, source, `management_status`) |
| `preview_alert_rule` | Dry-run create or update; reports `writable` and optional `managedBy` |
| `create_alert_rule` | Create a user-defined or platform rule |
| `update_alert_rule` | Patch labels, severity, classification, or drop/restore (`alerting_rule_enabled`) by rule id |
| `delete_alert_rules` | Delete by rule id (bulk, per-rule status) |

Writes identify rules by stable id (`openshift_io_alert_rule_id`, `rid_…`), not by alert name. Create/update/delete/preview reject `cluster` and `cluster_labels`. List may forward those as query parameters; they are optional filters, not a way to write across clusters.

## Silences (Alertmanager)

Silence create, update, and delete live on `observability/metrics` (the default toolset). They call Alertmanager API v2, not the management API this toolset uses:

| Tool | What it does |
|---|---|
| `get_silences` | List silences |
| `create_silence` | Create a silence (`comment` required; `labels` and/or `matchers`; default duration 2h) |
| `update_silence` | Change an existing silence by `silence_id` |
| `delete_silence` | Expire a silence by `silence_id` |

Use `get_alerts` first so matchers include more than `alertname` when you mean a specific firing instance. Explain the matchers and wait for agreement before create/update/delete. GitOps- and operator-managed **rules** that cannot be edited can still have their notifications muted this way.

## What an agent should do

obs-mcp injects this workflow into the MCP server instructions when the toolset is enabled. Clients should follow it even if a user speaks in alert names.

### Choose the right action

| User asked to | Tool |
|---|---|
| Mute / silence / stop paging | `create_silence` on `observability/metrics` (not delete or drop) |
| Drop / disable a **platform** rule | `update_alert_rule` with `alerting_rule_enabled=false` |
| Remove a **user-created** rule | `delete_alert_rules` |
| Change PromQL, name, `for`, or annotations | Not `update_alert_rule` (create a replacement after confirmation, or edit in Git) |

`list_alert_rules` is rule definitions. `get_alerts` is firing instances. Do not PATCH an id taken only from `get_alerts`.

### Create

1. Choose platform vs user-defined from what the user said. `prometheus_rule_name` + `namespace` → user-defined. Omit `prometheus_rule_name` → **platform**. Do not invent a PrometheusRule name. If unclear, ask.
2. Call `list_alert_rules` (filter by namespace when the user named one).
3. Compare the proposed PromQL with each listed `expr` (ignore trivial whitespace).
4. If a rule already has that expression, stop. Show `id`, name, namespace, source, and `management_status`. Ask whether to **update that rule**, **create a duplicate**, or **cancel**. Do not create unless the user explicitly chooses to create anyway.
5. If `observability/metrics` is enabled, confirm metric names in the expr with `list_metrics`.
6. If none match: `preview_alert_rule`, explain the plan (including platform vs user-defined), wait for agreement, then `create_alert_rule`.

Do not put secrets in labels or annotations.

### Update or delete by name

Alert name is not unique (same name in different namespaces, or different expr/labels).

1. `list_alert_rules` and keep rows whose `name` equals the requested alert name.
2. Zero matches: say so and stop.
3. Two or more: stop. List each hit (`id`, name, namespace, expr, severity, source, `management_status`) and ask **which rule**. Do not send every matching id. Changing all of them is allowed only if the user explicitly asks for that and confirms.
4. One match: still confirm before writing.

### Confirm before any write

Before `create_alert_rule`, `update_alert_rule`, or `delete_alert_rules`:

1. Call `preview_alert_rule` for create/update (delete: describe the listed rule; preview is create/update only).
2. Explain what will change (name, id, namespace, fields, preview `resources`).
3. Wait until the user explicitly agrees. “Update Watchdog” is not agreement to a specific patch.

Update/delete responses are per-rule `statusCode` (the envelope can be 200 with mixed results). After a label update, use the **returned** id; it may change. Do not retry 400, 404, or 413.

### GitOps-managed and operator-managed rules

Listed rules include `management_status`: `user-created`, `gitops`, or `operator`. Preview may set `writable: false` and `managedBy`.

| Status | What the MCP should do |
|---|---|
| `gitops` | Do not create, update, or delete (the API rejects it). Tell the user the rule is GitOps-managed. Direct them to edit the owning `PrometheusRule` or `AlertingRule` **in Git**, commit, and let reconciliation apply it. Use listed namespace, group, and name to find the manifest. |
| `operator` and not writable | Do not retry. Persistent spec edits belong in operator configuration. Notifications can be muted with Alertmanager silences. |
| `user-created` | After preview and confirmation, writes go through the management API. |

Do not retry a 409/405 GitOps or operator-managed conflict.

A 403 means the caller lacks RBAC for that namespace or scope. Write tools stay registered; the management API enforces permissions.

## Example prompts

These belong in a client (Cursor, Lightspeed, Claude). The agent should list, preview, and ask before writing.

- List user-defined alert rules in namespace `my-app`.
- Create an alert named `WidgetDown` with expr `up{job="widget"} == 0` and severity `critical` in PrometheusRule `user-alerts` in `my-app`.
- Update the severity of alert `WidgetDown` to `warning`.
- Why can’t you change Watchdog? If it is GitOps-managed, tell me how to change it.

More examples: [docs/dev/PROMPTS.md](dev/PROMPTS.md).

## Related docs

- [TOOLS.md — Alert Management](../TOOLS.md#alert-management) — generated tool schemas
- [TOOLS.md — Alertmanager](../TOOLS.md#alertmanager) — silence tools (`get_silences`, `create_silence`, `update_silence`, `delete_silence`)
- [DEPLOYMENT.md](DEPLOYMENT.md) — auth modes and `ALERT_MGMT_API_URL`
- [monitoring-plugin alert management API](https://github.com/openshift/monitoring-plugin/blob/main/docs/alert-management.md) — HTTP API consumed as-is
