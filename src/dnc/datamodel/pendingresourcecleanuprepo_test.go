package datamodel

import (
	"dnc/api"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggermock"
	"reflect"
	"testing"
)

func TestCreateAndDeletePendingCleanupRequest(t *testing.T) {
	pendingCleanupRepo := setup()
	requestKey := "nc1"

	ipCleanup := map[string][]string{
		"vnet_v1_subnet_s1": {"10.0.0.1/32", "10.0.0.2/32"},
		"vnet_v2_subnet_s2": {"10.0.0.1/32"}}

	cleanupRequest := &CleanupContext{
		ToBeDeletedIpAllocations: ipCleanup,
		Type:                     IPCleanup}

	entity := validateCreateOrUpdate(t, pendingCleanupRepo, requestKey, cleanupRequest, cleanupRequest)

	// Idempotent create, it will not fail again however it will
	// update the etag
	validateCreateOrUpdate(t, pendingCleanupRepo, requestKey, cleanupRequest, cleanupRequest)

	// Now update the new cleanup map and update, udpate should merge all the entries
	newIpCleanup := map[string][]string{
		"vnet_v1_subnet_s1": {"10.0.0.3/32"}}

	cleanupRequest = &CleanupContext{
		ToBeDeletedIpAllocations: newIpCleanup,
		Type:                     IPCleanup}

	var expectedIpCleanup map[string][]string = map[string][]string{
		"vnet_v1_subnet_s1": {"10.0.0.1/32", "10.0.0.2/32", "10.0.0.3/32"},
		"vnet_v2_subnet_s2": {"10.0.0.1/32"}}

	expectedCleanupRequest := &CleanupContext{
		ToBeDeletedIpAllocations: expectedIpCleanup,
		Type:                     IPCleanup}

	latestEntity := validateCreateOrUpdate(t, pendingCleanupRepo, requestKey, cleanupRequest, expectedCleanupRequest)

	// Now lets try to delete the entity using an old Tag
	t.Logf("Delete entity using an old etag %s",
		entity.Etag)
	err := pendingCleanupRepo.DeleteCleanupRequest(requestKey, entity.Etag)
	if err == nil {
		t.Fatalf("Expected to fail in deleting Pending Cleanup Request for requestKey:%s, etag %+v, err %s",
			requestKey,
			entity.Etag,
			err)
	}

	err = pendingCleanupRepo.DeleteCleanupRequest(requestKey, latestEntity.Etag)
	if err != nil {
		t.Fatalf("Failed to delete the Pending Cleanup Request for requestKey:%s, etag %+v, err %s",
			requestKey,
			entity.Etag,
			err)
	}
}

func TestCreateIpCleanupRequestFromReservationConfig(t *testing.T) {
	pendingCleanupRepo := setup()

	reserveConfig := &ReservationConfiguration{
		ReservationID:   "R1",
		NetworkID:       "v1",
		AllocatedSubnet: "10.0.0.1/32",
		AssociatedNCID:  "NC1",
		AllocationRequest: api.AllocationRequest{
			SubnetName: "sub1",
		},
		SecondaryAllocations: map[string]string{
			"A1": "10.0.0.2/32",
			"A2": "10.0.0.3/32",
		},
	}

	expectedIpCleanup := make(map[string][]string)
	expectedIpCleanup["Network_v1_Subnet_sub1"] = append(expectedIpCleanup["Network_v1_Subnet_sub1"], reserveConfig.AllocatedSubnet)
	for _, allocatedIp := range reserveConfig.SecondaryAllocations {
		expectedIpCleanup["Network_v1_Subnet_sub1"] = append(expectedIpCleanup["Network_v1_Subnet_sub1"], allocatedIp)
	}

	expectedCleanupRequest := &CleanupContext{
		ToBeDeletedIpAllocations: expectedIpCleanup,
		Type:                     IPCleanup}

	entity := validateCreateIPCleanupRequest(t, pendingCleanupRepo, reserveConfig, expectedCleanupRequest)

	reserveConfig = &ReservationConfiguration{
		ReservationID:   "R1",
		NetworkID:       "v1",
		AllocatedSubnet: "10.0.0.1/32",
		AssociatedNCID:  "NC1",
		AllocationRequest: api.AllocationRequest{
			SubnetName: "sub1",
		},
		SecondaryAllocations: map[string]string{
			"A1": "10.0.0.5/32",
		},
	}
	expectedIpCleanup["Network_v1_Subnet_sub1"] = append(expectedIpCleanup["Network_v1_Subnet_sub1"], "10.0.0.5/32")

	expectedCleanupRequest = &CleanupContext{
		ToBeDeletedIpAllocations: expectedIpCleanup,
		Type:                     IPCleanup}
	entity = validateCreateIPCleanupRequest(t, pendingCleanupRepo, reserveConfig, expectedCleanupRequest)

	requestKey := GetPendingResourceCleanupKey(IPCleanup, reserveConfig.NetworkID, reserveConfig.AssociatedNCID)
	err := pendingCleanupRepo.DeleteCleanupRequest(requestKey, entity.Etag)
	if err != nil {
		t.Fatalf("Failed to delete the Pending Cleanup Request for requestKey:%s, etag %+v, err %s",
			requestKey,
			entity.Etag,
			err)
	}
}

func TestGetAllPendingCleanupRequests(t *testing.T) {
	pendingCleanupRepo := setup()

	// Try to get all on emtpy collection
	// Get All Pending requests
	requests, err := pendingCleanupRepo.GetAllPendingResourceCleanups()
	if err != nil {
		t.Fatalf("Failed to get all pending requests, err %s",
			err)
	}

	if requests != nil && len(requests) != 0 {
		t.Fatalf("Expected to get an empty list, actual %+v",
			requests)
	}

	request1Key := "nc3"
	request2Key := "nc4"

	ipCleanup1 := map[string][]string{
		"vnet_v1_subnet_s1": {"10.0.0.1/32", "10.0.0.2/32"},
		"vnet_v2_subnet_s2": {"10.0.0.1/32"}}

	cleanupRequest1 := &CleanupContext{
		ToBeDeletedIpAllocations: ipCleanup1,
		Type:                     IPCleanup}

	entity1 := validateCreateOrUpdate(t, pendingCleanupRepo, request1Key, cleanupRequest1, cleanupRequest1)

	ipCleanup2 := map[string][]string{
		"vnet_v2_subnet_s2": {"192.168.0.1/32", "192.168.0.2/32"}}

	cleanupRequest2 := &CleanupContext{
		ToBeDeletedIpAllocations: ipCleanup2,
		Type:                     IPCleanup}

	// Idempotent create, it will not fail again however it will
	// update the etag
	entity2 := validateCreateOrUpdate(t, pendingCleanupRepo, request2Key, cleanupRequest2, cleanupRequest2)

	// Get All Pending requests
	requests, err = pendingCleanupRepo.GetAllPendingResourceCleanups()
	if err != nil {
		t.Fatalf("Failed to get all pending requests, err %s",
			err)
	}

	if len(requests) != 2 {
		t.Fatalf("Failed to get all pending requests, count is not expected, actual %d",
			len(requests))
	}

	for key, entity := range requests {
		if key == request1Key {
			if !reflect.DeepEqual(cleanupRequest1.ToBeDeletedIpAllocations, entity.CleanupContext.ToBeDeletedIpAllocations) {
				t.Fatalf("cleanupRequest not same for Key: %s, entity %s",
					key,
					entity.CleanupContext)
			}
		} else if key == request2Key {
			if !reflect.DeepEqual(cleanupRequest2.ToBeDeletedIpAllocations, entity.CleanupContext.ToBeDeletedIpAllocations) {
				t.Fatalf("cleanupRequest not same for Key: %s, entity %s",
					key,
					entity.CleanupContext)
			}
		} else {
			t.Fatalf("Unexpected entry found Key: %s, entity %+v",
				key,
				entity)
		}
	}

	pendingCleanupRepo.DeleteCleanupRequest(request1Key, entity1.Etag)
	pendingCleanupRepo.DeleteCleanupRequest(request2Key, entity2.Etag)
}

func setup() *PendingResourceCleanupRepo {
	d := testpersistencestore.TestPersistenceStore{}
	d.Init(TableToColumnMap, loggermock.NewDummyLogger())
	return &PendingResourceCleanupRepo{
		data:   &d,
		logger: loggermock.NewDummyLogger()}
}

func validateCreateOrUpdate(t *testing.T, pendingCleanupRepo *PendingResourceCleanupRepo, requestKey string, cleanupRequest *CleanupContext, expectedRequest *CleanupContext) *PendingResourceCleanupEntity {
	t.Logf("Create Or Update Pending Cleanup request for key %s, request %s",
		requestKey, cleanupRequest)
	err := pendingCleanupRepo.CreateOrMergePendingCleanupRequest(requestKey, cleanupRequest)
	if err != nil {
		t.Fatalf("Failed to create the Pending Cleanup Request for requestKey:%s, cleanupRequest %+v",
			requestKey,
			cleanupRequest)
	}

	// Get the request and validate
	return validateRequest(t, pendingCleanupRepo, requestKey, expectedRequest)
}

func validateCreateIPCleanupRequest(t *testing.T, pendingCleanupRepo *PendingResourceCleanupRepo, resConfig *ReservationConfiguration, expectedRequest *CleanupContext) *PendingResourceCleanupEntity {
	t.Logf("Create Or Update Pending IPCleanup request for reservation request %s",
		resConfig)
	err := pendingCleanupRepo.CreateOrMergeIPResourceCleanupRequest(resConfig)
	if err != nil {
		t.Fatalf("Failed to create the Pending Cleanup Request for reservation request  %+v",
			resConfig)
	}

	// Get the request and validate
	requestKey := GetPendingResourceCleanupKey(IPCleanup, resConfig.NetworkID, resConfig.AssociatedNCID)
	return validateRequest(t, pendingCleanupRepo, requestKey, expectedRequest)
}

func validateRequest(t *testing.T, pendingCleanupRepo *PendingResourceCleanupRepo, requestKey string, expectedRequest *CleanupContext) *PendingResourceCleanupEntity {
	t.Log("Validate request")
	entity, err := pendingCleanupRepo.TryGetPendingCleanupRequest(requestKey)
	if err != nil {
		t.Fatalf("Failed to get the Pending Cleanup Request for requestKey:%s",
			requestKey)
	}

	// validate
	if entity.CleanupContext.Type != expectedRequest.Type {
		t.Fatalf("Cleanuprequest, Type is not same, Expected %s, actual %s",
			expectedRequest.Type,
			entity.CleanupContext.Type)
	}

	if !reflect.DeepEqual(entity.CleanupContext.ToBeDeletedIpAllocations, expectedRequest.ToBeDeletedIpAllocations) {
		t.Fatalf("Cleanup request is not same, Expected %s, actual %s",
			expectedRequest,
			entity.CleanupContext)
	}

	return entity
}
