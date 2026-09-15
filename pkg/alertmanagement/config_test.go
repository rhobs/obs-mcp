package alertmanagement

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rhobs/obs-mcp/pkg/auth"
)

func TestConfigValidateAndAuthMode(t *testing.T) {
	require.NoError(t, (&Config{}).Validate())
	require.EqualError(t, (&Config{AuthMode: "oauth"}).Validate(),
		`invalid auth_mode: "oauth" (valid options: "header", "kubeconfig")`)
	require.Equal(t, auth.AuthModeHeader, (&Config{}).GetAuthMode())
	require.Equal(t, auth.AuthModeKubeConfig, (&Config{AuthMode: auth.AuthModeKubeConfig}).GetAuthMode())
}
