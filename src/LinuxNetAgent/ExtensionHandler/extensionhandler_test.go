package extensionhandler

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-container-networking/log"
)

func TestMain(m *testing.M) {
	log.SetName("testextensionhandler")
	log.SetLevel(log.LevelInfo)
	err := log.SetTargetLogDirectory(log.TargetLogfile, ".")
	if err == nil {
		fmt.Printf("TestExtension LogDir configuration succeeded\n")
	}

	os.Exit(m.Run())
}

func TestReadHandlerEnvironment(t *testing.T) {
	if _, err := ReadHandlerEnvironment("test/HandlerEnvironment.json"); err != nil {
		t.Fatalf("ReadHandlerEnvironment failed with error %v", err)
	}
}

func TestListFilesUnderConfig(t *testing.T) {
	handlerEnv, err := ReadHandlerEnvironment("test/HandlerEnvironment.json")
	if err != nil {
		t.Fatalf("ReadHandlerEnvironment failed with error %v", err)
	}

	fileNames, err := handlerEnv.ListFilesUnderConfigFolder()
	if err != nil {
		t.Fatalf("ListFilesUnderConfigFolder failed with error %v", err)
	}

	if len(fileNames) <= 0 {
		t.Fatalf("files not found")
	}
}

func TestFindLatestSettingNumber(t *testing.T) {
	handlerEnv, err := ReadHandlerEnvironment("test/HandlerEnvironment.json")
	if err != nil {
		t.Fatalf("ReadHandlerEnvironment failed with error %v", err)
	}

	fileNames, err := handlerEnv.ListFilesUnderConfigFolder()
	if err != nil {
		t.Fatalf("ListFilesUnderConfigFolder failed with error %v", err)
	}

	if len(fileNames) <= 0 {
		t.Fatalf("files not found")
	}

	settingsFiles := GetSettingsFiles(fileNames)
	if len(settingsFiles) <= 0 {
		t.Fatalf("setting files not found")
	}

	latestSetting := FindLatestSettingNumber(settingsFiles)
	if latestSetting != 1 {
		t.Fatalf("Failed to find latest setting number. Got %d", latestSetting)
	}
}

func TestReadServiceTunnelConfig(t *testing.T) {
	handlerEnv, err := ReadHandlerEnvironment("test/HandlerEnvironment.json")
	if err != nil {
		t.Fatalf("ReadHandlerEnvironment failed with error %v", err)
	}

	handlerConfig, err := handlerEnv.ReadHandlerConfig(1)
	if err != nil {
		t.Fatalf("ReadHandlerConfig failed with error %v", err)
	}

	stConfig, err := handlerConfig.ReadServiceTunnelConfig()
	if err != nil {
		t.Fatalf("ReadServiceTunnelConfig failed with error %v", err)
	}

	if len(stConfig.PortsToOpen) != 2 {
		t.Fatalf("Expected len is 2 but actual len of portstoopen is %d", len(stConfig.PortsToOpen))
	}

	for _, port := range stConfig.PortsToOpen {
		if port != 80 && port != 443 {
			t.Fatalf("Couldn't find expected ports")
		}
	}

}

func TestWriteStatus(t *testing.T) {
	handlerEnv, err := ReadHandlerEnvironment("test/HandlerEnvironment.json")
	if err != nil {
		t.Fatalf("ReadHandlerEnvironment failed with error %v", err)
	}

	if err := handlerEnv.WriteStatusLog(1, "Success", "TestOp", 0, "Unitest succeeded"); err != nil {
		t.Fatalf("WriteStatusLog failed with error %v", err)
	}
}
