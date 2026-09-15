package alertmanagement

const (
	alertRuleIDLabel      = "openshift_io_alert_rule_id"
	alertSourceLabel      = "openshift_io_alert_source"
	ruleManagedByLabel    = "openshift_io_rule_managed_by"
	managedClusterLabel   = "managed_cluster"
	alertSourceUser       = "user"
	alertSourcePlatform   = "platform"
	managementUserCreated = "user-created"
	managementOperator    = "operator"
	managementGitOps      = "gitops"

	clusterQueryKey       = "cluster"
	clusterLabelsQueryKey = "cluster_labels"

	errClusterWriteUnsupported = "cluster and cluster_labels are not supported on this operation; omit them"

	maxBulkRuleIDs = 100
	rulesAPIPath   = "/api/v1/alerting/rules"
	previewAPIPath = "/api/v1/alerting/rules/preview"
)

var validSeverities = map[string]struct{}{
	"critical": {},
	"warning":  {},
	"info":     {},
	"none":     {},
}

var validStates = map[string]struct{}{
	"pending":  {},
	"firing":   {},
	"silenced": {},
}

var validSources = map[string]struct{}{
	alertSourcePlatform: {},
	alertSourceUser:     {},
}

var reservedListLabelKeys = map[string]struct{}{
	clusterQueryKey:       {},
	clusterLabelsQueryKey: {},
}

// ListAlertRulesOutput is the structured result of list_alert_rules.
type ListAlertRulesOutput struct {
	Rules    []ListedAlertRule `json:"rules" jsonschema:"Managed alert rules matching the filters"`
	Warnings []string          `json:"warnings,omitempty" jsonschema:"Non-fatal backend warnings from the management API"`
}

// ListedAlertRule is a flattened alert rule for LLM consumption.
type ListedAlertRule struct {
	ID               string            `json:"id,omitempty" jsonschema:"Stable alert rule ID (openshift_io_alert_rule_id)"`
	Name             string            `json:"name" jsonschema:"Alert name"`
	Expr             string            `json:"expr,omitempty" jsonschema:"PromQL expression"`
	Severity         string            `json:"severity,omitempty" jsonschema:"severity label"`
	Namespace        string            `json:"namespace,omitempty" jsonschema:"namespace label"`
	Cluster          string            `json:"cluster,omitempty" jsonschema:"cluster or managed_cluster label when present"`
	Source           string            `json:"source,omitempty" jsonschema:"platform or user"`
	ManagementStatus string            `json:"management_status" jsonschema:"user-created, gitops, or operator"`
	Group            string            `json:"group,omitempty" jsonschema:"Prometheus rule group name"`
	Type             string            `json:"type,omitempty" jsonschema:"alerting or recording"`
	Health           string            `json:"health,omitempty" jsonschema:"Prometheus rule health"`
	Labels           map[string]string `json:"labels,omitempty" jsonschema:"Rule labels"`
	Annotations      map[string]string `json:"annotations,omitempty" jsonschema:"Rule annotations"`
}

// CreateAlertRuleOutput is the structured result of create_alert_rule.
type CreateAlertRuleOutput struct {
	ID string `json:"id" jsonschema:"Computed stable ID for the created alert rule"`
}

// RuleMutationResult is a per-rule result for bulk update or delete.
type RuleMutationResult struct {
	ID         string `json:"id" jsonschema:"Stable alert rule ID that was processed"`
	StatusCode int    `json:"statusCode" jsonschema:"HTTP status code for this rule"`
	Message    string `json:"message,omitempty" jsonschema:"Error message if the operation failed"`
}

// UpdateAlertRuleOutput is the structured result of update_alert_rule.
type UpdateAlertRuleOutput struct {
	Rules []RuleMutationResult `json:"rules" jsonschema:"Per-rule update results"`
}

// DeleteAlertRulesOutput is the structured result of delete_alert_rules.
type DeleteAlertRulesOutput struct {
	Rules []RuleMutationResult `json:"rules" jsonschema:"Per-rule deletion results"`
}

// PreviewAlertRuleOutput is the structured result of preview_alert_rule.
type PreviewAlertRuleOutput struct {
	Writable    bool             `json:"writable" jsonschema:"Whether the management API can persist this change"`
	ManagedBy   string           `json:"managedBy,omitempty" jsonschema:"gitops or operator when the target is not writable"`
	DesiredRule map[string]any   `json:"desiredRule,omitempty" jsonschema:"Desired alerting rule after the planned change"`
	Resources   []map[string]any `json:"resources,omitempty" jsonschema:"Kubernetes resources that would be created or modified"`
}

type prometheusRulesResponse struct {
	Data struct {
		Groups []prometheusRuleGroup `json:"groups"`
	} `json:"data"`
	Warnings []string `json:"warnings,omitempty"`
}

type prometheusRuleGroup struct {
	Name  string           `json:"name"`
	Rules []prometheusRule `json:"rules"`
}

type prometheusRule struct {
	Name        string            `json:"name,omitempty"`
	Query       string            `json:"query,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Health      string            `json:"health,omitempty"`
	Type        string            `json:"type,omitempty"`
}

type createAPIRequest struct {
	AlertingRule   *alertRuleSpecAPI     `json:"alertingRule,omitempty"`
	PrometheusRule *prometheusRuleTarget `json:"prometheusRule,omitempty"`
}

type createAPIResponse struct {
	ID string `json:"id"`
}

type alertRuleSpecAPI struct {
	Alert       *string           `json:"alert,omitempty"`
	Expr        *string           `json:"expr,omitempty"`
	For         *string           `json:"for,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type prometheusRuleTarget struct {
	PrometheusRuleName      string  `json:"prometheusRuleName"`
	PrometheusRuleNamespace string  `json:"prometheusRuleNamespace"`
	GroupName               *string `json:"groupName,omitempty"`
}

type errorAPIResponse struct {
	Error string `json:"error"`
}

type bulkMutationAPIResponse struct {
	Rules []ruleMutationAPIResult `json:"rules"`
}

type ruleMutationAPIResult struct {
	ID         string  `json:"id"`
	Message    *string `json:"message,omitempty"`
	StatusCode int     `json:"statusCode"`
}
