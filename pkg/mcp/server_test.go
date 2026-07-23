package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/rhobs/obs-mcp/pkg/auth"
	"github.com/rhobs/obs-mcp/pkg/otelcol"
)

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		wantCtxValue   string
		wantNoCtxValue bool
	}{
		{
			name:         "bearer token is stored in context",
			authHeader:   "Bearer some-token",
			wantCtxValue: "Bearer some-token",
		},
		{
			name:           "empty header does not set context value",
			authHeader:     "",
			wantNoCtxValue: true,
		},
		{
			name:           "non-bearer header does not set context value",
			authHeader:     "some-raw-value",
			wantNoCtxValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCtxValue any
			var ctxValueExists bool

			inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				gotCtxValue = r.Context().Value(kubernetes.OAuthAuthorizationHeader)
				ctxValueExists = gotCtxValue != nil
			})

			handler := authMiddleware(inner)

			req := httptest.NewRequest("GET", "/test", http.NoBody)
			if tt.authHeader != "" {
				req.Header.Set(string(kubernetes.OAuthAuthorizationHeader), tt.authHeader)
			}

			handler.ServeHTTP(httptest.NewRecorder(), req)

			if tt.wantNoCtxValue {
				if ctxValueExists {
					t.Errorf("expected no context value, got %q", gotCtxValue)
				}
				return
			}

			if !ctxValueExists {
				t.Fatal("expected context value to be set, but it was nil")
			}
			if gotCtxValue != tt.wantCtxValue {
				t.Errorf("context value = %q, want %q", gotCtxValue, tt.wantCtxValue)
			}
		})
	}
}

func TestHeaderAuthRejectsUnauthenticatedToolCall(t *testing.T) {
	kubeClientConfig := clientcmd.NewDefaultClientConfig(clientcmdapi.Config{
		Clusters:       map[string]*clientcmdapi.Cluster{"test": {Server: "https://localhost", InsecureSkipTLSVerify: true}},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{"test": {Token: "kubeconfig-token"}},
		Contexts:       map[string]*clientcmdapi.Context{"test": {Cluster: "test", AuthInfo: "test"}},
		CurrentContext: "test",
	}, &clientcmd.ConfigOverrides{})

	tests := []struct {
		name      string
		authMode  auth.AuthMode
		expectErr string
	}{
		{
			name:      "header mode rejects unauthenticated call",
			authMode:  auth.AuthModeHeader,
			expectErr: `calling "tools/call": oauth token required`,
		},
		{
			name:     "kubeconfig mode allows unauthenticated call",
			authMode: auth.AuthModeKubeConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, err := NewMCPServer(ObsMCPOptions{
				Toolsets:               []string{otelcol.ToolsetName},
				AuthMode:               tt.authMode,
				Otelcol:                otelcol.NewDefaultConfig(),
				KubernetesClientConfig: kubeClientConfig,
			})
			require.NoError(t, err)

			clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
			_, err = mcpServer.Connect(context.Background(), serverTransport, nil)
			require.NoError(t, err)

			client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
			session, err := client.Connect(context.Background(), clientTransport, nil)
			require.NoError(t, err)
			defer session.Close()

			_, err = session.CallTool(context.Background(), &mcpsdk.CallToolParams{
				Name: "otelcol_list_components",
			})

			if tt.expectErr != "" {
				require.EqualError(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
