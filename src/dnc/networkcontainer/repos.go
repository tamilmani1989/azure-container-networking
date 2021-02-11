package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
)

// A NodeRepo performs CRUD operations on network nodes
type NodeRepo interface {
	QueryNodeConfiguration(nodeID string) (*datamodel.NodeConfiguration, *api.JsonErr)
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
}

// An IPAllocationRepo performs CRUD operations on IP allocations
type IPAllocationRepo interface {
	TryQueryAllIPAllocations(networkID, subnetName string) ([]datamodel.IPAllocationEntity, *api.JsonErr)
	QueryIPAllocationsByAllocationID(networkID, subnetName string, allocIDs []string) ([]datamodel.IPAllocationEntity, *api.JsonErr)
	QueryIPAllocationsByToBeDeletedOwner(networkID, subnetName string, tobeDeletedOwnerId string) ([]datamodel.IPAllocationEntity, *api.JsonErr)
	// todo: mutation methods should take a batch of Allocation rows, not generic maps
	// this will decouple implementations from having to know the map keys, and just use the specific row struct
	AddIPAllocations(networkID, subnetName string, batch map[string]map[string]interface{}) *api.JsonErr
	UpdateIPAllocations(networkID, subnetName string, batch map[string]map[string]interface{}) *api.JsonErr
	DeleteIPAllocations(networkID, subnetName string, keys []string) *api.JsonErr
	DeleteIPAllocationEntityWithEtag(entity datamodel.IPAllocationEntity) *api.JsonErr
}
