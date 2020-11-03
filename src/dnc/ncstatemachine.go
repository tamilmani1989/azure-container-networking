package main

import (
	"dnc/datamodel"
	"dnc/networkcontainer"
)

var (
	preparer     networkcontainer.Preparer
	ipAllocator  networkcontainer.IPAllocator
	macAllocator networkcontainer.MacAllocator
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
		Nodes:            new(nodeRepo),
		NCs:              new(ncRepo),
		Subnets:          new(subnetRepo),
		Reservations:     new(reservationRepo),
		Allocations:      new(allocationRepo),
		IsFeatureEnabled: datamodel.IsFeatureSupported,
		Logger:           logger,
	}

	macAllocator = networkcontainer.MacAllocator{
		PersistNC: datamodel.PersistNCConfiguration,
		Logger:    logger,
	}
}
