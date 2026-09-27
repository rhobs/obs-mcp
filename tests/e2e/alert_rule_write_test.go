//go:build e2e && !openshift

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	e2eAlertRuleName      = "ObsMCPE2EWidgetDown"
	e2ePrometheusRuleName = "obs-mcp-e2e-user-alerts"
	e2eAlertRuleExpr      = `absent(obs_mcp_e2e_canary)`
)

func TestAlertRuleCreateUpdateDelete(t *testing.T) {
	skipIfMissingTool(t, "create_alert_rule")
	skipIfMissingTool(t, "update_alert_rule")
	skipIfMissingTool(t, "delete_alert_rules")
	probeAlertManagement(t)

	ns := ensureUserAlertRuleNamespace(t)

	preview := callAlertManagement(t, 230, "preview_alert_rule", map[string]any{
		"alert":                e2eAlertRuleName,
		"expr":                 e2eAlertRuleExpr,
		"severity":             "warning",
		"namespace":            ns,
		"prometheus_rule_name": e2ePrometheusRuleName,
	})
	if writable, _ := preview["writable"].(bool); !writable {
		t.Skipf("preview_alert_rule reported writable=false: %v", preview)
	}

	created := createAlertRuleWithRBACRetry(t, ns)
	ruleID, ok := created["id"].(string)
	require.True(t, ok, "expected id string")
	require.NotEmpty(t, ruleID)

	t.Cleanup(func() {
		cleanupToolCall(t, 234, "delete_alert_rules", map[string]any{
			"rule_id": ruleID,
		})
	})

	updated := callAlertManagement(t, 232, "update_alert_rule", map[string]any{
		"rule_id":  ruleID,
		"severity": "info",
		"labels": map[string]any{
			"obs_mcp_e2e": "true",
		},
	})
	ruleID = requireMutationSuccess(t, updated, "update")

	deleted := callAlertManagement(t, 233, "delete_alert_rules", map[string]any{
		"rule_id": ruleID,
	})
	_ = requireMutationSuccess(t, deleted, "delete")
}

func TestDisableRestorePlatformAlertRule(t *testing.T) {
	skipIfMissingTool(t, "update_alert_rule")
	skipIfMissingTool(t, "preview_alert_rule")
	probeAlertManagement(t)
	ensurePlatformDropRBAC(t)

	ruleID, name := pickPlatformAlertRule(t)
	t.Logf("dropping platform rule %s (%s)", name, ruleID)

	preview := callAlertManagement(t, 240, "preview_alert_rule", map[string]any{
		"rule_id":               ruleID,
		"alerting_rule_enabled": false,
	})
	if writable, _ := preview["writable"].(bool); !writable {
		t.Skipf("preview_alert_rule reported writable=false for %s (%s): %v", name, ruleID, preview)
	}

	dropped := updateAlertRuleEnabledWithRBACRetry(t, 241, ruleID, false)
	_ = requireMutationSuccess(t, dropped, "disable")
	t.Cleanup(func() {
		cleanupRestore(t, 245, "update_alert_rule", map[string]any{
			"rule_id":               ruleID,
			"alerting_rule_enabled": true,
		})
	})

	restored := updateAlertRuleEnabledWithRBACRetry(t, 242, ruleID, true)
	_ = requireMutationSuccess(t, restored, "restore")
}

func probeAlertManagement(t *testing.T) {
	t.Helper()
	skipIfMissingTool(t, "list_alert_rules")
	resp, err := mcpClient.CallTool(t, 229, "list_alert_rules", map[string]any{})
	require.NoError(t, err)
	if errText := mcpResultErrorText(resp); errText != "" {
		skipIfAlertManagementBackendUnavailable(t, errText)
		t.Fatalf("list_alert_rules probe failed: %s", errText)
	}
}

func ensureUserAlertRuleNamespace(t *testing.T) string {
	t.Helper()
	ctx := t.Context()
	cs := kubeClientset(t)
	ns, err := cs.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "obs-mcp-e2e-ar-"},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	nsName := ns.Name

	_, err = cs.RbacV1().Roles(nsName).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "obs-mcp-e2e-prometheusrules",
			Namespace: nsName,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"monitoring.coreos.com"},
			Resources: []string{"prometheusrules"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
		}},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = cs.RbacV1().RoleBindings(nsName).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "obs-mcp-e2e-prometheusrules",
			Namespace: nsName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      testConfig.ServiceAccountName,
			Namespace: testConfig.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "obs-mcp-e2e-prometheusrules",
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := cs.CoreV1().Namespaces().Delete(context.Background(), nsName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("cleanup namespace %s: %v", nsName, err)
		}
	})
	return nsName
}

func callAlertManagement(t *testing.T, id int, name string, args map[string]any) map[string]any {
	t.Helper()
	content, errText := callAlertManagementMaybeError(t, id, name, args)
	if errText != "" {
		t.Fatalf("%s failed: %s", name, errText)
	}
	return content
}

func callAlertManagementMaybeError(t *testing.T, id int, name string, args map[string]any) (map[string]any, string) {
	t.Helper()
	skipIfMissingTool(t, name)
	resp, err := mcpClient.CallTool(t, id, name, args)
	require.NoError(t, err)
	if errText := mcpResultErrorText(resp); errText != "" {
		skipIfAlertManagementBackendUnavailable(t, errText)
		return nil, errText
	}
	return structuredContent(t, resp), ""
}

func pickPlatformAlertRule(t *testing.T) (id, name string) {
	t.Helper()
	content := callAlertManagement(t, 239, "list_alert_rules", map[string]any{
		"source": "platform",
	})
	rules, ok := content["rules"].([]any)
	require.True(t, ok, "expected rules array")
	var fallbackID, fallbackName string
	for _, raw := range rules {
		rule, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ruleID, _ := rule["id"].(string)
		ruleName, _ := rule["name"].(string)
		if ruleID == "" {
			continue
		}
		if ruleName != "Watchdog" {
			return ruleID, ruleName
		}
		fallbackID, fallbackName = ruleID, ruleName
	}
	if fallbackID == "" {
		t.Skip("no platform alert rule with a stable id")
	}
	return fallbackID, fallbackName
}

func updateAlertRuleEnabledWithRBACRetry(t *testing.T, callID int, ruleID string, enabled bool) map[string]any {
	t.Helper()
	args := map[string]any{
		"rule_id":               ruleID,
		"alerting_rule_enabled": enabled,
	}
	return retryWhileForbidden(t, "update_alert_rule alerting_rule_enabled", func() (map[string]any, string) {
		return callAlertManagementMaybeError(t, callID, "update_alert_rule", args)
	})
}

func createAlertRuleWithRBACRetry(t *testing.T, namespace string) map[string]any {
	t.Helper()
	args := map[string]any{
		"alert":                e2eAlertRuleName,
		"expr":                 e2eAlertRuleExpr,
		"severity":             "warning",
		"namespace":            namespace,
		"prometheus_rule_name": e2ePrometheusRuleName,
	}
	return retryWhileForbidden(t, "create_alert_rule", func() (map[string]any, string) {
		return callAlertManagementMaybeError(t, 231, "create_alert_rule", args)
	})
}

func requireMutationSuccess(t *testing.T, content map[string]any, op string) string {
	t.Helper()
	rules, ok := content["rules"].([]any)
	require.True(t, ok, "expected rules array from %s", op)
	require.NotEmpty(t, rules, "expected at least one %s result", op)
	first, ok := rules[0].(map[string]any)
	require.True(t, ok, "expected %s result object", op)
	status, _ := first["statusCode"].(float64)
	require.True(t, status == 200 || status == 204,
		"%s statusCode=%v message=%v", op, status, first["message"])
	id, _ := first["id"].(string)
	require.NotEmpty(t, id, "expected id from %s", op)
	return id
}
