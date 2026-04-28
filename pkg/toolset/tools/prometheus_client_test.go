package tools

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"k8s.io/client-go/rest"

	toolsetconfig "github.com/rhobs/obs-mcp/pkg/toolset/config"
)

type mockKubernetesClient struct {
	api.KubernetesClient
	restConfig *rest.Config
}

func (m *mockKubernetesClient) RESTConfig() *rest.Config {
	return m.restConfig
}

type mockConfigProvider struct {
	config *toolsetconfig.Config
}

func (m *mockConfigProvider) GetProviderConfig(string) (api.ExtendedConfig, bool) {
	return nil, false
}

func (m *mockConfigProvider) GetToolsetConfig(name string) (api.ExtendedConfig, bool) {
	if name == toolsetconfig.MetricsToolSetName && m.config != nil {
		return m.config, true
	}
	return nil, false
}

type mockToolCallRequest struct{}

func (m *mockToolCallRequest) GetArguments() map[string]any {
	return nil
}

func newTestParams(ctx context.Context, restConfig *rest.Config, cfg *toolsetconfig.Config) api.ToolHandlerParams {
	return api.ToolHandlerParams{
		Context:                ctx,
		KubernetesClient:       &mockKubernetesClient{restConfig: restConfig},
		ExtendedConfigProvider: &mockConfigProvider{config: cfg},
		ToolCallRequest:        &mockToolCallRequest{},
	}
}

func TestBuildAPIConfig(t *testing.T) {
	tests := []struct {
		name       string
		authMode   toolsetconfig.AuthMode
		ctxToken   string
		restConfig *rest.Config
		wantErr    bool
	}{
		{
			name:       "header auth mode with token in context",
			authMode:   toolsetconfig.AuthModeHeader,
			ctxToken:   "header-token",
			restConfig: &rest.Config{},
			wantErr:    false,
		},
		{
			name:       "header auth mode without token in context fails",
			authMode:   toolsetconfig.AuthModeHeader,
			ctxToken:   "",
			restConfig: &rest.Config{BearerToken: "kubeconfig-token"},
			wantErr:    true,
		},
		{
			name:       "kubeconfig auth mode uses REST config token",
			authMode:   toolsetconfig.AuthModeKubeConfig,
			ctxToken:   "",
			restConfig: &rest.Config{BearerToken: "kubeconfig-token"},
			wantErr:    false,
		},
		{
			name:       "kubeconfig auth mode ignores context token",
			authMode:   toolsetconfig.AuthModeKubeConfig,
			ctxToken:   "header-token",
			restConfig: &rest.Config{},
			wantErr:    false,
		},
		{
			name:       "nil REST config fails",
			authMode:   toolsetconfig.AuthModeHeader,
			ctxToken:   "header-token",
			restConfig: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			if tt.ctxToken != "" {
				ctx = context.WithValue(ctx, kubernetes.OAuthAuthorizationHeader, tt.ctxToken)
			}

			_, err := buildAPIConfig(
				newTestParams(ctx, tt.restConfig, nil),
				"http://localhost:9090",
				false,
				tt.authMode,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildAPIConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetConfig_DefaultAuthMode(t *testing.T) {
	params := newTestParams(context.Background(), &rest.Config{}, nil)
	cfg := getConfig(params)
	if cfg.GetAuthMode() != toolsetconfig.AuthModeHeader {
		t.Errorf("expected default auth mode %q, got %q", toolsetconfig.AuthModeHeader, cfg.GetAuthMode())
	}
}

func TestGetConfig_CustomAuthMode(t *testing.T) {
	cfg := &toolsetconfig.Config{AuthMode: toolsetconfig.AuthModeKubeConfig}
	params := newTestParams(context.Background(), &rest.Config{}, cfg)
	got := getConfig(params)
	if got.GetAuthMode() != toolsetconfig.AuthModeKubeConfig {
		t.Errorf("expected auth mode %q, got %q", toolsetconfig.AuthModeKubeConfig, got.GetAuthMode())
	}
}

func TestReadTokenFromCtx(t *testing.T) {
	tests := []struct {
		name     string
		ctxValue any
		want     string
	}{
		{
			name:     "Bearer prefix is stripped",
			ctxValue: "Bearer my-token-123",
			want:     "my-token-123",
		},
		{
			name:     "raw token without prefix returned as-is",
			ctxValue: "my-raw-token",
			want:     "my-raw-token",
		},
		{
			name:     "no value in context returns empty",
			ctxValue: nil,
			want:     "",
		},
		{
			name:     "non-string value in context returns empty",
			ctxValue: 12345,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			if tt.ctxValue != nil {
				ctx = context.WithValue(ctx, kubernetes.OAuthAuthorizationHeader, tt.ctxValue)
			}
			params := newTestParams(ctx, &rest.Config{}, nil)
			got := readTokenFromCtx(params)
			if got != tt.want {
				t.Errorf("readTokenFromCtx() = %q, want %q", got, tt.want)
			}
		})
	}
}
