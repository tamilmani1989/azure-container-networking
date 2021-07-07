// Copyright 2020 Microsoft. All rights reserved.

package filesystem

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"dnc/requestcontroller"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubeClusterAuthProvider struct {
	KubeconfigFile string
	OrchestratorID string
}

func (k KubeClusterAuthProvider) GetAllClusterAuthConfigs(ctx context.Context) (map[string]requestcontroller.ClusterAuthConfig, error) {
	kubeconfigBytes, err := ioutil.ReadFile(k.KubeconfigFile)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("could not build REST config from kubeconfig file: %w", err)
	}

	return map[string]requestcontroller.ClusterAuthConfig{
		k.OrchestratorID: &kubeconfigClusterAuthConfig{
			restConfig: restConfig,
		},
	}, nil
}

type kubeconfigClusterAuthConfig struct {
	restConfig *rest.Config
	mutex      sync.RWMutex
}

func (k *kubeconfigClusterAuthConfig) GetConfig() *rest.Config {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	return k.restConfig
}

func (*kubeconfigClusterAuthConfig) Refresh(context.Context) error {
	// rest.Config is already initialized
	return nil
}

func (*kubeconfigClusterAuthConfig) Expires() time.Time {
	// TODO: Should we expire these on purpose to trigger a re-read from disk?
	// For now, return 1 year, which should be long enough between RC deployments to never
	// actually trigger a refresh
	return time.Now().Add(8760 * time.Hour) // 1 year
}
