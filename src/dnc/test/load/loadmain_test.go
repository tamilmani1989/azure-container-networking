//+build loadtest

package load

import (
	"dnc/test/junit"
	"dnc/test/k8s"
	"flag"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	loadTestNamespace             = "loadtest"
	baseDNCDeploymentYAMLFilePath = "testdata/base_dnc_deployment.yaml"
	baseDNCConfigJSONFilePath     = "testdata/base_dnc_config.json"
	dncImageEnvVar                = "DNC_IMAGE"
	mockNMAgentImageEnvVar        = "MOCK_NMAGENT_IMAGE"
	cosmosAccountNameEnvVar       = "COSMOS_ACCOUNT_NAME"
	cosmosAccountKeyEnvVar        = "COSMOS_ACCOUNT_KEY"
	junitXMLReportPath            = "report.xml"
)

var (
	restConfig        *rest.Config
	clientset         *kubernetes.Clientset
	k8sShim           *k8s.Shim
	dncImage          string
	mockNMAgentImage  string
	cosmosAccountName string
	cosmosAccountKey  string
	suites            junit.TestSuites
)

func TestMain(m *testing.M) {
	// importing controller-runtime/pkg/client/config has the side effect of a --kubeconfig flag registered
	// which works with config.GetConfig.
	flag.Parse()

	var err error
	if restConfig, err = config.GetConfig(); err != nil {
		logrus.Fatalf("could not get k8s rest config: %v", err)
	}

	logrus.Infof("k8s rest config for apiserver %s", restConfig.Host)

	if clientset, err = kubernetes.NewForConfig(restConfig); err != nil {
		logrus.Fatalf("could not get k8s clientset: %v", err)
	}

	k8sShim = k8s.NewShim(clientset)

	dncImage = mustExtractEnvVar(dncImageEnvVar)
	logrus.Infof("%v=%v", dncImageEnvVar, dncImage)

	mockNMAgentImage = mustExtractEnvVar(mockNMAgentImageEnvVar)
	logrus.Infof("%v=%v", mockNMAgentImageEnvVar, mockNMAgentImage)

	cosmosAccountName = mustExtractEnvVar(cosmosAccountNameEnvVar)
	logrus.Infof("%v=%v", cosmosAccountNameEnvVar, cosmosAccountName)

	cosmosAccountKey = mustExtractEnvVar(cosmosAccountKeyEnvVar)

	osExitCode := m.Run()

	writeReport(junitXMLReportPath)

	os.Exit(osExitCode)
}

func writeReport(path string) {
	f, err := os.Create(path)
	if err != nil {
		logrus.Errorf("could not create/truncate file %s: %v", path, err)
		return
	}
	defer f.Close()

	if err := junit.WriteReport(suites, f); err != nil {
		logrus.Errorf("could not write report to %s: %v", path, err)
		return
	}

	logrus.Infof("report successfully written to %s", path)
}

func mustExtractEnvVar(envVar string) string {
	s := os.Getenv(envVar)
	if s == "" {
		logrus.Fatalf("%v env var not set", envVar)
	}
	return s
}
