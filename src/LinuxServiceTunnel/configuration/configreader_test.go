package configuration

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-container-networking/log"
)

func TestMain(m *testing.M) {
	log.SetName("testservicetunnel")
	log.SetLevel(log.LevelInfo)
	err := log.SetTargetLogDirectory(log.TargetLogfile, "/var/log/")
	if err == nil {
		fmt.Printf("TestST LogDir configuration succeeded\n")
	}

	os.Exit(m.Run())
}

func TestReadConfig(t *testing.T) {
	os.Setenv("AZST_CONFIG_PATH", "testfiles/stconfig1.json")
	stconfig, err := ReadConfigFile("testfiles/stconfig1.json")
	if err != nil {
		t.Fatalf("TestReadConfig failed with error %v", err)
	}

	if len(stconfig.PortsToOpen) != 1 {
		t.Fatalf("Length of portstoopen array is not correct expected:2, Actual:%d", len(stconfig.PortsToOpen))
	}

	if stconfig.PortsToOpen[0] != 540 {
		t.Fatalf("The values of ports are not correct expected:80,443 Actual:%v", stconfig.PortsToOpen)
	}

	if stconfig.PrimaryInterfaceName != "eth1" {
		t.Fatalf("The value of PrimaryInterfaceName is not correct expected:eth1 Actual:%v", stconfig.PrimaryInterfaceName)
	}

	if stconfig.BridgeName != "az0" {
		t.Fatalf("The value of BridgeName is not correct expected:az0 Actual:%v", stconfig.BridgeName)

	}

	if stconfig.Veth1Name != "azveth1" {
		t.Fatalf("The value of Veth1Name is not correct expected:azveth1 Actual:%v", stconfig.Veth1Name)

	}

	if stconfig.Veth2Name != "azveth2" {
		t.Fatalf("The value of Veth2Name is not correct expected:azveth2 Actual:%v", stconfig.Veth2Name)
	}
}

func TestReadConfigEmptyArray(t *testing.T) {
	os.Setenv("AZST_CONFIG_PATH", "testfiles/stconfig2.json")
	stconfig, err := ReadConfigFile("testfiles/stconfig2.json")
	if err != nil {
		t.Fatalf("TestReadConfig failed with error %v", err)
	}

	if len(stconfig.PortsToOpen) != 0 {
		t.Fatalf("Length of portstoopen array is not correct expected:0, Actual:%d", len(stconfig.PortsToOpen))
	}

}

func TestSetDefaults(t *testing.T) {
	os.Setenv("AZST_CONFIG_PATH", "testfiles/stconfig2.json")
	stconfig, err := ReadConfigFile("testfiles/stconfig2.json")
	if err != nil {
		t.Fatalf("TestReadConfig failed with error %v", err)
	}

	SetConfigDefaults(&stconfig)

	if stconfig.PrimaryInterfaceName != "eth0" {
		t.Fatalf("The value of PrimaryInterfaceName is not correct expected:eth1 Actual:%v", stconfig.PrimaryInterfaceName)
	}

	if stconfig.BridgeName != "azurest" {
		t.Fatalf("The value of BridgeName is not correct expected:az0 Actual:%v", stconfig.BridgeName)

	}

	if stconfig.Veth1Name != "azstveth0" {
		t.Fatalf("The value of Veth1Name is not correct expected:azveth1 Actual:%v", stconfig.Veth1Name)

	}

	if stconfig.Veth2Name != "azstveth1" {
		t.Fatalf("The value of Veth2Name is not correct expected:azveth2 Actual:%v", stconfig.Veth2Name)
	}
}

func TestReadConfigEmptyJson(t *testing.T) {
	os.Setenv("AZST_CONFIG_PATH", "testfiles/stconfig3.json")
	stconfig, err := ReadConfigFile("testfiles/stconfig3.json")
	if err != nil {
		t.Fatalf("Not returning error for invalid json")
	}

	if len(stconfig.PortsToOpen) != 0 {
		t.Fatalf("Length of portstoopen array is not correct expected:0, Actual:%d", len(stconfig.PortsToOpen))
	}
}

func TestReadConfigFileNotFound(t *testing.T) {
	os.Setenv("AZST_CONFIG_PATH", "testfiles/stconfignotfound.json")
	_, err := ReadConfigFile("testfiles/stconfignotfound.json")
	if err == nil {
		t.Fatalf("Not returning error for file not found")
	}
}
