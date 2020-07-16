package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/utils"
	"testing"
)

func TestPrepareNetworkContainer_CreateHappyPath(t *testing.T) {
	var (
		nextReservationState datamodel.ReservationConfiguration
		nextNCState          datamodel.NCConfiguration
		nextNodeState        datamodel.NodeConfiguration
	)

	// The "prepare" stage only signals whether the nc should continue transitioning thru states, but there are side
	// effects (db state, essentially) associated with those transitions. The dummy db calls below capture these changes
	// so we can assert on those after the stage successfully completes.

	preparer := Preparer{
		PersistReservation: func(r *datamodel.ReservationConfiguration) *api.JsonErr {
			if r != nil {
				nextReservationState = *r
			}
			return nil
		},
		UpdateNodeContainers: func(_, networkID, ncID, reservationID, _ string, _, _ bool, _ int) *api.JsonErr {
			nextNodeState = datamodel.NodeConfiguration{
				DeployedNetworkContainerMap: map[string]map[string]string{
					networkID: {
						ncID: reservationID,
					},
				},
			}
			return nil
		},
		PersistNC: func(nc *datamodel.NCConfiguration, _ bool) *api.JsonErr {
			if nc != nil {
				nextNCState = *nc
			}
			return nil
		},
		CNS:    &fakeCNSChannel{limit: 3, err: nil},
		Locks:  utils.NewSessionManager(nil),
		Logger: new(dummyLogger),
	}

	ncID := "nc1"
	vnetID := "network1"
	nodeID := "node1"
	secondaryIPCount := 10

	currentNCState := datamodel.NCConfiguration{
		NetworkContainerID: ncID,
		NetworkID:          vnetID,
		RequestV2: api.NetworkContainerRequestV2{
			AllocationRequest: api.AllocationRequest{
				SecondaryIPCount: secondaryIPCount,
			},
			AssociationInfo: api.AssociationInfo{
				NodeID: nodeID,
			},
			ToBeDeletedIPIds: nil,
		},
	}

	removeFromQueue, err := preparer.PrepareNetworkContainerV2(&currentNCState)
	if err != nil {
		t.Errorf("did not expect error, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("did not expect removal from queue")
	}

	// network container assertions
	if nextNCState.State != datamodel.Reserve {
		t.Errorf("expected NC to be in reserve state, got %s", nextNCState.State)
	}
	if nextNCState.Status != datamodel.Pending {
		t.Errorf("expected NC to be in pending status, got %s", nextNCState.Status)
	}
	if nextNCState.ReservationID == "" {
		t.Error("expected reservation id to have been assigned to NC, got none")
	}

	// an important thing to note are also the side effects of the original node config pointer
	// since calling code keeps a reference to it without necessarily reloading from db
	if currentNCState.State != datamodel.Reserve {
		t.Errorf("expected input NC to be in reserve state, got %s", nextNCState.State)
	}
	if currentNCState.Status != datamodel.Pending {
		t.Errorf("expected input NC to be in pending status, got %s", nextNCState.Status)
	}
	if currentNCState.ReservationID == "" {
		t.Error("expected reservation id to have been assigned to input NC, got none")
	}

	// reservation assertions
	if nextReservationState.ReservationState != datamodel.Allocate {
		t.Errorf("expected reservation to be in allocate state, got %s", nextReservationState.ReservationState)
	}
	if nextReservationState.Status != datamodel.Pending {
		t.Errorf("expected reservation to be in pending status, got %s", nextReservationState.Status)
	}
	if len(nextReservationState.SecondaryAllocations) != secondaryIPCount {
		t.Errorf("expected secondary allocation count to be %d, got %d", secondaryIPCount, len(nextReservationState.SecondaryAllocations))
	}

	// node assertions
	res, ok := nextNodeState.DeployedNetworkContainerMap[vnetID][ncID]
	if !ok {
		t.Fatal("node nc map does not contain a reservation for the given vnet and nc")
	}
	if res == "" {
		t.Error("expected reservation id associated to node to not be empty")
	}
}

func TestPrepareNetworkContainer_Error(t *testing.T) {
	tests := []struct {
		name         string
		resErr       *api.JsonErr
		nodeErr      *api.JsonErr
		ncErr        *api.JsonErr
		expectRemove bool
	}{
		{
			name:         "when node container capacity is exhausted, the nc should not keep trying to move further",
			nodeErr:      &api.JsonErr{Code: api.NodeContainerCapacityExhausted},
			expectRemove: true,
		},
		{
			name:         "when node container capacity is exhausted, but nc cannot be persisted, keep in queue",
			nodeErr:      &api.JsonErr{Code: api.NodeContainerCapacityExhausted},
			ncErr:        &api.JsonErr{Code: api.StatusInternalServerError},
			expectRemove: false,
		},
		{
			name:         "any other node error should keep nc in queue",
			nodeErr:      &api.JsonErr{Code: api.StatusInternalServerError},
			expectRemove: false,
		},
		{
			name:         "any reservation error should keep nc in queue",
			resErr:       &api.JsonErr{Code: api.StatusInternalServerError},
			expectRemove: false,
		},
		{
			name:         "any node config error should keep nc in queue",
			ncErr:        &api.JsonErr{Code: api.StatusInternalServerError},
			expectRemove: false,
		},
	}

	for _, ts := range tests {
		tt := ts
		t.Run(tt.name, func(t *testing.T) {
			preparer := Preparer{
				PersistReservation:   func(_ *datamodel.ReservationConfiguration) *api.JsonErr { return tt.resErr },
				UpdateNodeContainers: func(_, _, _, _, _ string, _, _ bool, _ int) *api.JsonErr { return tt.nodeErr },
				PersistNC:            func(_ *datamodel.NCConfiguration, _ bool) *api.JsonErr { return tt.ncErr },
				CNS:                  &fakeCNSChannel{limit: 3, err: nil},
				Locks:                utils.NewSessionManager(nil),
				Logger:               new(dummyLogger),
			}

			currentNCState := datamodel.NCConfiguration{
				NetworkContainerID: "nc1",
				NetworkID:          "vnet1",
				RequestV2: api.NetworkContainerRequestV2{
					AllocationRequest: api.AllocationRequest{SecondaryIPCount: 5},
					AssociationInfo:   api.AssociationInfo{NodeID: "node1"},
				},
			}

			removeFromQueue, err := preparer.PrepareNetworkContainerV2(&currentNCState)
			if err != nil && removeFromQueue {
				t.Error("removing from queue should not be accompanied by an error in this stage")
			}
			if tt.expectRemove != removeFromQueue {
				t.Errorf("expected removal to be '%v', got '%v'", tt.expectRemove, removeFromQueue)
			}
		})
	}
}
