package main

import (
	"context"
	"dnc/datamodel"
	"dnc/networkcontainer"
)

var (
	preparer      networkcontainer.Preparer
	ipAllocator   networkcontainer.IPAllocator
	macAllocator  networkcontainer.MacAllocator
	resDispatcher *networkcontainer.IPReservationDispatcher
)

// todo: move other processing stages here
func initNetworkContainerStateMachine() {
	preparer = networkcontainer.Preparer{
		CNS:                  cnsInterface,
		Locks:                sessionManager,
		Reservations:         datamodel.IpReservationRepo,
		UpdateNodeContainers: datamodel.UpdateNodeConfigurationNetworkContainerSet,
		PersistNC:            datamodel.PersistNCConfiguration,
		Logger:               logger,
	}

	ipAllocator = networkcontainer.IPAllocator{
		Nodes:                       new(nodeRepo),
		NCs:                         new(ncRepo),
		Subnets:                     new(subnetRepo),
		Reservations:                datamodel.IpReservationRepo,
		Allocations:                 datamodel.AllocRepo,
		PendingCleanupRepo:          datamodel.PendingCleanupRepo,
		Logger:                      logger,
		MaxBatchSize:                config.DataStoreSettings.MaxBatchSize,
		DisableResMapInSubnetConfig: config.DNCSettings.DisableReservationMapInSubnetConfig,
		PubsubProxyEnabled:          config.PubSubProxySettings.EnablePubSubProxy,
	}

	macAllocator = networkcontainer.MacAllocator{
		PersistNC: datamodel.PersistNCConfiguration,
		Logger:    logger,
	}

	if config.FeatureFlagSettings.EnableBatchAllocationProcessing {
		logger.Debugf("batch allocation processing enabled")
		resDispatcher = networkcontainer.NewIPReservationDispatcher(
			networkcontainer.DispatcherDependencies{
				Logger:       logger,
				Reservations: datamodel.IpReservationRepo,
				Allocator:    &ipAllocator,
				Subnets:      new(subnetRepo),
			},
			config.FeatureFlagSettings.MaxIPReservationBatchSize,
			false, // todo: pass this from config
			config.PubSubProxySettings.EnablePubSubProxy,
		)

		resDispatcher.Run(context.Background())
	}
}
