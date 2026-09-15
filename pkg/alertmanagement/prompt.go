package alertmanagement

// ServerPrompt provides instructions for LLMs using the alert management toolset.
const ServerPrompt = `
## Alert Rule Management Workflow

This toolset manages OpenShift alert rules via the monitoring-plugin management API.
It supports both in-console (Lightspeed) and non-console (Cursor, Claude, CLI) usage.

You must follow this workflow. Do not skip confirmation, and do not guess which rule to change.

### Choose the right action

Map the user's words before calling a write tool:
- mute, silence, or stop paging → create_silence on observability/metrics (Alertmanager). This toolset has no silence tools. If create_silence is not available, say so. Do not delete or drop a rule to mute notifications.
- disable or drop a platform alert → update_alert_rule with alerting_rule_enabled=false (platform rules only).
- restore a dropped platform alert → alerting_rule_enabled=true.
- remove a user-created rule → delete_alert_rules.
- change PromQL, alert name, for, or annotations → not update_alert_rule (see Update limits).

### Always discover first

1. Call list_alert_rules before create, update, or delete. That tool returns rule definitions. get_alerts returns firing instances; do not treat alert instance labels or silencedBy ids as a rule id, and do not PATCH an id from get_alerts without listing the rule.
2. Identify rules by stable id (openshift_io_alert_rule_id). Alert name is not unique. If a listed rule has an empty id, do not invent one; explain that the rule cannot be updated or deleted through this API.
3. Show list_alert_rules warnings to the user.

### Create: choose platform vs user-defined; check for an existing expression

Omitting prometheus_rule_name creates a platform alerting rule. Setting prometheus_rule_name and namespace creates a user-defined rule in that PrometheusRule. Do not default to either, and do not invent a PrometheusRule name.

When the user asks to create a rule:
1. Choose the target from what they said. If they named a PrometheusRule and namespace, use those (user-defined). If they asked for a platform or cluster-monitoring rule, omit prometheus_rule_name. If it is unclear, ask before preview.
2. Call list_alert_rules (filter by namespace when the user named one).
3. Compare the proposed PromQL with each listed expr (ignore trivial whitespace).
4. If any existing rule has the same expression, stop. Show id, name, namespace, source, and management_status. Ask whether to update that rule, create a duplicate, or cancel. Do not call create_alert_rule unless the user explicitly chooses to create anyway. If they asked to change the expression of an existing rule, do not PATCH expr (unsupported); propose a replacement create or Git, after confirmation.
5. If observability/metrics is enabled, call list_metrics and confirm the expr's metric names exist before preview. Alert expr is not covered by PromQL query guardrails.
6. If none match: preview_alert_rule, explain the planned create (user-defined vs platform, and which PrometheusRule if any), and wait for the user to agree before create_alert_rule.

Do not put secrets, tokens, or passwords in labels or annotations.

### Update limits

update_alert_rule can only change labels, severity, classification, or drop/restore (alerting_rule_enabled).
It cannot change expr, alert name, for, or annotations.
alerting_rule_enabled (drop/restore) is platform-only. Do not set it on source=user rules.
Classification parameters are for platform rules. Do not use them on source=user rules unless the user explicitly asked to set those label keys.

### Update or delete by name: disambiguate

When the user names an alert (for example "update Watchdog") rather than an id:
1. Call list_alert_rules and keep every result whose name equals that alert name.
2. If zero matches, say so and stop.
3. If two or more match, stop. List each hit with id, name, namespace, expr, severity, source, and management_status. Ask which specific rule they mean. Do not pass every matching id. Do not update or delete all matches unless the user explicitly asks to change all of them and then confirms each action (or clearly confirms a bulk change of those ids).
4. If exactly one matches, still confirm before changing it.

### Confirm before any write

Before create_alert_rule, update_alert_rule, or delete_alert_rules:
1. Call preview_alert_rule (delete: describe the listed rule; preview is create/update only).
2. Explain in plain language what will change (name, id, namespace, fields, and which Kubernetes resources preview lists).
3. Stop and wait until the user explicitly agrees. A request like "update Watchdog" is not agreement to a specific patch.

### After update or delete

The HTTP call can succeed while individual rules fail. Read each per-rule statusCode and message and report failures. Do not claim overall success from the envelope alone.
After a label update, the stable id may change. Use the id returned in the result for any later call. Do not reuse the old id.

### GitOps-managed and operator-managed rules

If management_status is gitops, or preview returns writable=false with managedBy=gitops:
- Do not call create, update, or delete (the API rejects it).
- Tell the user the rule is GitOps-managed and this API cannot persist the change.
- Direct them to edit the owning PrometheusRule or AlertingRule in Git (the GitOps source of truth), commit, and let reconciliation apply it. Use the listed namespace, group, and name to identify the manifest.

If management_status is operator, or preview returns writable=false with managedBy=operator:
- Do not retry writes that preview marks not writable.
- Tell the user the rule is operator-managed. Persistent spec edits belong in the operator configuration, not this API. Notifications can be muted with create_silence (Alertmanager; observability/metrics).

Do not retry a 409/405 GitOps or operator-managed conflict.
Do not retry 400, 404, or 413; explain the error. A 403 means the caller lacks RBAC for that namespace or cluster scope; do not retry.

### Other rules

- Severity values: critical, warning, info, none.
- Authorization uses the caller's bearer token. Write tools are always registered; the management API enforces RBAC.
- Do not pass cluster or cluster_labels on create, update, delete, or preview.
`

const (
	listAlertRulesPrompt = `List managed OpenShift alert rules from the monitoring-plugin management API.

Returns each rule's stable ID, PromQL expression, severity, namespace, source (platform or user), and management_status (user-created, gitops, or operator).
This is rule definitions, not firing instances (use get_alerts for those). Use this before create (to detect the same expr) and before update/delete (to resolve names to ids).
Alert name is not unique: if several rows share a name, present them to the user and wait; do not pick one or change all of them.
If id is empty, do not invent an id. Surface warnings from the response.
GitOps-managed and operator-managed rules cannot be updated or deleted through this API; explain that and point GitOps users at editing the rule in Git.

Optional filters: namespace, severity, state (pending, firing, silenced), source, cluster, cluster_labels, matchers, and extra label equality filters.`

	createAlertRulePrompt = `Create an OpenShift alert rule via POST /api/v1/alerting/rules.

Set prometheus_rule_name and namespace for a user-defined rule in that PrometheusRule. Omit prometheus_rule_name to create a platform alerting rule. Do not invent a PrometheusRule name; if the target is unclear, ask before calling.
Do not put secrets in labels or annotations.
Do not pass cluster or cluster_labels on this operation.

Before calling this tool:
1. list_alert_rules and check whether any rule already has the same PromQL expression. If one does, ask the user whether to update that rule instead of creating a duplicate.
2. If observability/metrics is enabled, verify expr metric names with list_metrics.
3. preview_alert_rule for the create payload.
4. Explain the planned create (user-defined vs platform) and wait for the user to agree.

Do not retry a 409/405 GitOps or operator-managed conflict. If the target PrometheusRule is GitOps-managed, tell the user to add the alert in Git instead. Do not retry 400, 404, or 413.`

	updateAlertRulePrompt = `Update alert rule labels, severity, classification, or drop/restore (alerting_rule_enabled) via PATCH /api/v1/alerting/rules.

Cannot change expr, alert name, for, or annotations. If the user wants a new expression, do not call this tool.
Provide rule_id or rule_ids (1-100) of rules the user has confirmed. At least one mutation field is required.
alerting_rule_enabled cannot be combined with labels, severity, or classification in the same request.
Drop/restore (alerting_rule_enabled) is platform-only. Do not set it on source=user rules.
Do not use this tool to silence an alert; use create_silence.
Do not pass cluster or cluster_labels on this operation.

Never resolve an alert name to every matching id. If list_alert_rules returns more than one rule with that name, ask which id to use. Do not update all matches unless the user explicitly asks for that and confirms.
Call preview_alert_rule, explain the change, and wait for agreement before this call.
Read each per-rule statusCode. After a label update, use the returned id; it may differ from the id you sent.

management_status gitops: do not call this tool; tell the user to edit the PrometheusRule or AlertingRule in Git.
management_status operator or preview writable=false: do not retry; explain operator ownership. Silences may mute notifications.
Do not retry 400, 404, or 413.`

	deleteAlertRulesPrompt = `Delete alert rules by stable ID via DELETE /api/v1/alerting/rules.

Use this to remove a user-created rule, not to mute notifications (create_silence) and not to drop a platform rule (alerting_rule_enabled=false).
Provide rule_id or rule_ids (1-100) the user has confirmed. The response always includes per-rule statusCode and optional message so partial success is visible; report each failure.
GitOps-managed and operator-managed rules cannot be deleted.
Do not pass cluster or cluster_labels on this operation.

If the user asked by name and several rules share that name, ask which id to delete. Do not delete all matches unless the user explicitly asks for that and confirms.
If management_status is gitops, tell the user to remove the rule in Git instead of calling this tool.
Do not retry 400, 404, or 413.`

	previewAlertRulePrompt = `Dry-run a create or update without persisting cluster changes via POST /api/v1/alerting/rules/preview.

Create preview: set alert and expr. Include prometheus_rule_name and namespace for a user-defined rule; omit prometheus_rule_name for a platform rule. Do not invent a PrometheusRule name.
Update preview: set rule_id plus labels, severity, classification, or alerting_rule_enabled. Update cannot preview expr, alert name, for, or annotation changes.
Do not pass cluster or cluster_labels on this operation.

Required before create_alert_rule or update_alert_rule. After preview, explain the plan (writable, managedBy, resources, desiredRule) and wait for the user to agree.
If writable is false, do not call create_alert_rule or update_alert_rule.
If managedBy is gitops, tell the user to make the change in Git (PrometheusRule or AlertingRule source of truth).
If managedBy is operator, explain that the operator owns the rule and this API will not persist the spec change.`
)
