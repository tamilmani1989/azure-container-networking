// Copyright 2020 Microsoft. All rights reserved.

package filesystem

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"

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

func (*kubeconfigClusterAuthConfig) RefreshForever(context.Context) {
	// TODO: should we re-read the file each time we refresh? it probably won't change since it is generally a
	// k8s Secret mounted to the filesystem, but probably wouldn't hurt.

	// For now, do nothing and just return
}
