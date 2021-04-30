package k8s

import (
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClientset builds a kubernetes.Clientset from the given kubeconfig file path.
func GetClientset(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// ParseResource unmarshals a k8s resource from a yaml or json file into the provided interface (must be pointer).
func ParseResource(path string, out interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return yaml.NewYAMLOrJSONDecoder(f, 0).Decode(out)
}

// GetRestConfig builds a rest.Config from the given kubeconfig path.
func GetRestConfig(kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}
