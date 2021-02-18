// +build !enablePubsubProxy

package pubsubproxy

import (
	"dnc/api"
	"dnc/configuration"
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

// Init initializes the pubsubproxy by creating
// publisherContext, subscribers and worker pool.
func Init(
	configSettings configuration.PubSubProxySettings,
	replica *replica.Replica,
	certPath string) error {
	return fmt.Errorf("PubSubProxy module build disabled. Build DNC using -tags enablePubsubProxy")
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