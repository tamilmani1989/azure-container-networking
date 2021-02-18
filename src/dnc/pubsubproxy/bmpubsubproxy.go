// +build enablePubsubProxy

package pubsubproxy

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdn "dnc/microsoft/azure/sdncontracts"

	"github.com/Azure/azure-container-networking/cns"
	"google.golang.org/protobuf/proto"
)

const (
	// Version is unused field at this point. So are just using 1 for now.
	version = 1
	ipconfigName = "CA"
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
	ncId, _ := utils.ConvertGUIDStrToSdnContractGUID(repoNCConfig.ReservationID)

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
	macAddress, _ := utils.ConvertMacAddressToSdnContractMacAddress(repoNCConfig.MACAddress)

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
	sdnIpAddress, _ := utils.ConvertIPaddressToSdnContractIPAddress(ipAddress)

	ipConfig := &sdn.IPConfiguration{
		Name:              ipconfigName,
		IsPrimary:         true,
		CustomerIPAddress: sdnIpAddress,
	}

	// generate the array of interface ids in this container
	ipConfigs := make([]*sdn.IPConfiguration, 1)
	ipConfigs[0] = ipConfig

	networkInterfaceCompartment := &sdn.InterfaceCompartment{
		Id:               ncId,
		Version:          version,
		VnetSubnetName:   ncGoalState.BaseSubnetName,
		IsPrimary:        true,
		IPConfigurations: ipConfigs,
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

		// move this out the loop
		var secondaryContext []PublishJobContext

		// TO DO : Considering moving everthing ( i.e /Containers and /network interface to
		// secondarys fied, ( with TODO : Enhance Publish JobContext as a unit of context selector
		// and an array of path+ data)
		for _, networkInterface := range networkInterfaces {
			var pubJobContext PublishJobContext

			// Add config Path
			pubJobContext.Key = networkInterface.Key
			pubJobContext.Path = networkInterface.Path
			pubJobContext.Data = networkInterface.Data
			secondaryContext = append(secondaryContext, pubJobContext)
		}

		ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(
			ncGoalState.NetworkID,
			ncGoalState.NetworkContainerID,
			pubsubAzId)
		sessionInfo := map[string]string{datamodel.NCPendingPublishRepositoryName: ncPendingPublishContextKey}
		sessionManager.BeginWriteSession(sessionInfo)
		err := datamodel.CreateOrUpdateNcPendingPublishRequest(&ncPendingPublishRequestContext)
		sessionManager.EndWriteSession(sessionInfo)

		if err != nil {
			msg := fmt.Sprintf("PublishBareMetalNetworkContainer for NC: %s failed due to error: %v",
				ncPendingPublishRequestContext.NetworkContainerID, err)
			loggershim.LS.Errorf(msg)
			return api.InternalServerError(api.PublishingFailed, msg)
		}

		jobId := createNcJobId(ncGoalState.NetworkID, ncGoalState.NetworkContainerID, pubsubAzId)
		ncDataPublishJob := &ncJob{
			context: jobContext{
				id:              jobId,
				jobType:         BareMetalContainerPublish,
				pubsubAzId:      pubsubAzId,
				partitionKey:    getAzContainerPublishingContextSelector(podInfo.PodName),
				relativePath:    getAzContainerPublishingPath(),
				createdDateTime: ncPendingPublishRequestContext.CreatedDateTime,
				retryInterval:   baseRetryInterval,
				secondarys:      secondaryContext,
			},
			ncId:   ncGoalState.NetworkContainerID,
			vnetId: ncGoalState.NetworkID,
			// todo: centralize all C (de)ser logic to a single location
			// we are currently declaring the same C structs here and in workerpool.go as well
			ncData: ncDataBytes,
		}

		workerPool().scheduleJob(ncDataPublishJob)
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
		err := datamodel.CreateOrUpdateNcPendingPublishRequest(&pendingNcUnpublishRequestContext)
		sessionManager.EndWriteSession(sessionInfo)

		if err != nil {
			msg := fmt.Sprintf(
				"UnpublishNetworkContainer for NC: %s, failed to persist pending NC Unpublish request due to error : %v",
				ncGoalState.NetworkContainerID,
				err)
			loggershim.LS.Errorf(msg)
			return api.InternalServerError(api.UnpublishingFailed, msg)
		}

		var secondaryContext []PublishJobContext

		for _, networkInterface := range networkInterfaces {
			var pubJobContext PublishJobContext

			// Add config Path
			pubJobContext.Key = networkInterface.Key
			pubJobContext.Path = networkInterface.Path
			secondaryContext = append(secondaryContext, pubJobContext)
		}

		jobId := createNcJobId(ncGoalState.NetworkID, ncGoalState.NetworkContainerID, pubsubAzId)
		ncDataUnpublishJob := &ncJob{
			context: jobContext{
				id:              jobId,
				jobType:         BareMetalContainerUnpublish,
				pubsubAzId:      pubsubAzId,
				partitionKey:    getAzContainerPublishingContextSelector(podInfo.PodName),
				relativePath:    getAzContainerPublishingPath(),
				createdDateTime: pendingNcUnpublishRequestContext.CreatedDateTime,
				retryInterval:   baseRetryInterval,
				secondarys:      secondaryContext,
			},
			ncId:   ncGoalState.NetworkContainerID,
			vnetId: ncGoalState.NetworkID,
		}

		workerPool().scheduleJob(ncDataUnpublishJob)
	}

	return nil
}