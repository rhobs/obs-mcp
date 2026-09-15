package alertmanagement

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"

	"github.com/rhobs/obs-mcp/pkg/auth"
)

type mockKubernetesClient struct {
	api.KubernetesClient
	restConfig *rest.Config
}

func (m *mockKubernetesClient) RESTConfig() *rest.Config {
	return m.restConfig
}

type mockBaseConfig struct {
	api.BaseConfig
	config *Config
}

func (m *mockBaseConfig) GetToolsetConfig(name string) (api.ExtendedConfig, bool) {
	if name == ToolsetName && m.config != nil {
		return m.config, true
	}
	return nil, false
}

type mockToolCallRequest struct {
	arguments map[string]any
}

func (m *mockToolCallRequest) GetArguments() map[string]any {
	return m.arguments
}

func handlerParams(t *testing.T, cfg *Config, args map[string]any) api.ToolHandlerParams {
	t.Helper()
	if cfg.AuthMode == "" {
		cfg.AuthMode = auth.AuthModeKubeConfig
	}
	return api.ToolHandlerParams{
		Context: t.Context(),
		KubernetesClient: &mockKubernetesClient{
			restConfig: &rest.Config{BearerToken: "test-token"},
		},
		BaseConfig:      &mockBaseConfig{config: cfg},
		ToolCallRequest: &mockToolCallRequest{arguments: args},
	}
}

type capturedRequest struct {
	method string
	path   string
	query  string
	auth   string
	body   string
}

func managementServer(t *testing.T, status int, body string, captured *capturedRequest) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if captured != nil {
			captured.method = r.Method
			captured.path = r.URL.Path
			captured.query = r.URL.RawQuery
			captured.auth = r.Header.Get("Authorization")
			raw, _ := io.ReadAll(r.Body)
			captured.body = string(raw)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListAlertRulesHandler_Success(t *testing.T) {
	var captured capturedRequest
	srv := managementServer(t, http.StatusOK, `{
		"data":{"groups":[{"name":"general","rules":[
			{"name":"Watchdog","query":"vector(1)","type":"alerting","labels":{
				"severity":"none","namespace":"openshift-monitoring",
				"openshift_io_alert_rule_id":"id-1",
				"openshift_io_alert_source":"platform",
				"openshift_io_rule_managed_by":"operator"
			}}
		]}]},
		"warnings":["thanos tenancy unavailable"]
	}`, &captured)

	result, err := listAlertRulesHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"namespace": "openshift-monitoring",
		"severity":  "none",
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, http.MethodGet, captured.method)
	require.Equal(t, rulesAPIPath, captured.path)
	require.Contains(t, captured.query, "namespace=openshift-monitoring")
	require.Equal(t, "Bearer test-token", captured.auth)

	out := result.StructuredContent.(ListAlertRulesOutput)
	require.Len(t, out.Rules, 1)
	require.Equal(t, "id-1", out.Rules[0].ID)
	require.Equal(t, "operator", out.Rules[0].ManagementStatus)
	require.Equal(t, []string{"thanos tenancy unavailable"}, out.Warnings)
}

func TestListAlertRulesHandler_HeaderAuthForwardsToken(t *testing.T) {
	var captured capturedRequest
	srv := managementServer(t, http.StatusOK, `{"data":{"groups":[]}}`, &captured)
	params := handlerParams(t, &Config{
		ManagementAPIURL: srv.URL,
		AuthMode:         auth.AuthModeHeader,
	}, map[string]any{})
	params.Context = context.WithValue(params.Context, kubernetes.OAuthAuthorizationHeader, "Bearer header-token")

	result, err := listAlertRulesHandler(params)
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, "Bearer header-token", captured.auth)
}

func TestListAlertRulesHandler_SourceFilter(t *testing.T) {
	var captured capturedRequest
	srv := managementServer(t, http.StatusOK, `{"data":{"groups":[]}}`, &captured)
	result, err := listAlertRulesHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"source": "user",
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Contains(t, captured.query, alertSourceLabel+"=user")
}

func TestListAlertRulesHandler_ClusterFilter(t *testing.T) {
	var captured capturedRequest
	srv := managementServer(t, http.StatusOK, `{"data":{"groups":[]}}`, &captured)
	result, err := listAlertRulesHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"cluster": "cluster-a",
		"cluster_labels": map[string]any{
			"env": "prod",
		},
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Contains(t, captured.query, clusterQueryKey+"=cluster-a")
	require.Contains(t, captured.query, clusterLabelsQueryKey+"=env%3Dprod")
}

func TestListAlertRulesHandler_NotFound(t *testing.T) {
	srv := managementServer(t, http.StatusNotFound, `{"error":"not found"}`, nil)
	result, err := listAlertRulesHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{}))
	require.NoError(t, err)
	require.EqualError(t, result.Error, "management API returned HTTP 404: not found")
}

func TestCreateAlertRuleHandler_Success(t *testing.T) {
	var captured capturedRequest
	srv := managementServer(t, http.StatusCreated, `{"id":"created-1"}`, &captured)

	result, err := createAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"alert":                "HighErrorRate",
		"expr":                 `rate(errors[5m]) > 0.05`,
		"severity":             "critical",
		"namespace":            "app",
		"prometheus_rule_name": "user-alerts",
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, http.MethodPost, captured.method)
	require.Contains(t, captured.body, `"alert":"HighErrorRate"`)
	require.Contains(t, captured.body, `"prometheusRuleName":"user-alerts"`)
	out := result.StructuredContent.(CreateAlertRuleOutput)
	require.Equal(t, "created-1", out.ID)
}

func TestCreateAlertRuleHandler_Conflict(t *testing.T) {
	srv := managementServer(t, http.StatusConflict, `{"error":"rule already exists"}`, nil)
	result, err := createAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"alert": "HighErrorRate",
		"expr":  "vector(1)",
	}))
	require.NoError(t, err)
	require.EqualError(t, result.Error, "management API returned HTTP 409: rule already exists")
}

func TestCreateAlertRuleHandler_Validation(t *testing.T) {
	result, err := createAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: "http://example"}, map[string]any{
		"alert": "MissingExpr",
	}))
	require.NoError(t, err)
	require.Error(t, result.Error)
	require.EqualError(t, result.Error, "expr is required")
}

func TestCreateAlertRuleHandler_RejectsCluster(t *testing.T) {
	result, err := createAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: "http://example"}, map[string]any{
		"alert":   "A",
		"expr":    "up",
		"cluster": "cluster-a",
	}))
	require.NoError(t, err)
	require.EqualError(t, result.Error, errClusterWriteUnsupported)
}

func TestUpdateAlertRuleHandler_ClassificationAndLabels(t *testing.T) {
	var captured capturedRequest
	srv := managementServer(t, http.StatusOK, `{"rules":[{"id":"id-1","statusCode":204}]}`, &captured)
	result, err := updateAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"rule_id":                  "id-1",
		"severity":                 "warning",
		"labels_to_remove":         []any{"noise"},
		"classification_component": "monitoring",
		"classification_layer":     nil,
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, http.MethodPatch, captured.method)
	require.Contains(t, captured.body, `"ruleIds":["id-1"]`)
	require.Contains(t, captured.body, `"severity":"warning"`)
	require.Contains(t, captured.body, `"noise":null`)
	require.Contains(t, captured.body, `"openshift_io_alert_rule_component":"monitoring"`)
	require.Contains(t, captured.body, `"openshift_io_alert_rule_layer":null`)
}

func TestUpdateAlertRuleHandler_Drop(t *testing.T) {
	var captured capturedRequest
	srv := managementServer(t, http.StatusOK, `{"rules":[{"id":"id-1","statusCode":204}]}`, &captured)
	result, err := updateAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"rule_id":               "id-1",
		"alerting_rule_enabled": false,
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Contains(t, captured.body, `"alertingRuleEnabled":false`)
	require.NotContains(t, captured.body, `"labels"`)
}

func TestUpdateAlertRuleHandler_BadRequest(t *testing.T) {
	srv := managementServer(t, http.StatusBadRequest, `{"error":"invalid PromQL"}`, nil)
	result, err := updateAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"rule_id":  "id-1",
		"severity": "warning",
	}))
	require.NoError(t, err)
	require.EqualError(t, result.Error, "management API returned HTTP 400: invalid PromQL")
}

func TestUpdateAlertRuleHandler_Forbidden(t *testing.T) {
	srv := managementServer(t, http.StatusForbidden, `{"error":"insufficient permissions"}`, nil)
	result, err := updateAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"rule_id":  "id-1",
		"severity": "warning",
	}))
	require.NoError(t, err)
	require.Error(t, result.Error)
	require.EqualError(t, result.Error, "management API returned HTTP 403: insufficient permissions")
}

func TestUpdateAlertRuleHandler_GitOpsConflict(t *testing.T) {
	srv := managementServer(t, http.StatusMethodNotAllowed, `{"error":"rule is GitOps-managed and cannot be modified"}`, nil)
	result, err := updateAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"rule_id":  "id-1",
		"severity": "warning",
	}))
	require.NoError(t, err)
	require.Error(t, result.Error)
	require.Contains(t, result.Error.Error(), "GitOps-managed")
}

func TestUpdateAlertRuleHandler_PartialSuccess(t *testing.T) {
	srv := managementServer(t, http.StatusOK, `{
		"rules":[
			{"id":"ok","statusCode":204},
			{"id":"denied","statusCode":403,"message":"insufficient permissions"}
		]
	}`, nil)
	result, err := updateAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"rule_ids": []any{"ok", "denied"},
		"severity": "warning",
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	out := result.StructuredContent.(UpdateAlertRuleOutput)
	require.Len(t, out.Rules, 2)
	require.Equal(t, 204, out.Rules[0].StatusCode)
	require.Equal(t, 403, out.Rules[1].StatusCode)
	require.Equal(t, "insufficient permissions", out.Rules[1].Message)
}

func TestDeleteAlertRulesHandler_Success(t *testing.T) {
	var captured capturedRequest
	srv := managementServer(t, http.StatusOK, `{"rules":[{"id":"id-1","statusCode":204}]}`, &captured)
	result, err := deleteAlertRulesHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"rule_id": "id-1",
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, http.MethodDelete, captured.method)
	require.Contains(t, captured.body, `"ruleIds":["id-1"]`)
}

func TestDeleteAlertRulesHandler_TooManyIDs(t *testing.T) {
	ids := make([]any, maxBulkRuleIDs+1)
	for i := range ids {
		ids[i] = fmt.Sprintf("id-%d", i)
	}
	result, err := deleteAlertRulesHandler(handlerParams(t, &Config{ManagementAPIURL: "http://example"}, map[string]any{
		"rule_ids": ids,
	}))
	require.NoError(t, err)
	require.EqualError(t, result.Error, "rule_ids exceeds maximum of 100")
}

func TestPreviewAlertRuleHandler_Update(t *testing.T) {
	var captured capturedRequest
	srv := managementServer(t, http.StatusOK, `{"writable":true,"desiredRule":{"alert":"Watchdog"}}`, &captured)
	result, err := previewAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"rule_id":  "id-1",
		"severity": "warning",
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, http.MethodPost, captured.method)
	require.Equal(t, previewAPIPath, captured.path)
	require.Contains(t, captured.body, `"ruleId":"id-1"`)
	require.Contains(t, captured.body, `"severity":"warning"`)
	out := result.StructuredContent.(PreviewAlertRuleOutput)
	require.True(t, out.Writable)
}

func TestPreviewAlertRuleHandler_RejectsBulkIDs(t *testing.T) {
	result, err := previewAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: "http://example"}, map[string]any{
		"rule_ids": []any{"id-1", "id-2"},
		"severity": "warning",
	}))
	require.NoError(t, err)
	require.EqualError(t, result.Error, "preview_alert_rule accepts a single rule_id, not rule_ids")
}

func TestPreviewAlertRuleHandler_Create(t *testing.T) {
	srv := managementServer(t, http.StatusOK, `{
		"writable":false,
		"managedBy":"gitops",
		"desiredRule":{"alert":"HighErrorRate"},
		"resources":[{"resource":{"kind":"PrometheusRule"}}]
	}`, nil)
	result, err := previewAlertRuleHandler(handlerParams(t, &Config{ManagementAPIURL: srv.URL}, map[string]any{
		"alert": "HighErrorRate",
		"expr":  "vector(1)",
	}))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	out := result.StructuredContent.(PreviewAlertRuleOutput)
	require.False(t, out.Writable)
	require.Equal(t, "gitops", out.ManagedBy)
}

func TestListAlertRulesHandler_MissingURL(t *testing.T) {
	result, err := listAlertRulesHandler(handlerParams(t, &Config{}, map[string]any{}))
	require.NoError(t, err)
	require.Error(t, result.Error)
	require.Contains(t, result.Error.Error(), "alert management API URL is not configured")
}
