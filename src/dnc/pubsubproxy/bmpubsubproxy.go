// +build enablePubsubProxy

package pubsubproxy

import (
	"context"
	"dnc/aiwrapper"
	"dnc/api"
	"dnc/aznm"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"

	sdn "dnc/microsoft/azure/sdncontracts"

	"github.com/Azure/azure-container-networking/cns"
	"google.golang.org/protobuf/proto"
)

const (
	// Version is unused field at this point. So are just using 1 for now.
	version      = 1
	ipconfigName = "CA"
)

type aznmOperationType string

const (
	updateContainer aznmOperationType = "UpdateContainer"
	deleteContainer                   = "DeleteContainer"
)

// TODO:  remove duplication by combining to one function
func PublishBaremetalNetworkContainer(
	ncGoalState *datamodel.NCGoalState,
	repoNCConfig *datamodel.NCConfiguration) *api.JsonErr {

	loggershim.LS.Debugf("[PubSubProxy] PublishBareMetalNetworkContainer: %s", ncGoalState.NetworkContainerID)

	if err := validateAuthToken(
		ncGoalState.NetworkID,
		ncGoalState.BaseSubnetName,
		ncGoalState.AccessToken); err != nil {
		msg := fmt.Sprintf("PublishBareMetalNetworkContainer for NC: %s failed due to error: %v",
			ncGoalState.NetworkContainerID,
			err)
		loggershim.LS.Errorf("[PubSubProxy] %s", msg)
		return &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       msg,
			HTTPStatus: err.HTTPStatus}
	}

	// We use reservation ID to uniquely identify the NC on front end network
	ncId, _ := utils.ConvertGuidStrToCSharpGUID(repoNCConfig.ReservationID)
	vnetId, _ := utils.ConvertGuidStrToCSharpGUID(repoNCConfig.NetworkID)

	// generate the array of interface ids in this container
	networkInterfaceIds := make([]*sdn.Guid, 1)
	networkInterfaceIds[0] = ncId

	// create the container object that holds the mapping of nics.
	bmContainer := &sdn.Container{}
	bmContainer.NetworkInterfaceIds = networkInterfaceIds

	ncDataBytes, err := proto.Marshal(bmContainer)
	if err != nil {
		return api.InternalServerErrorf(
			api.PublishingFailed,
			"PublishBareMetalNetworkContainer for NC: %s,  Marshalling failed due to error: %v",
			ncGoalState.NetworkContainerID,
			err)
	}

	var networkInterfaces []datamodel.NetworkInterfacesPublishingContext
	var networkingInterfacePubContext datamodel.NetworkInterfacesPublishingContext

	// converts macaddress to proper format
	macAddress, err := utils.ConvertMacAddressToSdnContractMacAddress(repoNCConfig.MACAddress)
	if err != nil {
		return api.InternalServerErrorf(
			api.PublishingFailed,
			"PublishBareMetalNetworkContainer for NC: %s, Converting MAC %s to sdncontracts failed due to error: %s",
			ncGoalState.NetworkContainerID,
			repoNCConfig.MACAddress,
			err)
	}

	// do this for all the interfaces
	networkInterfaceConfiguration := &sdn.NetworkInterface{
		Id:         ncId,
		Version:    version,
		IsPrimary:  true,
		Name:       "eth0",
		Type:       sdn.NetworkInterfaceType_NetworkInterfaceType_Frontend,
		MacAddress: macAddress,
	}

	interfaceBytes, err := proto.Marshal(networkInterfaceConfiguration)
	if err != nil {
		return api.InternalServerErrorf(
			api.PublishingFailed,
			"PublishBareMetalNetworkContainer for NC: %s,  Marshalling failed for interface configuration due to error: %v",
			repoNCConfig.ReservationID,
			err)
	}

	networkingInterfacePubContext.Key = getNetworkInterfaceContextSelector(repoNCConfig.ReservationID)
	networkingInterfacePubContext.Path = getNetworkInterfaceConfigurationPublishingPath()
	networkingInterfacePubContext.Data = interfaceBytes

	networkInterfaces = append(networkInterfaces, networkingInterfacePubContext)

	// prune /32 from assigned ca
	ipAddress := strings.Split(ncGoalState.AllocatedIPSubnet, "/")[0]
	loggershim.LS.Errorf("[PubSubProxy] AllocatedIPSubnet: %s", ipAddress)
	sdnIpAddress, _ := utils.ConvertIPaddressToSdnContractIPAddress(ipAddress)

	ipConfig := &sdn.IPConfiguration{
		Name:              ipconfigName,
		IsPrimary:         true,
		CustomerIPAddress: sdnIpAddress,
	}

	// generate the array of interface ids in this container
	ipConfigs := make([]*sdn.IPConfiguration, 1)
	ipConfigs[0] = ipConfig

	// TODO: vnetSubetName needs to be a GUID from RNC rather than subnet name for BM scenario
	networkInterfaceCompartment := &sdn.InterfaceCompartment{
		Id:               ncId,
		Version:          version,
		VnetSubnetName:   ncGoalState.BaseSubnetName,
		IsPrimary:        true,
		IPConfigurations: ipConfigs,
		VnetId:           vnetId,
	}

	networkInterfaceCompartmentBytes, err := proto.Marshal(networkInterfaceCompartment)
	if err != nil {
		return api.InternalServerErrorf(
			api.PublishingFailed,
			"PublishBareMetalNetworkContainer for NC: %s,  Marshalling failed for interface conmpartment configuration due to error: %v",
			repoNCConfig.ReservationID,
			err)
	}

	networkingInterfacePubContext.Key = getNetworkInterfaceContextSelector(repoNCConfig.ReservationID)
	networkingInterfacePubContext.Path = getNetworkInterfaceCompartmentPublishingPath(repoNCConfig.ReservationID)
	networkingInterfacePubContext.Data = networkInterfaceCompartmentBytes

	networkInterfaces = append(networkInterfaces, networkingInterfacePubContext)

	// don't expect this to fail during unmarshaling
	var podInfo cns.KubernetesPodInfo
	if err := json.Unmarshal(repoNCConfig.GetAssociationInfo().OrchestratorContext, &podInfo); err != nil {
		return api.UnprocessableEntityf(
			api.JsonBodyUnMarshalFailed,
			"Failed to unmarshal pod info for NC: %s. Error:%s", ncGoalState.NetworkContainerID, err.Error())
	}

	for pubsubAzId, _ := range publisherContext.publishers {
		ncPendingPublishRequestContext := datamodel.NcPendingPublishRequestContext{
			Type:               datamodel.BareMetalNetworkContainerPublishRequest,
			NetworkID:          ncGoalState.NetworkID,
			NetworkContainerID: ncGoalState.NetworkContainerID,
			InfraNetworkID:     ncGoalState.InfraNetworkID,
			NodeInterfaceIP:    ncGoalState.AssociatedInterfaceID, // putting for now.
			PubsubAzId:         pubsubAzId,
			NcData:             ncDataBytes,
			CreatedDateTime:    time.Now().UTC(),
			RetryInterval:      baseRetryInterval,
			PodName:            podInfo.PodName,
			NetworkInterfaces:  networkInterfaces,
		}

		ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(
			ncGoalState.NetworkID,
			ncGoalState.NetworkContainerID,
			pubsubAzId)
		sessionInfo := map[string]string{datamodel.NCPendingPublishRepositoryName: ncPendingPublishContextKey}
		sessionManager.BeginWriteSession(sessionInfo)
		err := datamodel.NcPendingPublishRepo.CreateOrUpdateNcPendingPublishRequest(&ncPendingPublishRequestContext)
		sessionManager.EndWriteSession(sessionInfo)

		if err != nil {
			msg := fmt.Sprintf("PublishBareMetalNetworkContainer for NC: %s failed due to error: %v",
				ncPendingPublishRequestContext.NetworkContainerID, err)
			loggershim.LS.Errorf(msg)
			return api.InternalServerError(api.PublishingFailed, msg)
		}
	}

	assocInfo := repoNCConfig.GetAssociationInfo()
	logicalId := RetrieveLogicalContainerIdFromPodName(podInfo.PodName)
	jobId := createAznmJobId(repoNCConfig.NetworkID, repoNCConfig.NetworkContainerID)
	err = InvokeAznmContainerApi(updateContainer, ncGoalState.AZIdOrName, assocInfo.NodeID, logicalId, repoNCConfig.GetInstanceID(), jobId)
	if err != nil {
		msg := fmt.Sprintf("PublishBareMetalNetworkContainer for NC: %s failed due to aznm error: %v",
			repoNCConfig.NetworkContainerID, err)
		return api.InternalServerError(api.PublishingFailed, msg)
	}

	return nil
}

func UnpublishBaremetalNetworkContainer(
	ncGoalState *datamodel.NCGoalState,
	repoNCConfig *datamodel.NCConfiguration) *api.JsonErr {
	loggershim.LS.Debugf("[PubSubProxy] UnpublishNetworkContainer: %s", ncGoalState.NetworkContainerID)

	if err := validateAuthToken(
		ncGoalState.NetworkID,
		ncGoalState.BaseSubnetName,
		ncGoalState.AccessToken); err != nil {
		msg := fmt.Sprintf("UnpublishNetworkContainer for NC: %s failed due to error: %v",
			ncGoalState.NetworkContainerID,
			err)
		loggershim.LS.Errorf("[PubSubProxy]%s", msg)
		return &api.JsonErr{
			Code:       api.UnpublishingFailed,
			Text:       msg,
			HTTPStatus: err.HTTPStatus}
	}

	var podInfo cns.KubernetesPodInfo
	if err := json.Unmarshal(repoNCConfig.GetAssociationInfo().OrchestratorContext, &podInfo); err != nil {
		return api.UnprocessableEntityf(
			api.JsonBodyUnMarshalFailed,
			"UnpublishBaremetalNetworkContainer: Failed to unmarshal pod info for NC: %s. Error:%s",
			ncGoalState.NetworkContainerID, err.Error())
	}

	var networkInterfaces []datamodel.NetworkInterfacesPublishingContext
	var networkingInterfacePubContext datamodel.NetworkInterfacesPublishingContext

	// Add config Path
	networkingInterfacePubContext.Key = getNetworkInterfaceContextSelector(repoNCConfig.ReservationID)
	networkingInterfacePubContext.Path = getNetworkInterfaceConfigurationPublishingPath()
	networkInterfaces = append(networkInterfaces, networkingInterfacePubContext)

	// Add compartment path
	networkingInterfacePubContext.Key = getNetworkInterfaceContextSelector(repoNCConfig.ReservationID)
	networkingInterfacePubContext.Path = getNetworkInterfaceCompartmentPublishingPath(repoNCConfig.ReservationID)
	networkInterfaces = append(networkInterfaces, networkingInterfacePubContext)

	for pubsubAzId, _ := range publisherContext.publishers {
		pendingNcUnpublishRequestContext := datamodel.NcPendingPublishRequestContext{
			Type:               datamodel.BareMetalNetworkContainerUnpublishRequest,
			NetworkID:          ncGoalState.NetworkID,
			NetworkContainerID: ncGoalState.NetworkContainerID,
			InfraNetworkID:     ncGoalState.InfraNetworkID,
			NodeInterfaceIP:    ncGoalState.AssociatedInterfaceID, // putting for now.
			PubsubAzId:         pubsubAzId,
			CreatedDateTime:    time.Now().UTC(),
			RetryInterval:      baseRetryInterval,
			PodName:            podInfo.PodName,
			NetworkInterfaces:  networkInterfaces,
		}

		// TO DO : Switch to different key and struct but for now use this.
		ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(
			ncGoalState.NetworkID,
			ncGoalState.NetworkContainerID,
			pubsubAzId)
		sessionInfo := map[string]string{datamodel.NCPendingPublishRepositoryName: ncPendingPublishContextKey}
		sessionManager.BeginWriteSession(sessionInfo)
		err := datamodel.NcPendingPublishRepo.CreateOrUpdateNcPendingPublishRequest(&pendingNcUnpublishRequestContext)
		sessionManager.EndWriteSession(sessionInfo)

		if err != nil {
			msg := fmt.Sprintf(
				"UnpublishNetworkContainer for NC: %s, failed to persist pending NC Unpublish request due to error : %v",
				ncGoalState.NetworkContainerID,
				err)
			loggershim.LS.Errorf(msg)
			return api.InternalServerError(api.UnpublishingFailed, msg)
		}
	}

	assocInfo := repoNCConfig.GetAssociationInfo()
	logicalId := RetrieveLogicalContainerIdFromPodName(podInfo.PodName)
	jobId := createAznmJobId(repoNCConfig.NetworkID, repoNCConfig.NetworkContainerID)
	err := InvokeAznmContainerApi(deleteContainer, ncGoalState.AZIdOrName, assocInfo.NodeID, logicalId, repoNCConfig.GetInstanceID(), jobId)
	if err != nil {
		msg := fmt.Sprintf("UnpublishNetworkContainer for NC: %s failed due to aznm error: %v",
			repoNCConfig.NetworkContainerID, err)
		return api.InternalServerError(api.UnpublishingFailed, msg)
	}

	return nil
}

func RetrieveLogicalContainerIdFromPodName(podName string) string {
	var splits []string
	splits = strings.Split(podName, api.ContainerIdDelimiter)

	// Validation is not needed as it is already done during API call in handlers
	return splits[1]
}

//sendAznmAPIMetric sends performance time metric for the AznmContainerAPI calls
//Dimensions are - AzId or AzName, DNC Version, Call result (Success/Failure), ApiCallName - updateContainer/deleteContainer
//Metric value it sends - Time taken for the AznmContainerAPI call
func sendAznmAPIMetric(
	metricName string,
	metricValue float64,
	azId, apiName, aznmApiResult string) {
	customDimensions := make(map[string]string)
	metric := aitelemetry.Metric{
		Name:             metricName,
		Value:            metricValue,
		CustomDimensions: customDimensions,
	}

	metric.CustomDimensions[aiwrapper.AzIdorNameStr] = azId
	metric.CustomDimensions[aiwrapper.DNCVersionStr] = utils.Version
	metric.CustomDimensions[aiwrapper.AznmAPINameStr] = apiName
	metric.CustomDimensions[aiwrapper.AznmAPIResultStr] = aznmApiResult

	go aiwrapper.SendMetric(metric)
}

func InvokeAznmContainerApi(
	operType aznmOperationType,
	azIdOrName, nodeId, logicalId, physicalId, jobId string) error {

	var err error
	var aznmReq aznm.AznmRequest

	if aznmReq, err = aznm.CreateAznmRequestWithAzName(azIdOrName, aznmSettings.Port, azureEnvironmentName); err != nil {
		return err
	}

	loggershim.LS.Debugf("Starting %s in Aznm. ActivityId: %s, logicalId %s, physicalId %s, nodeId %s, Job %s",
		operType, aznmReq.ActivityId, logicalId, physicalId, nodeId, jobId)

	startTime := time.Now()
	var apiCallTimeMs int64

	if operType == updateContainer {
		err = aznmClient.UpdateContainer(
			context.Background(), aznmReq, logicalId, physicalId, nodeId)
	} else if operType == deleteContainer {
		err = aznmClient.DeleteContainer(
			context.Background(), aznmReq, logicalId, physicalId)
	} else {
		err = fmt.Errorf("invalid Aznm container api specified :%s", operType)
	}
	apiCallTimeMs = time.Since(startTime).Milliseconds()

	var aznmCallResult string
	if err != nil {
		aznmCallResult = aiwrapper.Failed
	} else {
		aznmCallResult = aiwrapper.Succeeded
	}

	//Emit Metric for AznmContainerAPI performance time if operation type is valid
	if operType == updateContainer || operType == deleteContainer {
		sendAznmAPIMetric("ApiCallTimeMetric", float64(apiCallTimeMs), azIdOrName, string(operType), aznmCallResult)
	}

	if err != nil {
		loggershim.LS.Debugf("%s in Aznm failed. ActivityId: %s, Job: %s, error: %s",
			operType, aznmReq.ActivityId, jobId, err)

		return err
	}

	loggershim.LS.Debugf("%s in Aznm succeeded. Job: %s, ActivityId: %s", operType, jobId, aznmReq.ActivityId)

	return err
}