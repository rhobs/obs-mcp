package k8s

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClientCmdConfig returns a ClientConfig that can produce both a REST config
// and raw kubeconfig data, using the default loading rules.
func GetClientCmdConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
}

// GetClientConfig returns a Kubernetes REST config using kubeconfig.
func GetClientConfig() (*rest.Config, error) {
	config, err := GetClientCmdConfig().ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	return config, nil
}
