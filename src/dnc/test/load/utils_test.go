package load

import (
	"context"
	"dnc/configuration"
	"dnc/test/k8s"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
)

type podPortForwardOpts struct {
	namespace       string
	labelSelector   string
	localPort       int
	destinationPort int
}

func getPortForwardHandle(ctx context.Context, t *testing.T, pf *k8s.PortForwarder, pfOpts podPortForwardOpts) (k8s.PortForwardStreamHandle, error) {
	var pfh k8s.PortForwardStreamHandle
	pfFunc := func() error {
		handle, err := pf.Forward(ctx, pfOpts.namespace, pfOpts.labelSelector, pfOpts.localPort, pfOpts.destinationPort)
		if err != nil {
			t.Logf("could not port forward: %v. retrying...", err)
			return err
		}
		pfh = handle
		return nil
	}

	if err := defaultRetrier.Do(ctx, pfFunc); err != nil {
		return k8s.PortForwardStreamHandle{}, err
	}

	t.Logf("port forward successful: %s", pfh.Url())
	return pfh, nil
}

func dncContainerSpec(image string, port int) v1.Container {
	return v1.Container{
		Name:            "dnc",
		Image:           image,
		ImagePullPolicy: v1.PullAlways,
		VolumeMounts: []v1.VolumeMount{
			{Name: "log", MountPath: "/var/log"},
			{Name: "dnc-config", MountPath: "/etc/dnc"},
		},
		Args: []string{"-stderr"},
		Ports: []v1.ContainerPort{
			{ContainerPort: int32(port)},
		},
		Env: []v1.EnvVar{
			{
				Name:  "DNC_CONFIGURATION_PATH",
				Value: "/etc/dnc/dnc_config.json",
			},
		},
	}
}

func dncConfigMap(cfg configuration.Configuration) (v1.ConfigMap, error) {
	dncCfgBs, err := json.Marshal(cfg)
	if err != nil {
		return v1.ConfigMap{}, err
	}

	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "dnc-config"},
		Data:       map[string]string{"dnc_config.json": string(dncCfgBs)},
	}, nil
}

func mockNmagentContainerSpec(image string, port int) v1.Container {
	address := fmt.Sprintf("localhost:%d", port)

	return v1.Container{
		Name:            "mocknmagent",
		Image:           image,
		ImagePullPolicy: v1.PullAlways,
		Args:            []string{"--address", address},
		Ports: []v1.ContainerPort{
			{ContainerPort: int32(port)},
		},
	}
}

func getBaseDNCConfig(path string) (configuration.Configuration, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return configuration.Configuration{}, err
	}

	var cfg configuration.Configuration
	if err := json.Unmarshal(bs, &cfg); err != nil {
		return configuration.Configuration{}, err
	}

	configuration.SetDncConfigDefaults(&cfg)
	return cfg, nil
}
