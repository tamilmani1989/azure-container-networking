package networkcontainer

import (
	"dnc/datamodel"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggermock"
	"net"
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
	rows, err := generateIPAllocationEntitiesForSubnet(baseSubnet)
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
