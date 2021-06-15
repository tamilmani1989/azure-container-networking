package networkcontainer

import (
	"dnc/api"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/logger/loggermock"
	"dnc/macmgr"
	"dnc/networkcontainer/fakes"
	"testing"
)

type FakeMacManagerConnection struct {
}

func (m *FakeMacManagerConnection) AllocateMac(allocationId string) (*macmgr.MacAllocatorResponse, *api.JsonErr) {
	resp := &macmgr.MacAllocatorResponse{
		AllocationId: allocationId,
		MacAddress:   "A1B1C1D1E1F1",
		IsSuccess:    true,
		ErrorMessage: "",
		ShouldRetry:  false,
	}
	return resp, nil
}

func (m *FakeMacManagerConnection) DeallocateMac(allocationId string) (*macmgr.MacAllocatorResponse, *api.JsonErr) {
	resp := &macmgr.MacAllocatorResponse{
		AllocationId: allocationId,
		MacAddress:   "",
		IsSuccess:    true,
		ErrorMessage: "",
		ShouldRetry:  false,
	}
	return resp, nil
}

func getConfiguration() *configuration.Configuration {
	config := configuration.Configuration{
		DNCSettings: configuration.DNCSettings{
			Managed: true,
		},
	}

	return &config
}

func getNodeRepo(nodeId string, nodeType string) *fakes.NodeRepo {
	nodeRepo := fakes.NodeRepo{
		FakeNodeConfig: datamodel.NodeConfiguration{
			NodeID: nodeId,
			NodeInfo: api.NodeInfoRequest{
				NodeType: nodeType,
			},
		},
	}

	return &nodeRepo
}

func getNC(nodeId string, containerType string) *datamodel.NetworkContainerEntity {
	ncID := "nc1"
	vnetID := "network1"
	subnetName := "subnet1"
	secondaryIPCount := 10

	currentNCState := datamodel.NetworkContainerEntity{
		RowKey: datamodel.GetNCKey(vnetID, ncID),
		Config:              datamodel.NCConfiguration{
			NetworkContainerID: ncID,
			NetworkID:          vnetID,
			RequestV2: api.NetworkContainerRequestV2{
				AllocationRequest: api.AllocationRequest{
					SecondaryIPCount: secondaryIPCount,
				},
				AssociationInfo: api.AssociationInfo{
					NodeID:        nodeId,
					ContainerType: containerType,
				},
				ToBeDeletedIPIds: nil,
			},
		},
		SubnetKey: datamodel.GetSubnetKey(vnetID, subnetName),
	}

	return &currentNCState
}

func assertMacAllocatedToNC(t *testing.T, nc datamodel.NCConfiguration) {
	if nc.State != datamodel.Publish {
		t.Errorf("expected NC to be in Publish state, got %s", nc.State)
	}
	if nc.Status != datamodel.Pending {
		t.Errorf("expected NC to be in Pending status, got %s", nc.Status)
	}

	// TODO: remove when MACAddress is properly populated.
	if nc.MACAddress != "A1B1C1D1E1F1" {
		t.Error("MAC address was expected to be TempMAC.")
	}
}

func assertMacDeallocatedFromNC(t *testing.T, nc datamodel.NCConfiguration) {
	if nc.State != datamodel.UnReserve {
		t.Errorf("expected NC to be in Unreserve state, got %s", nc.State)
	}
	if nc.Status != datamodel.Pending {
		t.Errorf("expected NC to be in Pending status, got %s", nc.Status)
	}

	if nc.MACAddress != "" {
		t.Error("MAC address was expected to be empty.")
	}
}

func TestAllocateMAC_HappyPath(t *testing.T) {
	var nextNCState datamodel.NetworkContainerEntity

	nodeId := "testNodeId"

	fakeMacMgrClient := FakeMacManagerConnection{}

	macAllocator := MacAllocator{
		PersistNC: func(nc *datamodel.NetworkContainerEntity) *api.JsonErr {
			if nc != nil {
				nextNCState = *nc
			}
			return nil
		},
		Logger:           loggermock.NewDummyLogger(),
		MacManagerClient: &fakeMacMgrClient,
	}

	removeFromQueue, err := macAllocator.ProcessAllocateMAC(getNC(nodeId, api.BareMetalContainerType))
	if err != nil {
		t.Errorf("Did not expect error from ProcessAllocateMAC, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("Did not expect removal from queue from ProcessAllocateMAC")
	}

	// Assert that MAC is properly allocated to network container.
	assertMacAllocatedToNC(t, nextNCState.Config)
}

func TestDeallocateMAC_HappyPath(t *testing.T) {
	var nextNCState datamodel.NetworkContainerEntity

	nodeId := "testNodeId"

	fakeMacMgrClient := FakeMacManagerConnection{}

	macAllocator := MacAllocator{
		PersistNC: func(nc *datamodel.NetworkContainerEntity) *api.JsonErr {
			if nc != nil {
				nextNCState = *nc
			}
			return nil
		},
		Logger:           loggermock.NewDummyLogger(),
		MacManagerClient: &fakeMacMgrClient,
	}

	removeFromQueue, err := macAllocator.ProcessAllocateMAC(getNC(nodeId, api.BareMetalContainerType))
	if err != nil {
		t.Errorf("Did not expect error from ProcessAllocateMAC, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("Did not expect removal from queue from ProcessAllocateMAC")
	}

	// Assert that MAC is properly allocated to network container.
	assertMacAllocatedToNC(t, nextNCState.Config)

	removeFromQueue, err = macAllocator.ProcessDeallocateMAC(getNC(nodeId, api.BareMetalContainerType))
	if err != nil {
		t.Errorf("Did not expect error from ProcessDeallocateMAC, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("Did not expect removal from queue from ProcessDeallocateMAC")
	}

	// Assert that MAC is properly deallocated from network container.
	assertMacDeallocatedFromNC(t, nextNCState.Config)
}

func TestAllocateMAC_NonManagedDNC(t *testing.T) {
	nodeId := "testNodeId"

	fakeMacMgrClient := FakeMacManagerConnection{}

	macAllocator := MacAllocator{
		PersistNC: func(nc *datamodel.NetworkContainerEntity) *api.JsonErr {
			return nil
		},
		Logger:           loggermock.NewDummyLogger(),
		MacManagerClient: &fakeMacMgrClient,
	}

	removeFromQueue, err := macAllocator.ProcessAllocateMAC(getNC(nodeId, api.BareMetalContainerType))
	if err != nil {
		t.Errorf("Did not expect error, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("Did not expect removal from queue")
	}
}
