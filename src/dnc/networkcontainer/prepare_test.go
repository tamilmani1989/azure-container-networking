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

func assertReservationIn(t *testing.T, res datamodel.ReservationConfiguration, state datamodel.ReservationState, status datamodel.Status) {
	if res.ReservationState != state {
		t.Errorf("expected reservation to be in %s state, got %s", state, res.ReservationState)
	}
	if res.Status != status {
		t.Errorf("expected reservation to be in %s status, got %s", status, res.Status)
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
	assertReservationIn(t, nextReservationState, datamodel.Allocate, datamodel.Pending)
	if len(nextReservationState.SecondaryAllocations) != secondaryIPCount {
		t.Errorf("expected secondary allocation count to be %d, got %d", secondaryIPCount, len(nextReservationState.SecondaryAllocations))
	}
	if nextReservationState.AllocationRequest.SecondaryIPCount != secondaryIPCount {
		t.Errorf("expected ip count in allocation request to be %d, got %d", secondaryIPCount, nextReservationState.AllocationRequest.SecondaryIPCount)
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

func TestPrepareNetworkContainerUpdate(t *testing.T) {
	tests := []struct {
		name                             string
		requestedIPCount                 int
		toBeDeletedIPs                   []string
		expectedReservationStatus        datamodel.Status
		expectedAllocationsInReservation int
		isErrExpected                    bool
	}{
		{
			name:                             "same ip counts, no IP ids to be deleted",
			requestedIPCount:                 5,
			toBeDeletedIPs:                   nil,
			expectedReservationStatus:        datamodel.Completed, // since no further allocations will be needed
			expectedAllocationsInReservation: 5,                   // since no change is needed
		},
		{
			name:                             "same ip counts, IP ids to be deleted belong to allocated ips",
			requestedIPCount:                 5,
			toBeDeletedIPs:                   []string{"key1", "key2"},
			expectedReservationStatus:        datamodel.Pending, // since we will need to allocate 2 more ips
			expectedAllocationsInReservation: 7,                 // we won't remove the ids until the allocations are properly removed, but we should have 2 more ids to allocate
		},
		{
			name:                             "scale up, no IP ids to be deleted",
			requestedIPCount:                 10,
			toBeDeletedIPs:                   nil,
			expectedReservationStatus:        datamodel.Pending,
			expectedAllocationsInReservation: 10, // only ids were added, and should match the requested count
		},
		{
			name:                             "scale up, IP ids to be deleted belong to allocated ips",
			requestedIPCount:                 8,
			toBeDeletedIPs:                   []string{"key1", "key2"},
			expectedReservationStatus:        datamodel.Pending,
			expectedAllocationsInReservation: 10, // ids to be deleted remain, but we add 5 to account for the 2 that will be deleted plus the 5 needed to meet request
		},
		{
			name:                             "scale down, IP ids to be deleted belong to allocated ips",
			requestedIPCount:                 2,
			toBeDeletedIPs:                   []string{"key2", "key4", "key5"},
			expectedReservationStatus:        datamodel.Completed, // once removed, no more allocations needed
			expectedAllocationsInReservation: 5,                   // no ids to add, but we won't remove until the reserve phase. deletion will happen regardless of reservation status
		},
		{
			name:                             "scale down, IP ids to be deleted belong to allocated ips, but allocations still needed",
			requestedIPCount:                 4,
			toBeDeletedIPs:                   []string{"key2", "key5"},
			expectedReservationStatus:        datamodel.Pending,
			expectedAllocationsInReservation: 6, // we add 1 id that needs allocation
		},
		{
			name:                      "scale down, not enough IP ids to be deleted to meet allocation request",
			requestedIPCount:          1,
			toBeDeletedIPs:            []string{"key2", "key5"},
			expectedReservationStatus: datamodel.Pending,
			isErrExpected:             true, // removing the 2 keys will not be enough to meet request
		},
	}

	existingReservation := datamodel.ReservationConfiguration{
		ReservationState: datamodel.Allocate,
		Status:           datamodel.Completed,
		SecondaryAllocations: map[string]string{
			"key1": "10.0.0.5/32",
			"key2": "10.0.0.6/32",
			"key3": "10.0.0.7/32",
			"key4": "10.0.0.8/32",
			"key5": "10.0.0.9/32",
		},
	}

	oldSecondaryIPCount := 5

	for _, tt := range tests {
		ts := tt
		t.Run(ts.name, func(t *testing.T) {
			reservation := cloneReservation(existingReservation)
			reservation.AllocationRequest.SecondaryIPCount = oldSecondaryIPCount

			incomingRequest := api.NetworkContainerRequestV2{
				AllocationRequest: api.AllocationRequest{SecondaryIPCount: ts.requestedIPCount},
				ToBeDeletedIPIds:  ts.toBeDeletedIPs,
			}

			updatedReservation, err := getUpdatedReservation(incomingRequest, reservation)
			if ts.isErrExpected {
				if err == nil {
					t.Fatal("expected error, got none")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				assertReservationIn(t, updatedReservation, datamodel.Allocate, ts.expectedReservationStatus)
			}
		})
	}
}

func cloneReservation(res datamodel.ReservationConfiguration) datamodel.ReservationConfiguration {
	m := make(map[string]string)
	for k, v := range res.SecondaryAllocations {
		m[k] = v
	}
	r := res
	r.SecondaryAllocations = m
	return r
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
