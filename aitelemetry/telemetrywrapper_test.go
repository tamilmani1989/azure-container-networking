package aitelemetry

import (
	"os"
	"runtime"
	"testing"

	"github.com/Azure/azure-container-networking/platform"
)

var th TelemetryHandle

func TestMain(m *testing.M) {

	if runtime.GOOS == "linux" {
		platform.ExecuteCommand("cp metadata_test.json /tmp/azuremetadata.json")
	} else {
		platform.ExecuteCommand("copy metadata_test.json azuremetadata.json")
	}

	exitCode := m.Run()

	if runtime.GOOS == "linux" {
		platform.ExecuteCommand("rm /tmp/azuremetadata.json")
	} else {
		platform.ExecuteCommand("del azuremetadata.json")
	}

	os.Exit(exitCode)
}

func TestNewAITelemetry(t *testing.T) {
	aiConfig := AIConfig{
		AppName:                     "testapp",
		AppVersion:                  "v1.0.26",
		BatchSize:                   4096,
		BatchInterval:               2,
		EnableMetadataRefreshThread: false,
		RefreshTimeout:              10,
		EnableLogging:               true,
		EnableMetric:                true,
		EnableTrace:                 true,
	}
	th = NewAITelemetry("00ca2a73-c8d6-4929-a0c2-cf84545ec225", aiConfig)
	if th == nil {
		t.Errorf("Error intializing AI telemetry")
	}
}

func TestTrackMetric(t *testing.T) {
	metric := Metric{
		Name:             "test",
		Value:            1.0,
		CustomDimensions: make(map[string]string),
	}

	metric.CustomDimensions["dim1"] = "col1"
	th.TrackMetric(metric)
}

func TestTrackLog(t *testing.T) {
	report := Report{
		Message:          "test",
		Context:          "10a",
		CustomDimensions: make(map[string]string),
	}

	report.CustomDimensions["dim1"] = "col1"
	th.TrackLog(report)
}

func TestClose(t *testing.T) {
	th.Close(10)
}

func TestClosewithoutSend(t *testing.T) {
	aiConfig := AIConfig{
		AppName:                     "testapp",
		AppVersion:                  "v1.0.26",
		BatchSize:                   4096,
		BatchInterval:               2,
		EnableMetadataRefreshThread: false,
		RefreshTimeout:              10,
		EnableLogging:               true,
	}

	thtest := NewAITelemetry("00ca2a73-c8d6-4929-a0c2-cf84545ec225", aiConfig)
	if thtest == nil {
		t.Errorf("Error intializing AI telemetry")
	}

	thtest.Close(10)
}
