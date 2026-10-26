//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestEnsureNamespacedRoleCleanupOwnership(t *testing.T) {
	const namespace, name = "monitoring", "obs-mcp-test"
	tests := []struct {
		name          string
		roleExists    bool
		bindingExists bool
		bindingDenied bool
	}{
		{name: "both created"},
		{name: "both preexisting", roleExists: true, bindingExists: true},
		{name: "preexisting role", roleExists: true},
		{name: "preexisting binding", bindingExists: true},
		{name: "binding denied after role creation", bindingDenied: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []runtime.Object{&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}}
			if tt.roleExists {
				objects = append(objects, &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
			}
			if tt.bindingExists {
				objects = append(objects, &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
			}
			cs := fake.NewClientset(objects...)
			if tt.bindingDenied {
				cs.PrependReactor("create", "rolebindings", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "rbac.authorization.k8s.io", Resource: "rolebindings"}, name, nil)
				})
			}
			t.Run("grant", func(t *testing.T) {
				ensureNamespacedRoleWithClient(t, cs, namespace, name, nil, rbacv1.Subject{
					Kind: "ServiceAccount", Name: "obs-mcp", Namespace: "obs-mcp",
				})
				_, err := cs.RbacV1().Roles(namespace).Get(t.Context(), name, metav1.GetOptions{})
				require.NoError(t, err, "Role must exist before cleanup")
				_, err = cs.RbacV1().RoleBindings(namespace).Get(t.Context(), name, metav1.GetOptions{})
				require.NoError(t, err, "RoleBinding must exist before cleanup")
			})
			// Subtest cleanup has run: preexisting objects survive, new objects do not.
			_, err := cs.RbacV1().Roles(namespace).Get(t.Context(), name, metav1.GetOptions{})
			if tt.roleExists {
				require.NoError(t, err, "preexisting Role must survive cleanup")
			} else {
				require.True(t, apierrors.IsNotFound(err), "created Role must be cleaned up, got %v", err)
			}
			_, err = cs.RbacV1().RoleBindings(namespace).Get(t.Context(), name, metav1.GetOptions{})
			if tt.bindingExists {
				require.NoError(t, err, "preexisting RoleBinding must survive cleanup")
			} else {
				require.True(t, apierrors.IsNotFound(err), "created RoleBinding must be cleaned up, got %v", err)
			}
		})
	}
}
