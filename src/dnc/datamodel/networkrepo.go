package datamodel

import (
	"dnc/api"
)
//TODO: Change from wrapper to having the actual implementation https://msazure.visualstudio.com/One/_workitems/edit/9570989
type NetworkRepo struct{}

func (*NetworkRepo) TryQueryNetworkConfiguration(networkID string) (*NetworkConfiguration, *api.JsonErr) {
	return TryQueryNetworkConfiguration(networkID)
}

func (*NetworkRepo) GetAllNetworkIDs() (map[string]bool, error) {
	return GetAllNetworkIDs()
}

func (*NetworkRepo) TryQueryAllNetworks() ([]string, error) {
	return TryQueryAllNetworks()
}

func (*NetworkRepo) PersistNetworkConfiguration(networkConfig *NetworkConfiguration) *api.JsonErr {
	return PersistNetworkConfiguration(networkConfig)
}
