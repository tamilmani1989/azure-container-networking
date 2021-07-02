package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/macmgr"
	"net/http"
)

type MacAllocator struct {
	PersistNC          PersistNCFunc
	Logger             loggershim.LoggerInterface
	MacManagerClient   macmgr.MacManagerClient
	PendingCleanupRepo *datamodel.PendingResourceCleanupRepo
}

// This function allocate a MAC address
func (m *MacAllocator) ProcessAllocateMAC(nc *datamodel.NetworkContainerEntity) (bool, *api.JsonErr) {
	containerType := nc.Config.GetAssociationInfo().ContainerType
	m.Logger.Debugf("[MacAllocator] Starting MAC allocation. The container type is : %s", containerType)

	allocationId := nc.Config.NetworkContainerID + nc.Config.NetworkID
	allocatMacResponse, err := m.MacManagerClient.AllocateMac(allocationId)

	if err != nil {
		if err.HTTPStatus == http.StatusBadRequest {
			m.Logger.Debugf("[MacAllocator] Failed to allocate MAC. Reason : %s, HttpStatus : %d. DNC will NOT retry.", err.Error, err.HTTPStatus)
			return true, err
		} else {
			m.Logger.Debugf("[MacAllocator] Failed to allocate MAC. Reason : %s, HttpStatus : %d. DNC will retry.", err.Error, err.HTTPStatus)
			return false, err
		}
	}

	allocatedMac := allocatMacResponse.MacAddress
	m.Logger.Debugf("[MacAllocator][Response] Allocate MAC Successful. NC = %s : Mac Address = %+v", nc.Config.NetworkContainerID, allocatedMac)

	nc.Config.State = datamodel.Publish
	nc.Config.MACAddress = allocatedMac
	m.Logger.Debugf("[MacAllocator] Setting MAC address: %s for NC %s", nc.Config.MACAddress, nc.Config.NetworkContainerID)

	err = m.markNCToPublish(nc)
	return false, err
}

// This function deallocates a MAC address.
func (m *MacAllocator) ProcessDeallocateMAC(ncId, networkId string) *api.JsonErr {
	m.Logger.Debugf("[MacAllocator] Resetting MAC address to empty for NC %s.", ncId)

	allocationId := ncId + networkId
	_, err := m.MacManagerClient.DeallocateMac(allocationId)

	if err != nil {
		m.Logger.Debugf("[MacAllocator] Failed to deallocate MAC. Reason : %s, HttpStatus : %d.", err.Error, err.HTTPStatus)
		return err
	}

	m.Logger.Debugf("[MacAllocator][Response] Deallocate MAC Successful. NC = %s.", ncId)
	return err
}

// Create a MAC address clean up request, which will deallcoate the MAC later when NC has been cleaned up from pubsub
func (m *MacAllocator) CreateMacCleanupRequest(nc *datamodel.NetworkContainerEntity) *api.JsonErr {
	return m.PendingCleanupRepo.CreateMACResourceCleanupRequest(nc.Config.NetworkID, nc.Config.NetworkContainerID)
}

// This utility function sets the NC state as published.
func (m *MacAllocator) markNCToPublish(nc *datamodel.NetworkContainerEntity) *api.JsonErr {
	nc.Config.State = datamodel.Publish
	nc.Config.Status = datamodel.Pending
	nc.Config.FailureDetail = api.JsonErr{}
	return m.PersistNC(nc)
}

// This utility function sets the NC state as published.
func (m *MacAllocator) markNCToUnreserve(nc *datamodel.NetworkContainerEntity) *api.JsonErr {
	nc.Config.State = datamodel.UnReserve
	nc.Config.Status = datamodel.Pending
	nc.Config.FailureDetail = api.JsonErr{}
	return m.PersistNC(nc)
}
