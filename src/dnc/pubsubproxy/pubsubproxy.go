// +build enablePubsubProxy

package pubsubproxy

/*
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

struct NC_DATA {
	uint64_t NetworkContainerVersion;
    char* SubnetPrefix;
    char* CustomerAddress;
	char* SecondaryCustomerAddresses;
    char* RoleInstanceId;
    char* RoleInstanceName;
    char* VnetId;
    char* VlanId;
    char* NcSlbGreKey;
};

struct TagConfiguration {
	char* Name;
	char* Type;
};

struct SubnetConfiguration {
	char* SubnetName;
	char* AddressPrefix;
	struct TagConfiguration* Tags;
    uint32_t TagCount;
	char* DefaultGateway;
	char* ExpectedToken;
    char* OwningService;
};

struct VnetConfiguration {
	char* VnetVersion;
	char* VNetSpace;
	char* CNetSpace;
	char* DNSServers;
	struct SubnetConfiguration* Subnets;
    uint32_t SubnetCount;
};

struct ResultStatus {
    int32_t ResultCode;
    const char* ErrorDetails;
};

struct ResultStatus DeserializeVnetConfiguration(void*, size_t, struct VnetConfiguration*);
struct ResultStatus SerializeNetworkContainerPublishData(struct NC_DATA*, void**, size_t*);
*/
import "C"

import (
	"context"
	"dnc/api"
	"dnc/aznm"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/replica"
	"dnc/utils"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

const (
	// DNC scheme and identifier for Pubsub connection
	scheme     = "DNC"
	identifier = "digest"

	// PubSub paths
	vnetPartitionKeyPrefix   = "/vnets/"
	vnetInterfacePathPrefix  = "/interfaces/"
	vnetNcPathPrefix         = "/ncs/"
	vnetConfigRelativePath   = "/configurations/vnet"
	subnetConfigRelativePath = "/configurations/subnets"

	// NC publish constants
	ncRoleInstanceId   = "WebFrontEnd_IN_0"
	ncRoleInstanceName = "WebFrontEnd"

	pubsubVipListSeparator = "|"

	// Keyvault
	maxKeyvaultRetrievalTime = 10
)

type pubsubSettings struct {
	pubsubVips         []configuration.PubsubVips
	publisherPort      string
	subscriberPort     string
	pubsubCertName     string
	pubsubConnectionEp string
}

type TagConfiguration struct {
	Name string
	Type string
}

type SubnetConfiguration struct {
	SubnetName     string
	AddressPrefix  string
	Tags           []TagConfiguration
	DefaultGateway string
	ExpectedToken  string
	OwningService  string
}

type VnetConfiguration struct {
	VnetVersion   string
	VNetSpace     string
	CNetSpace     string
	DNSServers    string
	SubnetConfigs []SubnetConfiguration
}

type PublishNetworkContainerRequest struct {
	NcID             string
	InfraVnetID      string
	NodeInterfaceIP  string
	SubnetAuthToken  string
	PrimaryIP        string
	SecondaryIPs     []string
	SubnetName       string
	Version          string
	Policies         []string
	VirtualNetworkID string
	VlanID           int
	GreKey           int
}

type UnpublishNetworkContainerRequest struct {
	NcID            string
	InfraVnetID     string
	NodeInterfaceIP string
	VnetID          string
	SubnetName      string
	NcAllocatedIP   string
	SubnetAuthToken string
}

// AZNM client interface. Parameters are described below
// reqContext - context
// logicalId - logical container id. Should be Guid
// physicalId - physicalId container id. Should be Guid
// bladeId - Azure blade or node id. Should be Guid
type AznmClient interface {
	UpdateContainer(ctx context.Context, req aznm.AznmRequest, logicalId, physicalId, bladeId string) error
	DeleteContainer(ctx context.Context, req aznm.AznmRequest, logicalId, physicalId string) error
}

var (
	maxConcurrentRequests   int
	baseRetryInterval       time.Duration
	maxRetryBackoffInterval time.Duration
	sessionManager          *utils.SessionManager
	replicaInstance         *replica.Replica
	azureEnvironmentName    string
	aznmClient              AznmClient
	aznmSettings            configuration.AznmSettings
	verboseLogging          bool
)

func generateVnetPartitionKey(
	vnetID string) string {
	return vnetPartitionKeyPrefix + vnetID
}

func generateVnetNcPath(
	nodeInterfaceIP string,
	ncID string) string {
	return vnetInterfacePathPrefix + nodeInterfaceIP + vnetNcPathPrefix + ncID
}

func serializeNetworkContainerPublishData(
	ncPublishRequest PublishNetworkContainerRequest) ([]byte, error) {
	var (
		networkContainerVersion uint64
		err                     error
		ncData                  C.struct_NC_DATA
		ncDataSize              C.size_t
		serializedNcData        unsafe.Pointer
	)

	networkContainerVersion, err = strconv.ParseUint(ncPublishRequest.Version, 10, 64)
	if err != nil {
		loggershim.LS.Errorf("Failed to parse network container version. Error: %v", err)
		return nil, err
	}

	cachedSubnetConfig, err := getActiveSubscriber().getCachedSubnetConfig(
		ncPublishRequest.VirtualNetworkID, ncPublishRequest.SubnetName)
	if err != nil {
		loggershim.LS.Errorf("[PubSubProxy] %v", err)
		return nil, err
	}

	ncData.NetworkContainerVersion = C.uint64_t(networkContainerVersion)
	ncData.SubnetPrefix = C.CString(cachedSubnetConfig.addressPrefix)
	ncData.CustomerAddress = C.CString(ncPublishRequest.PrimaryIP)
	ncData.SecondaryCustomerAddresses = C.CString(strings.Join(ncPublishRequest.SecondaryIPs, ","))
	ncData.RoleInstanceId = C.CString(ncRoleInstanceId)
	ncData.RoleInstanceName = C.CString(ncRoleInstanceName)
	ncData.VnetId = C.CString(ncPublishRequest.VirtualNetworkID)
	ncData.VlanId = C.CString(fmt.Sprintf("%d", ncPublishRequest.VlanID))
	ncData.NcSlbGreKey = C.CString(fmt.Sprintf("%d", ncPublishRequest.GreKey))

	defer func() {
		// Free the C memory holding the serialized NC data and ncData allocated strings
		C.free(serializedNcData)
		C.free(unsafe.Pointer(ncData.SubnetPrefix))
		C.free(unsafe.Pointer(ncData.CustomerAddress))
		C.free(unsafe.Pointer(ncData.SecondaryCustomerAddresses))
		C.free(unsafe.Pointer(ncData.RoleInstanceId))
		C.free(unsafe.Pointer(ncData.RoleInstanceName))
		C.free(unsafe.Pointer(ncData.VnetId))
		C.free(unsafe.Pointer(ncData.VlanId))
		C.free(unsafe.Pointer(ncData.NcSlbGreKey))
	}()

	var resultStatus C.struct_ResultStatus
	if resultStatus = C.SerializeNetworkContainerPublishData(
		&ncData, &serializedNcData, &ncDataSize); resultStatus.ResultCode != 0 {
		err = fmt.Errorf("Failed to serialize NC publish data for NC: %s due to error: %s",
			ncPublishRequest.NcID, C.GoString(resultStatus.ErrorDetails))
		loggershim.LS.Errorf("[PubSubProxy] %v", err)
		return nil, err
	}

	return C.GoBytes(serializedNcData, C.int(ncDataSize)), nil
}

// Validate the auth token by comparing with the expected token
// received from vnet config update notification
func validateAuthToken(
	vnetID string,
	subnetName string,
	accessToken string) *api.JsonErr {
	cachedSubnetConfig, err := getActiveSubscriber().getCachedSubnetConfig(vnetID, subnetName)
	if err != nil {
		loggershim.LS.Errorf("[PubSubProxy] %v", err)
		return &api.JsonErr{
			Code: api.StatusInternalServerError,
			Text: fmt.Sprintf("Failed to get subnetConfig for vnet: %s, subnet: %s. Error: %v",
				vnetID, subnetName, err),
			HTTPStatus: http.StatusInternalServerError}
	}

	if accessToken != cachedSubnetConfig.expectedToken {
		err := fmt.Errorf("Invalid Authentication token for vnet: %s, subnet: %s", vnetID, subnetName)
		loggershim.LS.Errorf("[PubSubProxy] %v", err)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       fmt.Sprintf("%v", err),
			HTTPStatus: http.StatusUnauthorized}
	}

	return nil
}

func PublishNetworkContainer(
	ncPublishRequest PublishNetworkContainerRequest) *api.JsonErr {
	// Hide the auth token in the logs
	subnetAuthToken := ncPublishRequest.SubnetAuthToken
	ncPublishRequest.SubnetAuthToken = "<contents-hidden>"
	loggershim.LS.Debugf("[PubSubProxy] PublishNetworkContainer: %s, %+v", ncPublishRequest.NcID, ncPublishRequest)
	ncPublishRequest.SubnetAuthToken = subnetAuthToken

	if err := validateAuthToken(
		ncPublishRequest.VirtualNetworkID,
		ncPublishRequest.SubnetName,
		ncPublishRequest.SubnetAuthToken); err != nil {
		loggershim.LS.Errorf("[PubSubProxy] PublishNetworkContainer for NC: %s failed due to error: %v",
			ncPublishRequest.NcID, err)
		return &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       fmt.Sprintf("PublishNetworkContainer for NC: %s failed due to error: %v", ncPublishRequest.NcID, err),
			HTTPStatus: err.HTTPStatus}
	}

	ncDataBytes, err := serializeNetworkContainerPublishData(ncPublishRequest)
	if err != nil {
		loggershim.LS.Errorf("Failed to serialize NC Publish Data for NC: %s. Error: %v", ncPublishRequest.NcID, err)
		return &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       fmt.Sprintf("PublishNetworkContainer for NC: %s failed due to error: %v", ncPublishRequest.NcID, err),
			HTTPStatus: http.StatusInternalServerError}
	}

	for pubsubAzId, _ := range publisherContext.publishers {
		ncPendingPublishRequestContext := datamodel.NcPendingPublishRequestContext{
			Type:               datamodel.NetworkContainerPublishRequest,
			NetworkID:          ncPublishRequest.VirtualNetworkID,
			NetworkContainerID: ncPublishRequest.NcID,
			InfraNetworkID:     ncPublishRequest.InfraVnetID,
			NodeInterfaceIP:    ncPublishRequest.NodeInterfaceIP,
			PubsubAzId:         pubsubAzId,
			NcData:             ncDataBytes,
			CreatedDateTime:    time.Now().UTC(),
			RetryInterval:      baseRetryInterval,
		}

		ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(ncPublishRequest.VirtualNetworkID,
			ncPublishRequest.NcID, pubsubAzId)
		sessionInfo := map[string]string{datamodel.NCPendingPublishRepositoryName: ncPendingPublishContextKey}
		sessionManager.BeginWriteSession(sessionInfo)
		err := datamodel.CreateOrUpdateNcPendingPublishRequest(&ncPendingPublishRequestContext)
		sessionManager.EndWriteSession(sessionInfo)

		if err != nil {
			loggershim.LS.Errorf("Failed to persist pending NC Publish request for NC: %s. Error: %v", ncPublishRequest.NcID, err)
			return &api.JsonErr{
				Code:       api.PublishingFailed,
				Text:       fmt.Sprintf("PublishNetworkContainer for NC: %s failed due to error: %v", ncPublishRequest.NcID, err),
				HTTPStatus: http.StatusInternalServerError}
		}

		ncDataPublishJob := &ncJob{
			context: jobContext{
				id:              createNcJobId(ncPublishRequest.VirtualNetworkID, ncPublishRequest.NcID, pubsubAzId),
				jobType:         NetworkContainerPublish,
				pubsubAzId:      pubsubAzId,
				partitionKey:    generateVnetPartitionKey(ncPublishRequest.InfraVnetID),
				relativePath:    generateVnetNcPath(ncPublishRequest.NodeInterfaceIP, ncPublishRequest.NcID),
				createdDateTime: ncPendingPublishRequestContext.CreatedDateTime,
				retryInterval:   baseRetryInterval,
			},
			ncId:   ncPublishRequest.NcID,
			vnetId: ncPublishRequest.VirtualNetworkID,
			// todo: centralize all C (de)ser logic to a single location
			// we are currently declaring the same C structs here and in workerpool.go as well
			ncData: ncDataBytes,
		}

		workerPool().scheduleJob(ncDataPublishJob)
	}

	return nil
}

func UnpublishNetworkContainer(
	ncUnpublishRequest UnpublishNetworkContainerRequest) *api.JsonErr {
	loggershim.LS.Debugf("[PubSubProxy] UnpublishNetworkContainer: %s", ncUnpublishRequest.NcID)

	if err := validateAuthToken(ncUnpublishRequest.VnetID,
		ncUnpublishRequest.SubnetName,
		ncUnpublishRequest.SubnetAuthToken); err != nil {
		loggershim.LS.Errorf("[PubSubProxy] UnpublishNetworkContainer for NC: %s failed due to error: %v",
			ncUnpublishRequest.NcID, err)
		return &api.JsonErr{
			Code:       api.UnpublishingFailed,
			Text:       fmt.Sprintf("UnpublishNetworkContainer for NC: %s failed due to error: %v", ncUnpublishRequest.NcID, err),
			HTTPStatus: err.HTTPStatus}
	}

	for pubsubAzId, _ := range publisherContext.publishers {
		pendingNcUnpublishRequestContext := datamodel.NcPendingPublishRequestContext{
			Type:               datamodel.NetworkContainerUnpublishRequest,
			NetworkID:          ncUnpublishRequest.VnetID,
			NetworkContainerID: ncUnpublishRequest.NcID,
			InfraNetworkID:     ncUnpublishRequest.InfraVnetID,
			NodeInterfaceIP:    ncUnpublishRequest.NodeInterfaceIP,
			PubsubAzId:         pubsubAzId,
			CreatedDateTime:    time.Now().UTC(),
			RetryInterval:      baseRetryInterval,
		}

		ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(ncUnpublishRequest.VnetID,
			ncUnpublishRequest.NcID, pubsubAzId)
		sessionInfo := map[string]string{datamodel.NCPendingPublishRepositoryName: ncPendingPublishContextKey}
		sessionManager.BeginWriteSession(sessionInfo)
		err := datamodel.CreateOrUpdateNcPendingPublishRequest(&pendingNcUnpublishRequestContext)
		sessionManager.EndWriteSession(sessionInfo)

		if err != nil {
			loggershim.LS.Errorf("Failed to persist pending NC Unpublish request for NC: %s. Error: %v",
				ncUnpublishRequest.NcID, err)
			return &api.JsonErr{
				Code:       api.UnpublishingFailed,
				Text:       fmt.Sprintf("UnpublishNetworkContainer for NC: %s failed due to error: %v", ncUnpublishRequest.NcID, err),
				HTTPStatus: http.StatusInternalServerError}
		}

		ncDataUnpublishJob := &ncJob{
			context: jobContext{
				id:              createNcJobId(ncUnpublishRequest.VnetID, ncUnpublishRequest.NcID, pubsubAzId),
				jobType:         NetworkContainerUnpublish,
				pubsubAzId:      pubsubAzId,
				partitionKey:    generateVnetPartitionKey(ncUnpublishRequest.InfraVnetID),
				relativePath:    generateVnetNcPath(ncUnpublishRequest.NodeInterfaceIP, ncUnpublishRequest.NcID),
				createdDateTime: pendingNcUnpublishRequestContext.CreatedDateTime,
				retryInterval:   baseRetryInterval,
			},
			ncId:   ncUnpublishRequest.NcID,
			vnetId: ncUnpublishRequest.VnetID,
		}

		workerPool().scheduleJob(ncDataUnpublishJob)
	}

	return nil
}

// RegisterVnet subscribes for getting updates on the vnet
func RegisterVnet(
	vnetId string) {
	// Subscribe only if not subscribed already
	if getActiveSubscriber().getCachedVnetConfig(vnetId) == nil ||
		getActiveSubscriber().getCachedVnetConfig(vnetId).isSubscribed == false {
		loggershim.LS.Debugf("[PubSubProxy] RegisterVnet: %s", vnetId)
		getActiveSubscriber().subscribeCachedVnetConfig(vnetId)
		vnetRegister := &vnetJob{
			context: jobContext{
				id:              createVnetJobId(vnetId),
				jobType:         VnetRegister,
				createdDateTime: time.Now().UTC(),
				retryInterval:   baseRetryInterval,
			},
			vnetId: vnetId,
		}
		workerPool().scheduleJob(vnetRegister)
	}
}

// UnregisterVnet deletes the subscription for the vnet
func UnregisterVnet(
	vnetId string) {
	loggershim.LS.Debugf("[PubSubProxy] UnregisterVnet: %s", vnetId)
	if getActiveSubscriber().getCachedVnetConfig(vnetId) != nil {
		// Unsubscribe from vnet notifications
		vnetUnregister := &vnetJob{
			context: jobContext{
				id:              createVnetJobId(vnetId),
				jobType:         VnetUnregister,
				createdDateTime: time.Now().UTC(),
				retryInterval:   baseRetryInterval,
			},
			vnetId: vnetId,
		}
		workerPool().scheduleJob(vnetUnregister)
	} else {
		loggershim.LS.Debugf("[PubSubProxy] vnet: %s not found for unregistering", vnetId)
	}
}

// Init initializes the pubsubproxy by creating
// publisherContext, subscribers and worker pool.
func Init(
	configSettings configuration.PubSubProxySettings,
	aznmConfigSettings configuration.AznmSettings,
	replica *replica.Replica,
	certPath string,
	azureEnvName string,
	client AznmClient) error {
	maxConcurrentRequests = configSettings.MaxConcurrentRequests
	baseRetryInterval = time.Duration(configSettings.RequestRetryBaseInterval) * time.Millisecond
	maxRetryBackoffInterval = time.Duration(configSettings.MaxRequestBackoffInterval) * time.Millisecond
	replicaInstance = replica
	sessionManager = utils.NewSessionManager([]string{
		datamodel.NCPendingPublishRepositoryName})
	azureEnvironmentName = azureEnvName
	aznmClient = client
	aznmSettings = aznmConfigSettings
	verboseLogging = configSettings.EnableVerboseLogging

	numPubSubs := len(configSettings.PubsubVips)
	if numPubSubs <= 0 {
		return fmt.Errorf("Invalid number of pubsubs: %d", numPubSubs)
	}

	pubsubSettings := pubsubSettings{
		pubsubVips:         configSettings.PubsubVips,
		publisherPort:      configSettings.PublisherPort,
		subscriberPort:     configSettings.SubscriberPort,
		pubsubCertName:     certPath,
		pubsubConnectionEp: configSettings.PubsubConnectionEndpoint,
	}

	go initPublishers(pubsubSettings)
	go initSubscriber(pubsubSettings)

	// Init worker pool
	workerPool()

	loggershim.LS.Errorf("[PubSubProxy] Successfully initialized")

	return nil
}

// Deinit deinitializes the pubsubproxy by deleting publisherContext and subscribers
func Deinit() {
	deinitPublishers()
	deinitSubscriber()
	loggershim.LS.Debugf("[PubSubProxy] Successfully deinitialized")
}