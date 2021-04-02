package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggermock"
	"net"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllocateFromPrepopulated(t *testing.T) {
	_, baseSubnet, _ := net.ParseCIDR("10.1.0.0/16")
	vnetID := "vnet1"
	subnet := "subnet1"

	datastore, err := getPrepopulatedDatastore(baseSubnet, vnetID, subnet)
	require.NoError(t, err)

	// todo: having to mock this is not easy to deduce. test datastore should have a proper implementation.
	datastore.QueryFilterTopNFunc = func(_ string, properties map[string]interface{}) bool {
		return properties[datamodel.IPAllocationProperty].(string) == ""
	}

	sh := prepopulatedStrategy{
		allocator: &IPAllocator{
			Allocations:  &datamodel.AllocationRepo{Datastore: datastore},
			Logger:       loggermock.NewDummyLogger(),
			MaxBatchSize: 100,
		},
		subnetInfo: subnetInfo{
			vnetID:     vnetID,
			subnetName: subnet,
			baseSubnet: baseSubnet,
		},
		logger: loggermock.NewDummyLogger(),
	}

	reservationCount := 100
	secondaryIPCount := 32

	set, _ := generateBatch(reservationCount, secondaryIPCount, vnetID)
	reservations := set.getAllReservations()

	result, err := sh.allocateFromPrepopulated(reservations)
	require.NoError(t, err)

	assert.Len(t, result.completed, reservationCount)
	assert.Len(t, result.failed, 0)

	expectedAllocationCount := reservationCount * (secondaryIPCount + 1)
	assert.Len(t, result.entitiesToPersist, expectedAllocationCount)

	index := buildIndex(result.entitiesToPersist)
	for _, completedReservation := range result.completed {
		allocationMap := index[completedReservation.ReservationID]
		require.NotNil(t, allocationMap, "expected updated allocations for reservation")
		require.NotEmpty(t, completedReservation.AllocatedSubnet, "expected reservation to contain a primary IP")

		expectedPrimaryIP := allocationMap[completedReservation.ReservationID]
		require.Equal(t, expectedPrimaryIP, completedReservation.AllocatedSubnet, "expected updated primary allocation to match primary IP in reservation")

		for allocationID, secondaryIP := range completedReservation.SecondaryAllocations {
			require.NotEmpty(t, secondaryIP, "expected secondary allocation to contain a secondary IP")
			expectedSecondaryIP := allocationMap[allocationID]
			require.Equal(t, expectedSecondaryIP, secondaryIP, "expected updated secondary allocation to match secondary IP")
		}
	}
}

func TestAllocateFromPrepopulatedErrors(t *testing.T) {
	_, baseSubnet, _ := net.ParseCIDR("10.1.0.0/24")
	vnetID := "vnet1"
	subnet := "subnet1"

	datastore, err := getPrepopulatedDatastore(baseSubnet, vnetID, subnet)
	require.NoError(t, err)

	// todo: having to mock this is not easy to deduce. test datastore should have a proper implementation.
	datastore.QueryFilterTopNFunc = func(_ string, properties map[string]interface{}) bool {
		return properties[datamodel.IPAllocationProperty].(string) == ""
	}

	sh := prepopulatedStrategy{
		allocator: &IPAllocator{
			Allocations:  &datamodel.AllocationRepo{Datastore: datastore},
			Logger:       loggermock.NewDummyLogger(),
			MaxBatchSize: 100,
		},
		subnetInfo: subnetInfo{
			vnetID:     vnetID,
			subnetName: subnet,
			baseSubnet: baseSubnet,
		},
		logger: loggermock.NewDummyLogger(),
	}

	reservationCount := 10
	secondaryIPCount := 32

	set, _ := generateBatch(reservationCount, secondaryIPCount, vnetID)
	reservations := set.getAllReservations()

	result, err := sh.allocateFromPrepopulated(reservations)
	require.NoError(t, err)

	// 256 IPs in /24. 5 are azure reserved, so 251 available IPs.
	// each reservation requests 33 IPs (1 primary + 32 secondaries).
	// only 7 (231 IPs) can be completely allocated, rest will be failed.
	// from failed, one will be partially filled.

	assert.Len(t, result.completed, 7)
	assert.Len(t, result.failed, 3)

	expectedAllocationCount := 251
	assert.Len(t, result.entitiesToPersist, expectedAllocationCount)

	index := buildIndex(result.entitiesToPersist)
	for _, completedReservation := range result.completed {
		assert.True(t, completedReservation.Status == datamodel.Completed)
		assert.Empty(t, completedReservation.FailureDetail)

		allocationMap := index[completedReservation.ReservationID]
		assert.NotNil(t, allocationMap, "expected updated allocations for reservation")
		assert.NotEmpty(t, completedReservation.AllocatedSubnet, "expected reservation to contain a primary IP")

		expectedPrimaryIP := allocationMap[completedReservation.ReservationID]
		assert.Equal(t, expectedPrimaryIP, completedReservation.AllocatedSubnet, "expected updated primary allocation to match primary IP in reservation")

		for allocationID, secondaryIP := range completedReservation.SecondaryAllocations {
			assert.NotEmpty(t, secondaryIP, "expected secondary allocation to contain a secondary IP")
			expectedSecondaryIP := allocationMap[allocationID]
			assert.Equal(t, expectedSecondaryIP, secondaryIP, "expected updated secondary allocation to match secondary IP")
		}
	}

	for _, failedReservation := range result.failed {
		assert.True(t, failedReservation.Status == datamodel.Failed)
		assert.Equal(t, api.SubnetFull, failedReservation.FailureDetail.Code)
		assert.Equal(t, http.StatusBadRequest, failedReservation.FailureDetail.HTTPStatus)

		allocationMap, ok := index[failedReservation.ReservationID]
		if !ok { // not the partially allocated reservation
			assert.Empty(t, failedReservation.AllocatedSubnet)
			for _, secondaryIP := range failedReservation.SecondaryAllocations {
				assert.Empty(t, secondaryIP)
			}
			continue
		}

		assert.NotEmpty(t, failedReservation.AllocatedSubnet, "expected reservation to contain a primary IP")
		expectedPrimaryIP := allocationMap[failedReservation.ReservationID]
		assert.Equal(t, expectedPrimaryIP, failedReservation.AllocatedSubnet, "expected updated primary allocation to match primary IP in reservation")

		for allocationID, secondaryIP := range failedReservation.SecondaryAllocations {
			expectedSecondaryIP := allocationMap[allocationID]
			assert.Equal(t, expectedSecondaryIP, secondaryIP, "expected updated secondary allocation to match secondary IP")
		}
	}
}

func buildIndex(entities []datamodel.IPAllocationEntity) map[string]map[string]string { // reservation id -> allocation id -> ip address
	m := make(map[string]map[string]string)
	for _, entity := range entities {
		ipAddress := datamodel.DecodeIPAllocationKey(entity.RowKey)

		reservationID := datamodel.GetReservationIDFromKey(entity.IPAllocation)
		if _, ok := m[reservationID]; !ok {
			m[reservationID] = make(map[string]string)
		}

		m[reservationID][entity.AllocationID] = ipAddress
	}
	return m
}

func getPrepopulatedDatastore(baseSubnet *net.IPNet, vnetID, subnet string) (*testpersistencestore.TestPersistenceStore, error) {
	rows, err := GenerateIPAllocationEntitiesForSubnet(baseSubnet)
	if err != nil {
		return nil, err
	}
	var entities []storage.Entity
	for _, row := range rows {
		entities = append(entities, row.ToStorageEntity(vnetID, subnet))
	}

	var datastore testpersistencestore.TestPersistenceStore
	datastore.Init(datamodel.TableToColumnMap, loggermock.NewDummyLogger())
	datastore.Seed(map[string][]storage.Entity{datamodel.IPAllocationsTableName: entities})

	return &datastore, nil
}
