package requestcontroller

import "testing"

func TestConfigApplyDefaults(t *testing.T) {
	config := Config{
		KubernetesConfig: KubernetesConfig{
			KubeconfigPath: "test",
		},
	}
	config.ApplyDefaults()

	want := "test"
	got := config.KubernetesConfig.KubeconfigPath

	if want != got {
		t.Errorf("wanted %s, got %s", want, got)
	}

	config = Config{}
	config.ApplyDefaults()

	want = defaultConfigKubeconfigPath
	got = config.KubernetesConfig.KubeconfigPath

	if want != got {
		t.Errorf("wanted %s, got %s", want, got)
	}
}
