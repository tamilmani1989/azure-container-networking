package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
)

type MacAllocator struct {
	PersistNC            PersistNCFunc
	Logger               loggershim.LoggerInterface
}

// This function allocate a MAC address
func (m *MacAllocator) ProcessAllocateMAC(ncConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	containerType := ncConfig.GetAssociationInfo().ContainerType
	m.Logger.Debugf("The container type is : %s", containerType)

	// TODO: assign the real MAC address here.
	m.Logger.Debugf("Setting dummy MAC address.")
	ncConfig.State = datamodel.Publish
	ncConfig.MACAddress = "TempMAC"

	err := m.markNCToPublish(ncConfig)
	return false, err
}

// This utility function sets the NC as Failed.
func (m *MacAllocator) markNCFailed(ncConfig *datamodel.NCConfiguration, err api.JsonErr) (bool, *api.JsonErr){
	ncConfig.Status = datamodel.Failed
	ncConfig.FailureDetail = err
	if err := m.PersistNC(ncConfig, datamodel.DontAddToPendingQ); err != nil {
		return false, err // don't remove from queue until failure can be persisted
	}
	return true, nil
}

// This utility function sets the NC state as published.
func (m *MacAllocator) markNCToPublish(ncConfig *datamodel.NCConfiguration) *api.JsonErr{
	ncConfig.State = datamodel.Publish
	ncConfig.Status = datamodel.Pending
	ncConfig.FailureDetail = api.JsonErr{}
	return m.PersistNC(ncConfig, datamodel.DontAddToPendingQ)
}