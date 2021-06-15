package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/macmgr"
	"net/http"
)

type MacAllocator struct {
	PersistNC        PersistNCFunc
	Logger           loggershim.LoggerInterface
	MacManagerClient macmgr.MacManagerClient
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
func (m *MacAllocator) ProcessDeallocateMAC(nc *datamodel.NetworkContainerEntity) (bool, *api.JsonErr) {
	if nc.Config.MACAddress == "" {
		m.Logger.Debugf("[MacAllocator] MAC address is already empty. No need to deallocate.")
	} else {
		m.Logger.Debugf("[MacAllocator] Resetting MAC address to empty for NC %s.", nc.Config.NetworkContainerID)

		allocationId := nc.Config.NetworkContainerID + nc.Config.NetworkID
		_, err := m.MacManagerClient.DeallocateMac(allocationId)

		if err != nil {
			if err.HTTPStatus == http.StatusBadRequest {
				m.Logger.Debugf("[MacAllocator] Failed to deallocate MAC. Reason : %s, HttpStatus : %d. DNC will NOT retry.", err.Error, err.HTTPStatus)
				return true, err
			} else {
				m.Logger.Debugf("[MacAllocator] Failed to deallocate MAC. Reason : %s, HttpStatus : %d. DNC will retry.", err.Error, err.HTTPStatus)
				return false, err
			}
		}

		m.Logger.Debugf("[MacAllocator][Response] Deallocate MAC Successful. NC = %s.", nc.Config.NetworkContainerID)
		nc.Config.MACAddress = ""
	}

	err := m.markNCToUnreserve(nc)
	return false, err
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
