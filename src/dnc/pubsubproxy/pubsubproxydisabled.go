// +build !enablePubsubProxy

package pubsubproxy

import (
	"context"
	"dnc/api"
	"dnc/aznm"
	"dnc/configuration"
	"dnc/logger/loggershim"
	"dnc/datamodel"
	"dnc/replica"
	"fmt"
	"net/http"
)

// todo: extract request structs to different package to avoid duplication
// we are currently declaring them twice due to build constraints
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
	PodName          string
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
// ctx - context
// activityId - random unique id to log API requests on server side
// logicalId - logical container id. Should be Guid
// physicalId - physicalId container id. Should be Guid
// bladeId - Azure blade or node id. Should be Guid
type AznmClient interface {
	UpdateContainer(ctx context.Context, req aznm.AznmRequest, logicalId, physicalId, bladeId string) error
	DeleteContainer(ctx context.Context, req aznm.AznmRequest, logicalId, physicalId string) error
}

// Init initializes the pubsubproxy by creating
// publisherContext, subscribers and worker pool.
func Init(
	configSettings configuration.PubSubProxySettings,
	aznmSettings configuration.AznmSettings,
	replica *replica.Replica,
	logger loggershim.LoggerInterface,
	certPath string,
	azureEnvName string,
	aznmClient AznmClient) error {
	return fmt.Errorf("PubSubProxy module build disabled. Build DNC using -tags enablePubsubProxy")
}

//just make a seperate class? A little dangerous to have unitialized package variables that could be used
//note since deleting is idempotent we don't need replica
func InitCleanup(configSettings configuration.PubSubProxySettings, certPath string) error {
	return fmt.Errorf("PubSubProxy module build disabled. Build using -tags enablePubsubProxy")
}

// Deinit deinitializes the pubsubproxy by deleting publisherContext and subscribers
func Deinit() {
}

type publisher struct {
	pubsubAzId string
}

func getAllPublishers() map[string]*publisher {
	return map[string]*publisher{
		"disabled": &publisher{
			pubsubAzId: "1",
		},
	}
}

func PublishNetworkContainer(
	ncPublishRequest PublishNetworkContainerRequest) *api.JsonErr {
	return &api.JsonErr{
		Code:       api.StatusInternalServerError,
		HTTPStatus: http.StatusBadRequest,
		Text:       fmt.Sprintf("PubSubProxy module build disabled. Build DNC using -tags enablePubsubProxy")}
}

func UnpublishNetworkContainer(
	ncUnpublishRequest UnpublishNetworkContainerRequest) *api.JsonErr {
	return &api.JsonErr{
		Code:       api.StatusInternalServerError,
		HTTPStatus: http.StatusBadRequest,
		Text:       fmt.Sprintf("PubSubProxy module build disabled. Build DNC using -tags enablePubsubProxy")}
}

func PublishBaremetalNetworkContainer(
	ncGoalState *datamodel.NCGoalState,
	repoNCConfig *datamodel.NCConfiguration) *api.JsonErr {
	return api.InternalServerErrorf(
		api.StatusBadRequest,
		fmt.Sprintf("PubSubProxy module build disabled. Build DNC using -tags enablePubsubProxy"))
}

func UnpublishBaremetalNetworkContainer(
	ncGoalState *datamodel.NCGoalState,
	repoNCConfig *datamodel.NCConfiguration) *api.JsonErr {
	return api.InternalServerErrorf(
		api.StatusBadRequest,
		fmt.Sprintf("PubSubProxy module build disabled. Build DNC using -tags enablePubsubProxy"))
}

// RegisterVnet subscribes for getting updates on the the vnet
func RegisterVnet(
	vnetId string) {
}

// UnregisterVnet deletes the subscription for the vnet
func UnregisterVnet(
	vnetId string) {
}

func ForceUnpublishInAz(infraVnetID, nodeInterfaceIP, ncID, pubsubAzId string) error {
	//validateAuthToken
	return fmt.Errorf("PubSubProxy module build disabled. Build using -tags enablePubsubProxy")
}

func ForceUnpublishInAllAzs(infraVnetID, nodeInterfaceIP, ncID string) error {
	//validateAuthToken
	return fmt.Errorf("PubSubProxy module build disabled. Build using -tags enablePubsubProxy")
}
