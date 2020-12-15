package main

import (
	"dnc/api"
	"dnc/datamodel"
)

/*
  This file contains types that encapsulate repo-like functionality for different entities in DNC.
  Different packages can define their dependencies as interfaces, and these types can implement them.
  This is still a temporary solution, as we continue to work towards having more explicit dependency injection.
*/

type nodeRepo struct{}

func (*nodeRepo) UpdateNodeContainers(nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool, ncLimit int) *api.JsonErr {
	return datamodel.UpdateNodeConfigurationNetworkContainerSet(nodeID, networkID, ncID, reservationID, containerType, updateVlan, remove, ncLimit)
}

func (*nodeRepo) QueryNodeConfiguration(nodeID string) (*datamodel.NodeConfiguration, *api.JsonErr) {
	return datamodel.QueryNodeConfiguration(nodeID)
}

func (*nodeRepo) UpdateNodeConstrainedReservationSetByNodeID(nodeID, networkID, reservationID string, remove bool) *api.JsonErr {
	return datamodel.UpdateNodeConstrainedReservationSetByNodeID(nodeID, networkID, reservationID, remove)
}

type ncRepo struct{}

func (*ncRepo) TryQueryNetworkContainer(networkID, ncID string) (*datamodel.NCConfiguration, *api.JsonErr) {
	return datamodel.TryQueryNetworkContainer(networkID, ncID)
}

func (*ncRepo) PersistNCConfiguration(ncConfiguration *datamodel.NCConfiguration, addToPendingQueue bool) *api.JsonErr {
	return datamodel.PersistNCConfiguration(ncConfiguration, addToPendingQueue)
}

type subnetRepo struct{}

func (*subnetRepo) TryQuerySubnetConfiguration(networkID, subnetName string) (*datamodel.SubnetConfiguration, *api.JsonErr) {
	return datamodel.TryQuerySubnetConfiguration(networkID, subnetName)
}

func (*subnetRepo) UpdateSubnetReservationSet(networkID, reservationID, allocatedSubnet string, remove bool, subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr {
	return datamodel.UpdateSubnetReservationSet(networkID, reservationID, allocatedSubnet, remove, subnetConfig)
}

func (*subnetRepo) PutSubnetConfiguration(subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr {
	return datamodel.PutSubnetConfiguration(subnetConfig)
}

type reservationRepo struct{}

func (*reservationRepo) TryQueryIPReservation(networkID, reservationID string) (*datamodel.ReservationConfiguration, *api.JsonErr) {
	return datamodel.TryQueryIPReservation(networkID, reservationID)
}

func (*reservationRepo) TryQueryIPReservationByKey(key string) (*datamodel.ReservationConfiguration, *api.JsonErr) {
	return datamodel.TryQueryIPReservationByKey(key)
}

func (*reservationRepo) PutIPReservationConfiguration(res *datamodel.ReservationConfiguration) *api.JsonErr {
	return datamodel.PutIPReservationConfiguration(res)
}

func (r *reservationRepo) GetReservations(networkID string, reservationIDs []string) ([]datamodel.ReservationConfiguration, *api.JsonErr) {
	return datamodel.QueryIPReservations(networkID, reservationIDs)
}

func (r *reservationRepo) UpdateReservations(reservations []datamodel.ReservationConfiguration) *api.JsonErr {
	return datamodel.UpdateIPReservations(reservations)
}

type allocationRepo struct{}

func (r *allocationRepo) QueryIPAllocationsByAllocationID(networkID, subnetName string, allocIDs []string) (map[string]interface{}, *api.JsonErr) {
	return datamodel.QueryIPAllocationsByAllocationID(networkID, subnetName, allocIDs)
}

func (*allocationRepo) TryQueryAllIPAllocations(networkID, subnetName string, pageToken interface{}) (map[string]interface{}, bool, interface{}, *api.JsonErr) {
	return datamodel.TryQueryAllIPAllocations(networkID, subnetName, pageToken)
}

func (*allocationRepo) AddIPAllocations(networkID, subnetName string, batch map[string]map[string]interface{}) *api.JsonErr {
	return datamodel.AddIPAllocations(networkID, subnetName, batch)
}

func (*allocationRepo) DeleteIPAllocations(networkID, subnetName string, keys []string) *api.JsonErr {
	return datamodel.DeleteIPAllocations(networkID, subnetName, keys)
}
