package aitelemetry

import (
	"os"
	"runtime"
	"testing"

	"github.com/Azure/azure-container-networking/platform"
)

var th *TelemetryHandle

func TestMain(m *testing.M) {

	if runtime.GOOS == "linux" {
		platform.ExecuteCommand("cp metadata_test.json /tmp/azuremetadata.json")
	} else {
		platform.ExecuteCommand("copy metadata_test.json azuremetadata.json")
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestNewAITelemetry(t *testing.T) {
	th = NewAITelemetry("00ca2a73-c8d6-4929-a0c2-cf84545ec225", "testapp", "v1.0.26")
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
	if th.metadata.Location != "eastus" {
		t.Errorf("Metadata is not retrieved Got location as %v", th.metadata.Location)
	}
}

func TestTrackLog(t *testing.T) {
	report := Report{
		Message:          "test",
		Context:          "10a",
		CustomDimensions: make(map[string]string),
	}

	report.CustomDimensions["dim1"] = "col1"
	th.TrackLog(report)
	if th.metadata.Location != "eastus" {
		t.Errorf("Metadata is not retrieved Got location as %v", th.metadata.Location)
	}
}

func TestClose(t *testing.T) {
	th.Close(10)
}
