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
	repo "dnc/networkcontainer"
	"dnc/replica"
	"dnc/utils"
	"fmt"
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
	logger                  loggershim.LoggerInterface
	ncPendingPublishRequestContextRepo repo.NcPendingPublishRequestContextRepo
	networkRepo             repo.NetworkRepo
	azureEnvironmentName    string
	aznmClient              AznmClient
	aznmSettings            configuration.AznmSettings
	verboseLogging          bool
)

func logDebugf(format string, args ...interface{}) {
	logger.Debugf("[PubSubProxy] %s", fmt.Sprintf(format, args...))
}

func logErrorf(format string, args ...interface{}) {
	logger.Errorf("[PubSubProxy] %s", fmt.Sprintf(format, args...))
}

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

	if networkContainerVersion, err = strconv.ParseUint(ncPublishRequest.Version, 10, 64); err != nil {
		return nil, fmt.Errorf("Failed to parse network container version. Error: %v", err)
	}

	cachedSubnetConfig, err := getActiveSubscriber().getCachedSubnetConfig(ncPublishRequest.VirtualNetworkID, ncPublishRequest.SubnetName)
	if err != nil {
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
		return nil, err
	}

	return C.GoBytes(serializedNcData, C.int(ncDataSize)), nil
}

// Validate the auth token by comparing with the expected token received from vnet config update notification
func validateAuthToken(
	vnetID string,
	subnetName string,
	accessToken string) *api.JsonErr {
	cachedSubnetConfig, err := getActiveSubscriber().getCachedSubnetConfig(vnetID, subnetName)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to get subnetConfig for vnet: %s, subnet: %s. Error: %v", vnetID, subnetName, err)
	}

	if accessToken != cachedSubnetConfig.expectedToken {
		return api.StatusUnauthorizedf(api.StatusInternalServerError,
			"Invalid Authentication token for vnet: %s, subnet: %s", vnetID, subnetName)
	}

	return nil
}

func PublishNetworkContainer(
	ncPublishRequest PublishNetworkContainerRequest) *api.JsonErr {
	// Hide the auth token in the logs
	subnetAuthToken := ncPublishRequest.SubnetAuthToken
	ncPublishRequest.SubnetAuthToken = "<contents-hidden>"
	logDebugf("PublishNetworkContainer: %s, %+v", ncPublishRequest.NcID, ncPublishRequest)
	ncPublishRequest.SubnetAuthToken = subnetAuthToken

	if err := validateAuthToken(
		ncPublishRequest.VirtualNetworkID,
		ncPublishRequest.SubnetName,
		ncPublishRequest.SubnetAuthToken); err != nil {
		logErrorf("PublishNetworkContainer for NC: %s failed due to error: %v", ncPublishRequest.NcID, err)
		return &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       fmt.Sprintf("PublishNetworkContainer for NC: %s failed due to error: %v", ncPublishRequest.NcID, err),
			HTTPStatus: err.HTTPStatus}
	}

	ncDataBytes, err := serializeNetworkContainerPublishData(ncPublishRequest)
	if err != nil {
		logErrorf("Failed to serialize NC Publish Data for NC: %s. Error: %v", ncPublishRequest.NcID, err)
		return api.InternalServerErrorf(api.PublishingFailed,
			"PublishNetworkContainer for NC: %s failed due to error: %v", ncPublishRequest.NcID, err)
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
		err := ncPendingPublishRequestContextRepo.CreateOrUpdateNcPendingPublishRequest(&ncPendingPublishRequestContext)
		sessionManager.EndWriteSession(sessionInfo)

		if err != nil {
			logErrorf("Failed to persist pending NC Publish request for NC: %s. Error: %v", ncPublishRequest.NcID, err)
			return api.InternalServerErrorf(api.PublishingFailed,
				"PublishNetworkContainer for NC: %s failed due to error: %v", ncPublishRequest.NcID, err)
		}
	}

	return nil
}

func UnpublishNetworkContainer(
	ncUnpublishRequest UnpublishNetworkContainerRequest) *api.JsonErr {
	logDebugf("UnpublishNetworkContainer: %s", ncUnpublishRequest.NcID)
	if err := validateAuthToken(ncUnpublishRequest.VnetID,
		ncUnpublishRequest.SubnetName,
		ncUnpublishRequest.SubnetAuthToken); err != nil {
		logErrorf("UnpublishNetworkContainer for NC: %s failed due to error: %v", ncUnpublishRequest.NcID, err)
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
		err := ncPendingPublishRequestContextRepo.CreateOrUpdateNcPendingPublishRequest(&pendingNcUnpublishRequestContext)
		sessionManager.EndWriteSession(sessionInfo)

		if err != nil {
			logErrorf("Failed to persist pending NC Unpublish request for NC: %s. Error: %v", ncUnpublishRequest.NcID, err)
			return api.InternalServerErrorf(api.UnpublishingFailed,
				"UnpublishNetworkContainer for NC: %s failed due to error: %v", ncUnpublishRequest.NcID, err)
		}
	}

	return nil
}

// RegisterVnet subscribes for getting updates on the vnet
func RegisterVnet(
	vnetId string) {
	// Subscribe only if not subscribed already
	if getActiveSubscriber().getCachedVnetConfig(vnetId) == nil ||
		getActiveSubscriber().getCachedVnetConfig(vnetId).isSubscribed == false {
		logDebugf("RegisterVnet: %s", vnetId)
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
	logDebugf("UnregisterVnet: %s", vnetId)
	if getActiveSubscriber().getCachedVnetConfig(vnetId) == nil {
		logErrorf("vnet: %s not found for unregistering", vnetId)
		return
	}

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
}

// Init initializes the PubSubProxy by creating publisherContext, subscribers and worker pool.
func Init(
	configSettings configuration.PubSubProxySettings,
	aznmConfigSettings configuration.AznmSettings,
	replica *replica.Replica,
	lgr loggershim.LoggerInterface,
	certPath string,
	azureEnvName string,
	client AznmClient) error {
	if len(configSettings.PubsubVips) <= 0 {
		return fmt.Errorf("Invalid number of pubsubs: %d", len(configSettings.PubsubVips))
	}

	maxConcurrentRequests = configSettings.MaxConcurrentRequests
	baseRetryInterval = time.Duration(configSettings.RequestRetryBaseInterval) * time.Millisecond
	maxRetryBackoffInterval = time.Duration(configSettings.MaxRequestBackoffInterval) * time.Millisecond
	replicaInstance = replica
	logger = lgr
	sessionManager = utils.NewSessionManager([]string{
		datamodel.NCPendingPublishRepositoryName})
	ncPendingPublishRequestContextRepo = new(datamodel.NcPendingPublishRequestContextRepo)
	networkRepo = new(datamodel.NetworkRepo)
	azureEnvironmentName = azureEnvName
	aznmClient = client
	aznmSettings = aznmConfigSettings
	verboseLogging = configSettings.EnableVerboseLogging

	pubsubSettings := pubsubSettings{
		pubsubVips:         configSettings.PubsubVips,
		publisherPort:      configSettings.PublisherPort,
		subscriberPort:     configSettings.SubscriberPort,
		pubsubCertName:     certPath,
		pubsubConnectionEp: configSettings.PubsubConnectionEndpoint,
	}

	madariClient := newMadariClient()

	go initPublishers(madariClient, pubsubSettings)
	go initSubscriber(madariClient, pubsubSettings)

	// Init worker pool
	workerPool()

	logDebugf("Successfully initialized")

	return nil
}

// De-init de-initializes the pubsubproxy by deleting publisherContext and subscribers
func Deinit() {
	deinitPublishers()
	deinitSubscriber()
	logDebugf("Successfully de-initialized")
}
