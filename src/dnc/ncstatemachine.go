package main

import (
	"context"
	"dnc/datamodel"
	"dnc/macmgr"
	"dnc/networkcontainer"
)

var (
	preparer         networkcontainer.Preparer
	ipAllocator      networkcontainer.IPAllocator
	macAllocator     networkcontainer.MacAllocator
	macManagerClient macmgr.MacManagerClient
	resDispatcher    *networkcontainer.IPReservationDispatcher
)

// todo: move other processing stages here
func initNetworkContainerStateMachine() {
	preparer = networkcontainer.Preparer{
		CNS:                  cnsInterface,
		Locks:                sessionManager,
		Reservations:         datamodel.IpReservationRepo,
		UpdateNodeContainers: datamodel.NodeConfigRepo.UpdateNodeConfigurationNetworkContainerSet,
		PersistNC:            datamodel.NcRepo.PersistNetworkContainerData,
		Logger:               logger,
	}

	ipAllocator = networkcontainer.IPAllocator{
		Nodes:                       datamodel.NodeConfigRepo,
		NCs:                         datamodel.NcRepo,
		Subnets:                     new(subnetRepo),
		Reservations:                datamodel.IpReservationRepo,
		Allocations:                 datamodel.AllocRepo,
		PendingCleanupRepo:          datamodel.PendingCleanupRepo,
		Logger:                      logger,
		MaxBatchSize:                config.DataStoreSettings.MaxBatchSize,
		DisableResMapInSubnetConfig: config.DNCSettings.DisableReservationMapInSubnetConfig,
		PubsubProxyEnabled:          config.PubSubProxySettings.EnablePubSubProxy,
		UsePrepopulatedAllocations:  config.FeatureFlagSettings.UsePrepopulatedIPAllocations,
	}

	macManagerClient = &macmgr.MacManagerConnection{
		HttpClient:        macAllocatorHttpClient,
		MacManagerSetting: &config.MacManagerSettings,
	}

	macAllocator = networkcontainer.MacAllocator{
		PersistNC:        datamodel.NcRepo.PersistNetworkContainerData,
		Logger:           logger,
		MacManagerClient: macManagerClient,
	}

	if config.FeatureFlagSettings.EnableBatchAllocationProcessing {
		dispatchCfg := networkcontainer.DispatcherConfig{
			Logger:                     logger,
			Subnets:                    new(subnetRepo),
			Reservations:               datamodel.IpReservationRepo,
			Allocator:                  &ipAllocator,
			HandlerMaxBatchSize:        config.FeatureFlagSettings.MaxIPReservationBatchSize,
			UsePrepopulatedAllocations: config.FeatureFlagSettings.UsePrepopulatedIPAllocations,
		}

		resDispatcher = networkcontainer.NewIPReservationDispatcher(dispatchCfg)

		batchSettingsFmt := "batch allocation processing enabled with settings: " +
			"datastore max batch size %d, " +
			"max nc request batch size per subnet: %d, " +
			"use prepopulated allocation strategy: %v"
		logger.Debugf(batchSettingsFmt, ipAllocator.MaxBatchSize, dispatchCfg.HandlerMaxBatchSize, dispatchCfg.UsePrepopulatedAllocations)

		resDispatcher.Run(context.Background())
	}
}
