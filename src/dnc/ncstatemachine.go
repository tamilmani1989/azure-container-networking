package main

import (
	"dnc/datamodel"
	"dnc/networkcontainer"
)

var preparer networkcontainer.Preparer

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
}
