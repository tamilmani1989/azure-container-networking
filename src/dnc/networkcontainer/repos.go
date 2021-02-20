package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
)

// A NodeRepo performs CRUD operations on network nodes
type NodeRepo interface {
	TryQueryNodeConfiguration(nodeID string) (*datamodel.NodeConfiguration, *api.JsonErr)
	QueryNodeConfiguration(nodeID string) (*datamodel.NodeConfiguration, *api.JsonErr)
	UpdateNodeConstrainedReservationSet(networkID, reservationID string, remove bool, nodeConfiguration *datamodel.NodeConfiguration) *api.JsonErr
	UpdateNodeConstrainedReservationSetByNodeID(nodeID, networkID, reservationID string, remove bool) *api.JsonErr
	UpdateNodeContainers(nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool, ncLimit int) *api.JsonErr
}

// A NetworkContainerRepo performs CRUD operations on network containers
type NCRepo interface {
	TryQueryNetworkContainer(networkID, ncID string) (*datamodel.NCConfiguration, *api.JsonErr)
	PersistNCConfiguration(ncConfiguration *datamodel.NCConfiguration, addToPendingQueue bool) *api.JsonErr
}

// A SubnetRepo performs CRUD operations on subnets
type SubnetRepo interface {
	TryQuerySubnetConfiguration(networkID, subnetName string) (*datamodel.SubnetConfiguration, *api.JsonErr)
	UpdateSubnetReservationSet(reservationID, allocatedSubnet string, remove bool, subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr
	PutSubnetConfiguration(subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr
}

// An IPReservationRepo performs CRUD operations on IP reservations
type IPReservationRepo interface {
	TryQueryIPReservation(networkID, reservationID string) (datamodel.ReservationRow, *api.JsonErr)
	TryQueryIPReservationByKey(key string) (datamodel.ReservationRow, *api.JsonErr)
	PutIPReservationConfiguration(res *datamodel.ReservationConfiguration) *api.JsonErr
	UpdateReservations(reservations []datamodel.ReservationConfiguration) *api.JsonErr
	DeleteReservation(reservationKey string) *api.JsonErr
}

// todo: mutation methods should take a batch of Allocation rows, not generic maps
// this will decouple implementations from having to know the map keys, and just use the specific row struct
// An IPAllocationRepo performs CRUD operations on IP allocations
type IPAllocationRepo interface {
	TryQueryIPAllocation(networkID, subnetName, rowKey string) (datamodel.IPAllocationEntity, *api.JsonErr)
	TryQueryAllIPAllocations(networkID, subnetName string) ([]datamodel.IPAllocationEntity, *api.JsonErr)
	QueryIPAllocationsByReservationKey(networkID, subnetName string, reservationKey string) ([]datamodel.IPAllocationEntity, *api.JsonErr)
	QueryIPAllocationsByAllocationID(networkID, subnetName string, allocIDs []string) ([]datamodel.IPAllocationEntity, *api.JsonErr)
	QueryIPAllocationsByToBeDeletedOwner(networkID, subnetName string, tobeDeletedOwnerId string) ([]datamodel.IPAllocationEntity, *api.JsonErr)
	QueryUnreservedAllocations(networkID, subnetName string, count int) ([]datamodel.IPAllocationEntity, *api.JsonErr)
	QueryIPAllocationsByReservationKeys(networkID, subnetName string, reservationIDs []string) ([]datamodel.IPAllocationEntity, *api.JsonErr)
	// deprecated: use InsertIPAllocationEntities
	AddIPAllocations(networkID, subnetName string, batch map[string]map[string]interface{}) *api.JsonErr
	InsertIPAllocationEntities(networkID, subnetName string, entities []datamodel.IPAllocationEntity) *api.JsonErr
	UpdateIPAllocationEntities(networkID, subnetName string, ipAllocationEntities []datamodel.IPAllocationEntity) *api.JsonErr
	DeleteIPAllocation(networkID, subnetName, allocatedIPKey string) *api.JsonErr
	DeleteIPAllocations(networkID, subnetName string, keys []string) *api.JsonErr
	DeleteIPAllocationEntityWithEtag(entity datamodel.IPAllocationEntity) *api.JsonErr
}
