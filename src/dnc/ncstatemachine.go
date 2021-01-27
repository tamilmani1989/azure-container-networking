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
		Reservations:         new(reservationRepo),
		UpdateNodeContainers: datamodel.UpdateNodeConfigurationNetworkContainerSet,
		PersistNC:            datamodel.PersistNCConfiguration,
		Logger:               logger,
	}

	ipAllocator = networkcontainer.IPAllocator{
		Nodes:        new(nodeRepo),
		NCs:          new(ncRepo),
		Subnets:      new(subnetRepo),
		Reservations: new(reservationRepo),
		Allocations:  datamodel.AllocRepo,
		Logger:       logger,
		MaxBatchSize: config.DataStoreSettings.MaxBatchSize,
	}

	macAllocator = networkcontainer.MacAllocator{
		PersistNC: datamodel.PersistNCConfiguration,
		Logger:    logger,
	}

	if config.DNCSettings.Managed {
		logger.Debugf("[mDNC] batch reservation enabled")
		resDispatcher = networkcontainer.NewIPReservationDispatcher(
			networkcontainer.DispatcherDependencies{
				Logger:       logger,
				Reservations: new(reservationRepo),
				Allocator:    &ipAllocator,
				Subnets:      new(subnetRepo),
			},
			sessionManager,
			config.DNCSettings.MaxIPReservationBatchSize,
		)

		resDispatcher.Run(context.Background())
	}
}
