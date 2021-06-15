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

type subnetRepo struct{}

func (*subnetRepo) TryQuerySubnetConfiguration(networkID, subnetName string) (*datamodel.SubnetConfiguration, *api.JsonErr) {
	return datamodel.TryQuerySubnetConfiguration(networkID, subnetName)
}

func (*subnetRepo) UpdateSubnetReservationSet(reservationID, allocatedSubnet string, remove bool, subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr {
	return datamodel.UpdateSubnetReservationSet(reservationID, allocatedSubnet, remove, subnetConfig)
}

func (*subnetRepo) PutSubnetConfiguration(subnetConfig *datamodel.SubnetConfiguration) *api.JsonErr {
	return datamodel.PutSubnetConfiguration(subnetConfig)
}

func (*subnetRepo) TryQueryAllSubnetConfigurations() ([]datamodel.SubnetConfiguration, *api.JsonErr) {
	return datamodel.TryQueryAllSubnetConfigurations()
}
