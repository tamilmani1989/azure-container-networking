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
	UpdateSubnetReservationSet(networkID, reservationID, allocatedSubnet string, remove bool, subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr
}

// An IPReservationRepo performs CRUD operations on IP reservations
type IPReservationRepo interface {
	TryQueryIPReservation(networkID, reservationID string) (*datamodel.ReservationConfiguration, *api.JsonErr)
	TryQueryIPReservationByKey(key string) (*datamodel.ReservationConfiguration, *api.JsonErr)
	PutIPReservationConfiguration(res *datamodel.ReservationConfiguration) *api.JsonErr
}

// An IPAllocationRepo performs CRUD operations on IP allocations
type IPAllocationRepo interface {
	TryQueryAllIPAllocations(networkID, subnetName string, pageToken interface{}) (map[string]interface{}, bool, interface{}, *api.JsonErr)
	QueryIPAllocationsByAllocationID(networkID, subnetName string, allocIDs []string) (map[string]interface{}, *api.JsonErr)
	AddIPAllocations(networkID, subnetName string, batch map[string]map[string]interface{}) *api.JsonErr
	DeleteIPAllocations(networkID, subnetName string, keys []string) *api.JsonErr
}
