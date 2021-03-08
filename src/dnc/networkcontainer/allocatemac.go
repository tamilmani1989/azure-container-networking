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
func (m *MacAllocator) ProcessAllocateMAC(ncConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	containerType := ncConfig.GetAssociationInfo().ContainerType
	m.Logger.Debugf("[MacAllocator] Starting MAC allocation. The container type is : %s", containerType)

	allocationId := ncConfig.NetworkContainerID + ncConfig.NetworkID
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
	m.Logger.Debugf("[MacAllocator][Response] Allocate MAC Successful. NC = %s : Mac Address = %+v", ncConfig.NetworkContainerID, allocatedMac)

	ncConfig.State = datamodel.Publish
	ncConfig.MACAddress = allocatedMac
	m.Logger.Debugf("[MacAllocator] Setting MAC address: %s for NC %s", ncConfig.MACAddress, ncConfig.NetworkContainerID)

	err = m.markNCToPublish(ncConfig)
	return false, err
}

// This function deallocates a MAC address.
func (m *MacAllocator) ProcessDeallocateMAC(ncConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	if ncConfig.MACAddress == "" {
		m.Logger.Debugf("[MacAllocator] MAC address is already empty. No need to deallocate.")
	} else {
		m.Logger.Debugf("[MacAllocator] Resetting MAC address to empty for NC %s.", ncConfig.NetworkContainerID)

		allocationId := ncConfig.NetworkContainerID + ncConfig.NetworkID
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

		m.Logger.Debugf("[MacAllocator][Response] Deallocate MAC Successful. NC = %s.", ncConfig.NetworkContainerID)
		ncConfig.MACAddress = ""
	}

	err := m.markNCToUnreserve(ncConfig)
	return false, err
}

// This utility function sets the NC state as published.
func (m *MacAllocator) markNCToPublish(ncConfig *datamodel.NCConfiguration) *api.JsonErr {
	ncConfig.State = datamodel.Publish
	ncConfig.Status = datamodel.Pending
	ncConfig.FailureDetail = api.JsonErr{}
	return m.PersistNC(ncConfig, datamodel.DontAddToPendingQ)
}

// This utility function sets the NC state as published.
func (m *MacAllocator) markNCToUnreserve(ncConfig *datamodel.NCConfiguration) *api.JsonErr {
	ncConfig.State = datamodel.UnReserve
	ncConfig.Status = datamodel.Pending
	ncConfig.FailureDetail = api.JsonErr{}
	return m.PersistNC(ncConfig, datamodel.DontAddToPendingQ)
}
