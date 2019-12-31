package aitelemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/platform"
)

var th TelemetryHandle

func TestMain(m *testing.M) {
	log.SetLogDirectory("/var/log/")
	log.SetName("testaitelemetry")
	log.SetLevel(log.LevelInfo)
	err := log.SetTarget(log.TargetLogfile)
	if err == nil {
		fmt.Printf("TestST LogDir configuration succeeded\n")
	}

	if runtime.GOOS == "linux" {
		platform.ExecuteCommand("cp metadata_test.json /tmp/azuremetadata.json")
	} else {
		metadataFile := filepath.FromSlash(os.Getenv("TEMP")) + "\\azuremetadata.json"
		cmd := fmt.Sprintf("copy metadata_test.json %s", metadataFile)
		platform.ExecuteCommand(cmd)
	}

	exitCode := m.Run()

	if runtime.GOOS == "linux" {
		platform.ExecuteCommand("rm /tmp/azuremetadata.json")
	} else {
		metadataFile := filepath.FromSlash(os.Getenv("TEMP")) + "\\azuremetadata.json"
		cmd := fmt.Sprintf("del %s", metadataFile)
		platform.ExecuteCommand(cmd)
	}

	os.Exit(exitCode)
}

func TestEmptyAIKey(t *testing.T) {
	os.Setenv("AZACN_TESTENV", "test")

	aiConfig := AIConfig{
		AppName:                      "testapp",
		AppVersion:                   "v1.0.26",
		BatchSize:                    4096,
		BatchInterval:                2,
		RefreshTimeout:               10,
		DebugMode:                    true,
		DisableMetadataRefreshThread: true,
	}
	th := NewAITelemetry("", aiConfig)
	if th == nil {
		t.Errorf("Error intializing AI telemetry")
	}
	th.Close(10)
}

func TestNewAITelemetry(t *testing.T) {
	os.Setenv("AZACN_TESTENV", "test")

	aiConfig := AIConfig{
		AppName:                      "testapp",
		AppVersion:                   "v1.0.26",
		BatchSize:                    4096,
		BatchInterval:                2,
		RefreshTimeout:               10,
		DebugMode:                    true,
		DisableMetadataRefreshThread: true,
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
	os.Setenv("AZACN_TESTENV", "test")

	aiConfig := AIConfig{
		AppName:                      "testapp",
		AppVersion:                   "v1.0.26",
		BatchSize:                    4096,
		BatchInterval:                2,
		DisableMetadataRefreshThread: true,
		RefreshTimeout:               10,
	}

	thtest := NewAITelemetry("00ca2a73-c8d6-4929-a0c2-cf84545ec225", aiConfig)
	if thtest == nil {
		t.Errorf("Error intializing AI telemetry")
	}

	thtest.Close(10)
}
