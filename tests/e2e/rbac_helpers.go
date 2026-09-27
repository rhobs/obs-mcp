//go:build e2e && !openshift

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	e2eAlertmanagerWriteRole = "obs-mcp-e2e-alertmanagers-api"
	e2ePlatformDropRole      = "obs-mcp-e2e-alertrelabelconfigs"
)

func kubeClientset(t *testing.T) *kubernetes.Clientset {
	t.Helper()
	cfg, err := getKubeConfig()
	require.NoError(t, err)
	cs, err := kubernetes.NewForConfig(cfg)
	require.NoError(t, err)
	return cs
}

func namespaceExists(t *testing.T, cs *kubernetes.Clientset, namespace string) bool {
	t.Helper()
	_, err := cs.CoreV1().Namespaces().Get(t.Context(), namespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false
	}
	skipIfInsufficientRBAC(t, err, "get namespace "+namespace)
	require.NoError(t, err)
	return true
}

func skipIfInsufficientRBAC(t *testing.T, err error, action string) {
	t.Helper()
	if err == nil {
		return
	}
	if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
		t.Skipf("%s: %v", action, err)
	}
}

func ensureNamespacedRole(t *testing.T, namespace, name string, rules []rbacv1.PolicyRule) {
	t.Helper()
	cs := kubeClientset(t)
	if !namespaceExists(t, cs, namespace) {
		t.Skipf("namespace %s not found", namespace)
	}

	ctx := t.Context()
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Rules:      rules,
	}
	if _, err := cs.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		skipIfInsufficientRBAC(t, err, "create Role "+namespace+"/"+name)
		t.Fatalf("create Role %s/%s: %v", namespace, name, err)
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      testConfig.ServiceAccountName,
			Namespace: testConfig.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
	}
	if _, err := cs.RbacV1().RoleBindings(namespace).Create(ctx, rb, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		skipIfInsufficientRBAC(t, err, "create RoleBinding "+namespace+"/"+name)
		t.Fatalf("create RoleBinding %s/%s: %v", namespace, name, err)
	}

	t.Cleanup(func() {
		delCtx := context.Background()
		if err := cs.RbacV1().RoleBindings(namespace).Delete(delCtx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("cleanup RoleBinding %s/%s: %v", namespace, name, err)
		}
		if err := cs.RbacV1().Roles(namespace).Delete(delCtx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("cleanup Role %s/%s: %v", namespace, name, err)
		}
	})
}

func ensureAlertmanagerWriteRBAC(t *testing.T) {
	t.Helper()
	cs := kubeClientset(t)
	namespace := ""
	for _, candidate := range []string{"openshift-monitoring", "monitoring"} {
		if namespaceExists(t, cs, candidate) {
			namespace = candidate
			break
		}
	}
	if namespace == "" {
		t.Skip("no monitoring namespace for Alertmanager write RBAC")
	}
	ensureNamespacedRole(t, namespace, e2eAlertmanagerWriteRole, []rbacv1.PolicyRule{{
		APIGroups:     []string{"monitoring.coreos.com"},
		Resources:     []string{"alertmanagers/api"},
		ResourceNames: []string{"main"},
		Verbs:         []string{"get", "list", "create", "update", "patch", "delete"},
	}})
}

func ensurePlatformDropRBAC(t *testing.T) {
	t.Helper()
	ensureNamespacedRole(t, "openshift-monitoring", e2ePlatformDropRole, []rbacv1.PolicyRule{{
		APIGroups: []string{"monitoring.openshift.io"},
		Resources: []string{"alertrelabelconfigs"},
		Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
	}})
}
