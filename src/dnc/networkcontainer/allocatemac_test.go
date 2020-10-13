package networkcontainer

import (
	"dnc/api"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/networkcontainer/fakes"
	"testing"
)

func getConfiguration (isManaged bool) *configuration.Configuration  {
	config := configuration.Configuration{
		DNCSettings: configuration.DNCSettings{
			Managed: isManaged,
		},
	}

	return &config
}

func getNodeRepo (nodeId string, nodeType string) *fakes.NodeRepo  {
	nodeRepo := fakes.NodeRepo{
		FakeNodeConfig: datamodel.NodeConfiguration{
			NodeID:                      nodeId,
			NodeInfo:                    api.NodeInfoRequest{
				NodeType: nodeType,
			},
		},
	}

	return &nodeRepo
}

func getNCConfig (nodeId string, containerType string) *datamodel.NCConfiguration  {
	ncID := "nc1"
	vnetID := "network1"
	secondaryIPCount := 10

	currentNCState := datamodel.NCConfiguration{
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
	if nc.MACAddress != "TempMAC" {
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

func TestAllocateMAC_HappyPath(t *testing.T){
	var nextNCState   datamodel.NCConfiguration

	nodeId := "testNodeId"

	macAllocator := MacAllocator{
		PersistNC: func(nc *datamodel.NCConfiguration, _ bool) *api.JsonErr {
			if nc != nil {
				nextNCState = *nc
			}
			return nil
		},
		Logger: new(dummyLogger),
	}

	removeFromQueue, err := macAllocator.ProcessAllocateMAC(getNCConfig(nodeId, api.BareMetalContainerType))
	if err != nil {
		t.Errorf("Did not expect error from ProcessAllocateMAC, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("Did not expect removal from queue from ProcessAllocateMAC")
	}

	// Assert that MAC is properly allocated to network container.
	assertMacAllocatedToNC(t, nextNCState)
}

func TestDeallocateMAC_HappyPath(t *testing.T){
	var nextNCState   datamodel.NCConfiguration

	nodeId := "testNodeId"

	macAllocator := MacAllocator{
		PersistNC: func(nc *datamodel.NCConfiguration, _ bool) *api.JsonErr {
			if nc != nil {
				nextNCState = *nc
			}
			return nil
		},
		Logger: new(dummyLogger),
	}

	removeFromQueue, err := macAllocator.ProcessAllocateMAC(getNCConfig(nodeId, api.BareMetalContainerType))
	if err != nil {
		t.Errorf("Did not expect error from ProcessAllocateMAC, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("Did not expect removal from queue from ProcessAllocateMAC")
	}

	// Assert that MAC is properly allocated to network container.
	assertMacAllocatedToNC(t, nextNCState)

	removeFromQueue, err = macAllocator.ProcessDeallocateMAC(getNCConfig(nodeId, api.BareMetalContainerType))
	if err != nil {
		t.Errorf("Did not expect error from ProcessDeallocateMAC, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("Did not expect removal from queue from ProcessDeallocateMAC")
	}

	// Assert that MAC is properly deallocated from network container.
	assertMacDeallocatedFromNC(t, nextNCState)
}

func TestAllocateMAC_NonManagedDNC(t *testing.T){
	nodeId := "testNodeId"

	macAllocator := MacAllocator{
		PersistNC: func(nc *datamodel.NCConfiguration, _ bool) *api.JsonErr {
			return nil
		},
		Logger: new(dummyLogger),
	}

	removeFromQueue, err := macAllocator.ProcessAllocateMAC(getNCConfig(nodeId, api.BareMetalContainerType))
	if err != nil {
		t.Errorf("Did not expect error, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("Did not expect removal from queue")
	}
}
