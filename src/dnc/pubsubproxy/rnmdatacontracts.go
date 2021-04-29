// +build enablePubsubProxy

package pubsubproxy

import (
	"dnc/datamodel"
	"errors"
	"fmt"
	"strings"
	"time"
)

// implementation of madariDataContracts interface. This struct
// does handling of vnet/subnet data that is published by RNM
type rnmDataContracts struct {
}

// TODO: Use retry package here for retries and back off
func (r *rnmDataContracts) rehydrateSubscriptions() {
	logDebugf("Rehydrating vnet subscriptions")
	for {
		networksMap, err := networkRepo.GetAllNetworkIDs()
		if err == nil {
			for networkKey := range networksMap {
				networkID := datamodel.GetNetworkIdFromKey(networkKey)
				RegisterVnet(networkID)
			}
			// Exit after rehydrating to all vnet subscriptions
			return
		} else {
			logErrorf("Failed to rehydrate vnets. Error: %v", err)
			time.Sleep(time.Second * 1)
		}
	}
}

func (r *rnmDataContracts) getCachedSubnetConfig(
	vnetId string,
	subnetName string) (cachedSubnetConfig, error) {

	s := getActiveSubscriber()

	vnetConfig, found := s.cachedVnetConfigMap.Load(vnetId)
	if !found {
		return cachedSubnetConfig{}, fmt.Errorf("Configuration not found for VnetId: %s", vnetId)
	}

	for _, subnetConfig := range vnetConfig.(cachedVnetConfig).subnetConfigs {
		if strings.EqualFold(subnetConfig.subnetName, subnetName) {
			return subnetConfig, nil
		}
	}

	return cachedSubnetConfig{}, fmt.Errorf("Subnet: %s not found for VnetId: %s", subnetName, vnetId)
}

func (r *rnmDataContracts) deleteCachedSubnetConfig(
	vnetId, subnetId string) error {
	return errors.New("delete subnet config is not supported in RNM data contracts")
}

func (r *rnmDataContracts) updateCachedSubnetConfig(
	vnetId string,
	subnetId string,
	subnetConfig *SubnetConfiguration) error {
	return errors.New("update subnet config is not supported in RNM data contracts")
}

func (r *rnmDataContracts) subscribeCachedSubnetConfig(
	vnetId, subnetId string) error {
	return errors.New("subscribe subnet config is not supported in RNM data contracts")
}

func (r *rnmDataContracts) unsubscribeCachedSubnetConfigs() error {
	return errors.New("subscribe subnet config is not supported in RNM data contracts")
}

func (r *rnmDataContracts) getSubnetPartitionKey(vnetId, subnetId string) string {
	return ""
}

func (r *rnmDataContracts) getSubnetRelativePath(vnetId, subnetId string) string {
	return ""
}