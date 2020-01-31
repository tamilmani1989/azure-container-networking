package servicetunnel

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-container-networking/log"
)

const (
	primary = "eth0"
	target  = "azure0"
	veth0   = "veth0"
	veth1   = "veth1"
)

func TestMain(m *testing.M) {
	log.SetName("testservicetunnel")
	log.SetLevel(log.LevelInfo)
	err := log.SetTargetLogDirectory(log.TargetLogfile, ".")
	if err == nil {
		fmt.Printf("TestST LogDir configuration succeeded\n")
	}

	os.Exit(m.Run())
}

func TestCloudGetPrimaryIP(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: primary, TargetInterfaceName: target}

	if err := st.GetPrimaryIP(); err != nil {
		t.Fatalf("GetPrimaryIP failed with error %v", err)
	}
}

func TestCloudGetIPv6Address(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: primary, TargetInterfaceName: target}

	if err := st.GetPrimaryIP(); err != nil {
		t.Fatalf("GetPrimaryIP failed with error %v", err)
	}

	if err := st.GetIPv6Address(); err != nil {
		t.Fatalf("GetIPv6Address failed with error %v", err)
	}
}

func TestCloudGetRoutesAndGateway(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: primary, TargetInterfaceName: target}

	if err := st.GetRoutesAndGateway(); err != nil {
		t.Fatalf("GetRoutesAndGateway failed with error %v", err)
	}
}

func TestCreateAndDeleteSTBridge(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: primary, TargetInterfaceName: target}

	if err := st.CreateSTBridge(); err != nil {
		t.Fatalf("CreateSTBridge failed with error %s", err.Error())
	}

	st = ServiceTunnel{PrimaryInterfaceName: target}

	if err := st.DeleteSTBridge(false); err != nil {
		t.Fatalf("DeleteSTBridge failed with error %s", err.Error())
	}
}

func TestApplyIPAddresses(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: primary, TargetInterfaceName: target}

	if err := st.GetPrimaryIP(); err != nil {
		t.Fatalf("GetPrimaryIP failed with error %v", err)
	}

	if err := st.CreateSTBridge(); err != nil {
		t.Fatalf("CreateSTBridge failed with error %s", err.Error())
	}

	if err := st.ApplyIPAddresses(); err != nil {
		t.Fatalf("ApplyIPAddresses failed with error %s", err.Error())
	}

	st = ServiceTunnel{PrimaryInterfaceName: target}

	if err := st.GetPrimaryIP(); err != nil {
		t.Fatalf("GetPrimaryIP failed with error %v", err)
	}

	if err := st.RemoveIP(); err != nil {
		t.Fatalf("RemoveIP failed with error %v", err)
	}

	if err := st.DeleteSTBridge(false); err != nil {
		t.Fatalf("DeleteSTBridge failed with error %s", err.Error())
	}
}

func CreateAndSetupBridge(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: primary, TargetInterfaceName: target}

	if err := st.GetPrimaryIP(); err != nil {
		t.Fatalf("GetPrimaryIP failed with error %v", err)
	}

	if err := st.GetRoutesAndGateway(); err != nil {
		t.Fatalf("GetRoutesAndGateway failed with error %v", err)
	}

	if err := st.CreateSTBridge(); err != nil {
		t.Fatalf("CreateSTBridge failed with error %s", err.Error())
	}

	if err := st.ApplyIPAddresses(); err != nil {
		t.Fatalf("ApplyIPAddresses failed with error %s", err.Error())
	}

	if err := st.RemoveIP(); err != nil {
		t.Fatalf("RemoveIP failed with error %v", err)
	}

	if err := st.SetBridgeMasterInterface(st.PrimaryInterfaceName); err != nil {
		t.Fatalf("SetBridgeMasterInterface failed with error %v", err)
	}

	if err := st.ApplyRoutes(); err != nil {
		t.Fatalf("ApplyRoutes failed with error %v", err)
	}
}

func DeleteAndSetupPrimary(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: target, TargetInterfaceName: primary}

	if err := st.GetPrimaryIP(); err != nil {
		t.Fatalf("GetPrimaryIP failed with error %v", err)
	}

	if err := st.GetRoutesAndGateway(); err != nil {
		t.Fatalf("GetRoutesAndGateway failed with error %v", err)
	}

	if err := st.DeleteSTBridge(true); err != nil {
		t.Fatalf("DeleteSTBridge failed with error %s", err.Error())
	}

	if err := st.ApplyIPAddresses(); err != nil {
		t.Fatalf("ApplyIPAddresses failed with error %v", err)
	}

	if err := st.ApplyRoutes(); err != nil {
		t.Fatalf("ApplyRoutes failed with error %v", err)
	}
}

func TestSetBridgeMaster(t *testing.T) {
	CreateAndSetupBridge(t)
	DeleteAndSetupPrimary(t)
}

func TestCreateVethPair(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: primary, TargetInterfaceName: target}

	if err := st.CreateVethPair(veth0, veth1); err != nil {
		t.Fatalf("CreateVethPair failed with error %v", err)
	}

	if err := st.DeleteVethPair(); err != nil {
		t.Fatalf("DeleteVethPair failed with error %v", err)
	}
}

func TestSetupVeth(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: primary, TargetInterfaceName: target}

	if err := st.GetPrimaryIP(); err != nil {
		t.Fatalf("GetPrimaryIP failed with error %v", err)
	}

	if err := st.GetIPv6Address(); err != nil {
		t.Fatalf("GetIPv6Address failed with error %v", err)
	}

	if err := st.CreateSTBridge(); err != nil {
		t.Fatalf("CreateAcrBridge failed with error %v", err)
	}

	if err := st.CreateVethPair(veth0, veth1); err != nil {
		t.Fatalf("CreateVethPair failed with error %v", err)
	}

	if err := st.SetBridgeMasterInterface(veth0); err != nil {
		t.Fatalf("SetBridgeMasterInterface failed with error %v", err)
	}

	if err := st.ApplyIPv6ToVeth(); err != nil {
		t.Fatalf("ApplyIPv6ToVeth failed with error %v", err)
	}

	if err := st.DeleteVethPair(); err != nil {
		t.Fatalf("DeleteVethPair failed with error %v", err)
	}

	st = ServiceTunnel{PrimaryInterfaceName: target}
	if err := st.DeleteSTBridge(false); err != nil {
		t.Fatalf("DeleteSTBridge failed with error %s", err.Error())
	}
}

func TestSetupRoutesAndTables(t *testing.T) {
	st := ServiceTunnel{PrimaryInterfaceName: primary, TargetInterfaceName: target}

	if err := st.GetPrimaryIP(); err != nil {
		t.Fatalf("GetPrimaryIP failed with error %v", err)
	}

	if err := st.GetIPv6Address(); err != nil {
		t.Fatalf("GetIPv6Address failed with error %v", err)
	}

	if err := st.CreateVethPair(veth0, veth1); err != nil {
		t.Fatalf("CreateVethPair failed with error %v", err)
	}

	if err := st.GetVethMac(); err != nil {
		t.Fatalf("GetInterfaceMacAddress failed with error %v", err)
	}

	if err := st.ApplyMacDnatRule("A"); err != nil {
		t.Fatalf("AddMacDnatRule failed with error %v", err)
	}

	if err := st.SetFakeIPv6Route(); err != nil {
		t.Fatalf("SetFakeIPv6Route failed with error %v", err)
	}

	if err := st.SetFakeIPv6Neighbor(); err != nil {
		t.Fatalf("SetFakeIPv6Route failed with error %v", err)
	}

	if err := st.DeleteVethPair(); err != nil {
		t.Fatalf("DeleteVethPair failed with error %v", err)
	}

	if err := st.ApplyMacDnatRule("D"); err != nil {
		t.Fatalf("Removing mac dnat rule failed with error %v", err)
	}

	st.TargetInterfaceName = primary
	if err := st.DeleteFakeIPv6Route(); err != nil {
		t.Fatalf("DeleteFakeIPv6Route failed with error %v", err.Error())
	}

	if err := st.DeleteFakeIPv6Neighbor(); err != nil {
		t.Fatalf("DeleteFakeIPv6Neighbor failed with error %v", err)
	}
}
