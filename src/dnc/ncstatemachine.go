package main

import (
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/networkcontainer"
)

var (
	preparer    networkcontainer.Preparer
	ipAllocator networkcontainer.IPAllocator
)

// todo: move other processing stages here
func initNetworkContainerStateMachine() {
	preparer = networkcontainer.Preparer{
		CNS:                  cnsInterface,
		Locks:                sessionManager,
		PersistReservation:   datamodel.PutIPReservationConfiguration,
		UpdateNodeContainers: datamodel.UpdateNodeConfigurationNetworkContainerSet,
		PersistNC:            datamodel.PersistNCConfiguration,
		Logger:               logger,
	}

	ipAllocator = networkcontainer.IPAllocator{
		Nodes:                  new(nodeRepo),
		NCs:                    new(ncRepo),
		Subnets:                new(subnetRepo),
		Reservations:           new(reservationRepo),
		Allocations:            new(allocationRepo),
		EnableSubnetDelegation: datamodel.IsFeatureSupported(configuration.SharedSubnetDelegation),
		Logger:                 logger,
	}
}
