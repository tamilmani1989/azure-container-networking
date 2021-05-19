// +build enablePubsubProxy

package pubsubproxy

import (
	"context"
	"dnc/retry"
	"dnc/utils"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// implementation of madariDataContracts interface. This struct
// does handling of vnet/subnet data that is published by RNM
type rncDataContracts struct {
	cachedVnetSubnetMap sync.Map
}

const (
	// PubSub paths
	rncVnetPartitionKeyFormat   = "/virtualnetworks/%s"
	rncSubnetRelativePathFormat = "/subnets/%s/config"
)

func (s *rncDataContracts) rehydrateSubscriptions() {
	logDebugf("Rehydrating subnet subscriptions via RNC path")
	retrier := retry.Retrier{
		Attempts: math.MaxInt32,
		Delay:    time.Second,
		MaxDelay: 60 * time.Second,
		ExpBackoff: true,
	}

	retrier.Do(context.Background(), func() error {
		subnetConfigs, err := subnetRepo.TryQueryAllSubnetConfigurations()
		if err != nil {
			logErrorf("Failed to rehydrate subnets. Error: %v", err)
			return err
		}

		for _, config := range subnetConfigs {
			// if subnet name is not valid uuid, then skip
			if !utils.IsValidUUID(config.SubnetName) {
				logDebugf("Skip rehydrating subnet %s as it is not valid guid", config.SubnetName)
				continue
			}

			RegisterSubnet(config.NetworkID, config.SubnetName)
		}

		// Return nil after rehydrating to all subnet subscriptions
		return nil
	})
}

func (r *rncDataContracts) handleUpdateNotification(partitionKey, relativePath string, data []byte) {
	logDebugf("rncDataContracts: handleMadariNotification for partitionKey:%s, path:%s", partitionKey, relativePath)
	// Handle the notification for the data update
	if strings.HasPrefix(relativePath, "/subnets") &&
	   strings.HasSuffix(relativePath, "/config") {
		// If the subscribeStatusError channel is setup, send nil error since
		// the notification is received for that path
		path := partitionKey + relativePath
		getActiveSubscriber().sendSubscribeStatusError(path, nil)

		vnetId, verr := r.getVnetIdFromPartitionKey(partitionKey)
		if verr != nil {
			logDebugf("handleUpdateNotification: %s", verr)
			return
		}

		subnetId, serr := r.getSubnetIdFromRelativePath(relativePath)
		if serr != nil {
			logDebugf("handleUpdateNotification: %s", serr)
			return
		}

		subnetUpdate := &subnetJob{
			context: jobContext{
				id:              createSubnetJobId(vnetId, subnetId),
				jobType:         SubnetUpdate,
				createdDateTime: time.Now().UTC(),
				retryInterval:   baseRetryInterval,
			},
			vnetId:   vnetId,
			subnetId: subnetId,
			subnetData: data,
		}
		workerPool().scheduleJob(subnetUpdate)
	}
}

func (s *rncDataContracts) getCachedSubnetConfig(
	vnetId string,
	subnetId string) (cachedSubnetConfig, error) {

	key := fmt.Sprintf("%s_%s", vnetId, subnetId)
	config, found := s.cachedVnetSubnetMap.Load(key)
	if !found {
		return cachedSubnetConfig{}, fmt.Errorf("Configuration not found for VnetId: %s, Subnet:%s", vnetId, subnetId)
	}

	return config.(cachedSubnetConfig), nil
}

func (s *rncDataContracts) deleteCachedSubnetConfig(
	vnetId, subnetId string) error {
	key := fmt.Sprintf("%s_%s", vnetId, subnetId)
	s.cachedVnetSubnetMap.Delete(key)
	return nil
}

func (s *rncDataContracts) updateCachedSubnetConfig(
	vnetId string,
	subnetId string,
	subnetConfig *SubnetConfiguration) error {

	cachedSubnetConfig := cachedSubnetConfig{
		subnetName:    subnetConfig.SubnetName,
		addressPrefix: subnetConfig.AddressPrefix,
		expectedToken: subnetConfig.ExpectedToken,
		owningService: subnetConfig.OwningService,
		subnetVersion: subnetConfig.SubnetVersion,
        isSubscribed: true,
	}

	key := fmt.Sprintf("%s_%s", vnetId, subnetId)
	s.cachedVnetSubnetMap.Store(key, cachedSubnetConfig)
	return nil
}

func (s *rncDataContracts) subscribeCachedSubnetConfig(
	vnetId, subnetId string) error {
	subnetConfig, err := s.getCachedSubnetConfig(vnetId, subnetId)
	if err == nil {
		// Mark the existing vnet as subscribed
		subnetConfig.isSubscribed = true
		s.cachedVnetSubnetMap.Store(vnetId, subnetConfig)
		return nil
	}

	// If subnet config doesn't exist, create an entry as a subscribed vnet
	key := fmt.Sprintf("%s_%s", vnetId, subnetId)
	s.cachedVnetSubnetMap.Store(key, cachedSubnetConfig{
		isSubscribed: true,
	})

	return nil
}

func (s *rncDataContracts) unsubscribeCachedSubnetConfigs() error {
	s.cachedVnetSubnetMap.Range(func(key interface{}, config interface{}) bool {
		mapKey := key.(string)
		if strings.Contains(mapKey, "_") {
			parts := strings.Split(mapKey, "_")
			subnetConfig, err := s.getCachedSubnetConfig(parts[0], parts[1])
			if err == nil {
				subnetConfig.isSubscribed = false
				s.cachedVnetSubnetMap.Store(mapKey, subnetConfig)
			}
		} else {
			// This is no-op for now. In future if we store other data
			// in this map, this can be implemented
			logDebugf("unsubscribeCachedSubnetConfigs does not support non-subnet paths")
		}
		return true
	})

	return nil
}

func (s *rncDataContracts) getSubnetPartitionKey(vnetId, subnetId string) string {
	return fmt.Sprintf(rncVnetPartitionKeyFormat, vnetId)
}

func (s *rncDataContracts) getSubnetRelativePath(vnetId, subnetId string) string {
	return fmt.Sprintf(rncSubnetRelativePathFormat, subnetId)
}

func (s *rncDataContracts) getVnetIdFromPartitionKey(partitionKey string) (string, error) {
	// Example relative path : /virtualnetworks/db8baa97-1baf-4d16-aeb4-b2716665bd22
	parts := strings.Split(partitionKey, "/")
	if  len(parts) == 3 {
		return parts[2], nil
	}

	return "", fmt.Errorf("rncDataContracts. Invalid subnet partitionKey:%s", partitionKey)
}

func (s *rncDataContracts) getSubnetIdFromRelativePath(relativePath string) (string, error) {
	// Example relative path : /subnets/cb8baa97-1baf-4d16-aeb4-b2716665bd22/config
	parts := strings.Split(relativePath, "/")
	if  len(parts) == 4 {
		return parts[2], nil
	}

	return "", fmt.Errorf("rncDataContracts. Invalid subnet relativePath:%s", relativePath)
}