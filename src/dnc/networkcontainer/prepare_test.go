package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/networkcontainer/fakes"
	"dnc/utils"
	"testing"
)

func assertNCInReservePending(t *testing.T, nc datamodel.NCConfiguration) {
	if nc.State != datamodel.Reserve {
		t.Errorf("expected NC to be in reserve state, got %s", nc.State)
	}
	if nc.Status != datamodel.Pending {
		t.Errorf("expected NC to be in pending status, got %s", nc.Status)
	}
	if nc.ReservationID == "" {
		t.Error("expected reservation id to have been assigned to NC, got none")
	}
}

func assertReservationAllocatePending(t *testing.T, res datamodel.ReservationConfiguration, expectedIPCount int) {
	if res.ReservationState != datamodel.Allocate {
		t.Errorf("expected reservation to be in allocate state, got %s", res.ReservationState)
	}
	if res.Status != datamodel.Pending {
		t.Errorf("expected reservation to be in pending status, got %s", res.Status)
	}
	if len(res.SecondaryAllocations) != expectedIPCount {
		t.Errorf("expected secondary allocation count to be %d, got %d", expectedIPCount, len(res.SecondaryAllocations))
	}
	if res.AllocationRequest.SecondaryIPCount != expectedIPCount {
		t.Errorf("expected ip count in allocation request to be %d, got %d", expectedIPCount, res.AllocationRequest.SecondaryIPCount)
	}
}

func TestPrepareNetworkContainer_CreateHappyPath(t *testing.T) {
	var (
		nextNCState   datamodel.NCConfiguration
		nextNodeState datamodel.NodeConfiguration
	)

	// The "prepare" stage only signals whether the nc should continue transitioning thru states, but there are side
	// effects (db state, essentially) associated with those transitions. The dummy db calls below capture these changes
	// so we can assert on those after the stage successfully completes.

	reservationRepo := fakes.NewReservationRepo()

	preparer := Preparer{
		Reservations: reservationRepo,
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
	assertNCInReservePending(t, nextNCState)

	// an important thing to note are also the side effects of the original node config pointer
	// since calling code keeps a reference to it without necessarily reloading from db
	assertNCInReservePending(t, currentNCState)

	// reservation assertions
	expectedResKey := datamodel.GetReservationKey(currentNCState.NetworkID, currentNCState.ReservationID)
	nextReservationState, ok := reservationRepo.Data[expectedResKey]
	if !ok {
		t.Fatalf("expected reservation with key %s, got none", expectedResKey)
	}
	assertReservationAllocatePending(t, nextReservationState, secondaryIPCount)

	// node assertions
	res, ok := nextNodeState.DeployedNetworkContainerMap[vnetID][ncID]
	if !ok {
		t.Fatal("node nc map does not contain a reservation for the given vnet and nc")
	}
	if res == "" {
		t.Error("expected reservation id associated to node to not be empty")
	}
}

func TestPrepareNetworkContainer_UpdateScaleUp(t *testing.T) {
	reservationID := "res1"
	ncID := "nc1"
	vnetID := "network1"
	nodeID := "node1"
	reservationKey := datamodel.GetReservationKey(vnetID, reservationID)
	oldIPRequestCount := 5
	newIPRequestCount := 10

	// existing reservation
	reservationRepo := fakes.NewReservationRepo()
	reservationRepo.Data[reservationKey] = datamodel.ReservationConfiguration{
		ReservationID:     reservationID,
		NetworkID:         vnetID,
		AllocatedSubnet:   "10.0.0.4/32",
		ReservationState:  datamodel.Allocate,
		Status:            datamodel.Completed,
		AllocationRequest: api.AllocationRequest{SecondaryIPCount: oldIPRequestCount},
		AssociatedNCID:    ncID,
		SecondaryAllocations: map[string]string{
			"key1": "10.0.0.5/32",
			"key2": "10.0.0.6/32",
			"key3": "10.0.0.7/32",
			"key4": "10.0.0.8/32",
			"key5": "10.0.0.9/32",
		},
	}

	var nextNCState datamodel.NCConfiguration
	preparer := Preparer{
		Reservations:         reservationRepo,
		UpdateNodeContainers: func(_, _, _, _, _ string, _, _ bool, _ int) *api.JsonErr { return nil },
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

	currentNCState := datamodel.NCConfiguration{
		NetworkContainerID: ncID,
		NetworkID:          vnetID,
		ReservationID:      reservationID,
		RequestV2: api.NetworkContainerRequestV2{
			AllocationRequest: api.AllocationRequest{SecondaryIPCount: newIPRequestCount},
			AssociationInfo:   api.AssociationInfo{NodeID: nodeID},
			ToBeDeletedIPIds:  nil,
		},
	}

	removeFromQueue, err := preparer.PrepareNetworkContainerV2(&currentNCState)
	if err != nil {
		t.Errorf("did not expect error, got: %+v", err)
	}
	if removeFromQueue {
		t.Error("did not expect removal from queue")
	}

	expectedResKey := datamodel.GetReservationKey(currentNCState.NetworkID, currentNCState.ReservationID)
	nextReservationState, ok := reservationRepo.Data[expectedResKey]
	if !ok {
		t.Fatalf("expected reservation with key %s, got none", expectedResKey)
	}
	assertReservationAllocatePending(t, nextReservationState, newIPRequestCount)

	assertNCInReservePending(t, nextNCState)
	assertNCInReservePending(t, currentNCState)
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
			reservationRepo := fakes.NewReservationRepo()
			reservationRepo.PutErr = tt.resErr
			preparer := Preparer{
				Reservations:         reservationRepo,
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
