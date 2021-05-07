package datamodel

import "dnc/api"

type SubnetRepo struct{}

func (*SubnetRepo) TryQuerySubnetConfiguration(networkID, subnetName string) (*SubnetConfiguration, *api.JsonErr) {
	return TryQuerySubnetConfiguration(networkID, subnetName)
}

func (*SubnetRepo) UpdateSubnetReservationSet(reservationID, allocatedSubnet string, remove bool, subnetConfig *SubnetConfiguration) *api.JsonErr {
	return UpdateSubnetReservationSet(reservationID, allocatedSubnet, remove, subnetConfig)
}

func (*SubnetRepo) PutSubnetConfiguration(subnetConfig *SubnetConfiguration) *api.JsonErr {
	return PutSubnetConfiguration(subnetConfig)
}

func (*SubnetRepo) TryQueryAllSubnetConfigurations() ([]SubnetConfiguration, *api.JsonErr) {
	return TryQueryAllSubnetConfigurations()
}
