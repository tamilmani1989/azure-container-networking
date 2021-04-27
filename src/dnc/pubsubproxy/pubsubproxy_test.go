// +build enablePubsubProxy

package pubsubproxy

import (
	. "dnc/MadariClient"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/logger/loggermock"
	"dnc/networkcontainer/fakes"
	"dnc/replica"
	"dnc/utils"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeMadariClient struct {
}

type fakeMadariPublisher struct {
	callback publisherCallback
}

type fakeMadariSubscriber struct {
	callback subscriberCallback
}

var (
	fakeMadariCl *fakeMadariClient
	settings     pubsubSettings
)

var defaultNcPublishRequest = PublishNetworkContainerRequest{
	NcID:             "nc-success",
	InfraVnetID:      "bc75f302-2682-4dd0-95be-e84f15a9baf4",
	NodeInterfaceIP:  "172.18.1.5",
	SubnetAuthToken:  "4406767b-4a52-49cf-8772-6455d17bfd16",
	PrimaryIP:        "192.168.2.4",
	SubnetName:       "todelegate",
	Version:          "0",
	VirtualNetworkID: "vnet-success",
	VlanID:           1,
}

var defaultNcUnpublishRequest = UnpublishNetworkContainerRequest{
	NcID:            "nc-success",
	InfraVnetID:     "bc75f302-2682-4dd0-95be-e84f15a9baf4",
	NodeInterfaceIP: "172.18.1.5",
	VnetID:          "vnet-success",
	SubnetName:      "todelegate",
	SubnetAuthToken: "4406767b-4a52-49cf-8772-6455d17bfd16",
}

/* This is a serialized vnet data that pubsubproxy receives from pubsub. Pubsubproxy deserializes it to extract the vnet config.
   This byte data is for vnet with
   vnetVersion:  "6",
   subnetConfigs:
       {subnetName:    "default",
		addressPrefix: "192.168.1.0/24",
		expectedToken: "",
		owningService: ""},
       {subnetName:    "todelegate",
		addressPrefix: "192.168.2.0/24",
		expectedToken: "4406767b-4a52-49cf-8772-6455d17bfd16",
		owningService: "Kubernetes"},
*/

var serializedVnetData = []byte{123, 34, 86, 101, 114, 115, 105, 111, 110, 34, 58, 54, 44, 34, 73, 100, 34, 58, 34, 102, 57, 99, 50, 53, 100, 49, 55, 45, 48, 53,
	48, 98, 45, 52, 56, 57, 50, 45, 97, 55, 50, 48, 45, 56, 57, 100, 51, 57, 56, 97, 49, 57, 48, 97, 100, 34, 44, 34, 86, 78, 101, 116, 83, 112, 97, 99,
	101, 34, 58, 91, 91, 123, 34, 65, 100, 100, 114, 101, 115, 115, 34, 58, 123, 34, 84, 121, 112, 101, 34, 58, 49, 44, 34, 68, 97, 116, 97, 49, 34, 58,
	49, 57, 50, 44, 34, 68, 97, 116, 97, 50, 34, 58, 49, 54, 56, 125, 44, 34, 80, 114, 101, 102, 105, 120, 34, 58, 49, 54, 125, 93, 93, 44, 34, 67, 78,
	101, 116, 83, 112, 97, 99, 101, 34, 58, 91, 91, 123, 34, 65, 100, 100, 114, 101, 115, 115, 34, 58, 123, 34, 84, 121, 112, 101, 34, 58, 49, 44, 34, 68,
	97, 116, 97, 49, 34, 58, 49, 48, 125, 44, 34, 80, 114, 101, 102, 105, 120, 34, 58, 56, 125, 44, 123, 34, 65, 100, 100, 114, 101, 115, 115, 34, 58, 123,
	34, 84, 121, 112, 101, 34, 58, 49, 44, 34, 68, 97, 116, 97, 49, 34, 58, 49, 55, 50, 44, 34, 68, 97, 116, 97, 50, 34, 58, 49, 54, 125, 44, 34, 80, 114,
	101, 102, 105, 120, 34, 58, 49, 50, 125, 44, 123, 34, 65, 100, 100, 114, 101, 115, 115, 34, 58, 123, 34, 84, 121, 112, 101, 34, 58, 49, 44, 34, 68, 97,
	116, 97, 49, 34, 58, 49, 57, 50, 44, 34, 68, 97, 116, 97, 50, 34, 58, 49, 54, 56, 125, 44, 34, 80, 114, 101, 102, 105, 120, 34, 58, 49, 54, 125, 93, 93,
	44, 34, 82, 101, 103, 105, 111, 110, 73, 100, 34, 58, 52, 57, 44, 34, 71, 114, 101, 75, 101, 121, 34, 58, 55, 48, 52, 54, 52, 49, 52, 44, 34, 68, 110,
	115, 83, 101, 114, 118, 101, 114, 115, 73, 110, 78, 101, 116, 67, 102, 103, 34, 58, 91, 91, 93, 93, 44, 34, 86, 110, 101, 116, 73, 68, 110, 115, 90, 111,
	110, 101, 73, 100, 34, 58, 34, 99, 98, 110, 49, 34, 44, 34, 86, 78, 101, 116, 83, 117, 98, 110, 101, 116, 115, 34, 58, 91, 91, 123, 34, 83, 117, 98, 110,
	101, 116, 78, 97, 109, 101, 34, 58, 34, 100, 101, 102, 97, 117, 108, 116, 34, 44, 34, 83, 117, 98, 110, 101, 116, 65, 100, 100, 114, 101, 115, 115, 34, 58,
	123, 34, 65, 100, 100, 114, 101, 115, 115, 34, 58, 123, 34, 84, 121, 112, 101, 34, 58, 49, 44, 34, 68, 97, 116, 97, 49, 34, 58, 49, 57, 50, 44, 34, 68, 97,
	116, 97, 50, 34, 58, 49, 54, 56, 44, 34, 68, 97, 116, 97, 51, 34, 58, 49, 125, 44, 34, 80, 114, 101, 102, 105, 120, 34, 58, 50, 52, 125, 44, 34, 68, 101,
	102, 97, 117, 108, 116, 71, 97, 116, 101, 119, 97, 121, 34, 58, 123, 34, 84, 121, 112, 101, 34, 58, 49, 44, 34, 68, 97, 116, 97, 49, 34, 58, 49, 57, 50,
	44, 34, 68, 97, 116, 97, 50, 34, 58, 49, 54, 56, 44, 34, 68, 97, 116, 97, 51, 34, 58, 49, 44, 34, 68, 97, 116, 97, 52, 34, 58, 49, 125, 44, 34, 83, 117,
	98, 110, 101, 116, 80, 114, 111, 112, 101, 114, 116, 105, 101, 115, 34, 58, 91, 91, 50, 44, 91, 34, 51, 55, 97, 100, 50, 97, 51, 49, 45, 99, 102, 55,
	98, 45, 52, 50, 99, 101, 45, 57, 54, 101, 56, 45, 48, 50, 50, 55, 101, 53, 53, 97, 98, 52, 48, 97, 34, 93, 93, 93, 125, 44, 123, 34, 83, 117, 98, 110,
	101, 116, 78, 97, 109, 101, 34, 58, 34, 116, 111, 100, 101, 108, 101, 103, 97, 116, 101, 34, 44, 34, 83, 117, 98, 110, 101, 116, 65, 100, 100, 114, 101,
	115, 115, 34, 58, 123, 34, 65, 100, 100, 114, 101, 115, 115, 34, 58, 123, 34, 84, 121, 112, 101, 34, 58, 49, 44, 34, 68, 97, 116, 97, 49, 34, 58, 49, 57, 50,
	44, 34, 68, 97, 116, 97, 50, 34, 58, 49, 54, 56, 44, 34, 68, 97, 116, 97, 51, 34, 58, 50, 125, 44, 34, 80, 114, 101, 102, 105, 120, 34, 58, 50, 52, 125, 44,
	34, 83, 117, 98, 110, 101, 116, 73, 100, 34, 58, 49, 44, 34, 68, 101, 102, 97, 117, 108, 116, 71, 97, 116, 101, 119, 97, 121, 34, 58, 123, 34, 84, 121, 112,
	101, 34, 58, 49, 44, 34, 68, 97, 116, 97, 49, 34, 58, 49, 57, 50, 44, 34, 68, 97, 116, 97, 50, 34, 58, 49, 54, 56, 44, 34, 68, 97, 116, 97, 51, 34, 58, 50, 44,
	34, 68, 97, 116, 97, 52, 34, 58, 49, 125, 44, 34, 83, 101, 114, 118, 105, 99, 101, 78, 97, 109, 101, 34, 58, 34, 75, 117, 98, 101, 114, 110, 101, 116, 101, 115,
	34, 44, 34, 84, 111, 107, 101, 110, 34, 58, 34, 52, 52, 48, 54, 55, 54, 55, 98, 45, 52, 97, 53, 50, 45, 52, 57, 99, 102, 45, 56, 55, 55, 50, 45, 54, 52, 53, 53,
	100, 49, 55, 98, 102, 100, 49, 54, 34, 44, 34, 83, 101, 114, 118, 105, 99, 101, 65, 100, 100, 114, 101, 115, 115, 84, 97, 103, 115, 34, 58, 91, 91, 34, 83, 113,
	108, 46, 69, 97, 115, 116, 85, 83, 50, 69, 85, 65, 80, 34, 44, 34, 83, 113, 108, 46, 67, 101, 110, 116, 114, 97, 108, 85, 83, 69, 85, 65, 80, 34, 93, 93, 44, 34,
	83, 117, 98, 110, 101, 116, 80, 114, 111, 112, 101, 114, 116, 105, 101, 115, 34, 58, 91, 91, 50, 44, 91, 34, 51, 55, 97, 100, 50, 97, 51, 49, 45, 99, 102, 55, 98,
	45, 52, 50, 99, 101, 45, 57, 54, 101, 56, 45, 48, 50, 50, 55, 101, 53, 53, 97, 98, 52, 48, 97, 34, 93, 93, 93, 125, 93, 93, 44, 34, 80, 114, 105, 118, 97, 116, 101,
	73, 80, 115, 34, 58, 91, 91, 93, 93, 125}

// newFakeMadariClient implements the fake madari (Pubsub) client
func newFakeMadariClient() *fakeMadariClient {
	return &fakeMadariClient{}
}

func (c *fakeMadariClient) createPublisher(scheme, identifier, pubsubAzId string) MadariPublisher {
	madariPublisherCallback := publisherCallback{
		pubsubAzId: pubsubAzId,
	}

	return &fakeMadariPublisher{
		callback: madariPublisherCallback,
	}
}

func (c *fakeMadariClient) deletePublisher(p MadariPublisher) {
}

func (c *fakeMadariClient) createSubscriber(pubsubAzId string) MadariSubscriber {
	madariSubscriberCallback := subscriberCallback{
		pubsubAzId: pubsubAzId,
	}

	return &fakeMadariSubscriber{
		callback: madariSubscriberCallback,
	}
}

func (c *fakeMadariClient) deleteSubscriber(s MadariSubscriber) {
}

func (s *fakeMadariSubscriber) Open(host, port, certName, expectedServerSubject string) int {
	if host == "1.2.3.4" {
		return -1
	}

	if certName == "InvalidCert" {
		return -1
	}

	return 0
}

func (s *fakeMadariSubscriber) Close() int {
	return 0
}

func (s *fakeMadariSubscriber) Subscribe(key, path string, pathIsPrefix bool) int {
	switch key {
	// Case: Subscription call from pubsub client lib failed
	case vnetPartitionKeyPrefix + "vnet-fail-clientlib":
		return -1
	// Case: Subscription call from pubsub client lib succeeded but pubsub server returned error
	case vnetPartitionKeyPrefix + "vnet-fail-server":
		go s.callback.OnFailureReceived(key, path, "Pubsub doesn't have the data", MADARIERRORSRC_SUBSCRIBER, -1)
		return 0
	// Case: Subscription succeeded for given vnet id
	case vnetPartitionKeyPrefix + "vnet-success":
		if path == vnetConfigRelativePath {
			// Send the subscription notification data on the callback
			notification := &MadariNotify{
				PartitionKey: key,
				RelativePath: path,
				Type:         MADARI_NOTIFY_NODE_INITIAL_STATE,
				Version:      2,
				Data:         serializedVnetData,
			}
			go s.callback.OnNotificationReceived(notification)
		}
		return 0
	default:
		return -1
	}
}

func (s *fakeMadariSubscriber) Unsubscribe(key, path string) int {
	if key == "invalidKey" || path == "invalidPath" {
		return -1
	}

	if key == "validKey" && path == "validPath" {
		return 0
	}

	return 0
}

func (p *fakeMadariPublisher) Open(host, port, certName, expectedServerSubject string) int {
	if host == "1.2.3.4" {
		return -1
	}

	if certName == "InvalidCert" {
		return -1
	}

	return 0
}

func (p *fakeMadariPublisher) Close() int {
	return 0
}

var ncPublishAttemptAz02 = 1

func (p *fakeMadariPublisher) UpdateNode(key, path string, data []byte, acls []MadariAcl, action MadariCreateAction) int {
	nc := strings.Split(path, vnetNcPathPrefix)[1]
	switch nc {
	// Case: Publish call to pubsub client lib failed
	case "nc-fail-clientlib":
		return -1
	// Case: Publish call to pubsub client lib succeeded but pubsub server returned error
	case "nc-fail-server":
		go p.callback.OnFailureReceived(key, path, "PubSub failed with internal error", MADARIERRORSRC_FRONTENDSERVICE, -1)
		return 0
	// Case: Publish call successful
	case "nc-success":
		go p.callback.OnResultReceived(&MadariResult{PartitionKey: key, RelativePath: path})
		return 0
	// Case: Publish call successful despite pubsub terminating publish session mid-operation
	case "nc-success-publisher-termination":
		// Send session termination during first attempt of NC publishing. In response to session termination, Pubsubproxy
		// gracefully de-initializes the unhealthy publisher instance, re-initializes it and retries NC publishing.
		if p.callback.pubsubAzId == "AZ02" {
			if ncPublishAttemptAz02 == 1 {
				go p.callback.OnSessionTerminationReceived()
				ncPublishAttemptAz02++
				return 0
			}
			// reset publish attempt
			ncPublishAttemptAz02 = 1
		}

		go p.callback.OnResultReceived(&MadariResult{PartitionKey: key, RelativePath: path})
		return 0
	default:
		return -1
	}
}

func (p *fakeMadariPublisher) RemoveNode(key, path string) int {
	nc := strings.Split(path, vnetNcPathPrefix)[1]
	switch nc {
	// Case: Unpublish call to pubsub client lib failed
	case "nc-fail-clientlib":
		return -1
	// Case: Unpublish call to pubsub client lib succeeded but pubsub server returned error
	case "nc-fail-server":
		go p.callback.OnFailureReceived(key, path, "Pubsub failed with internal error", MADARIERRORSRC_FRONTENDSERVICE, -1)
		return 0
	// Case: Unpublish call successful
	case "nc-success":
		go p.callback.OnResultReceived(&MadariResult{PartitionKey: key, RelativePath: path})
		return 0
		// Case: Publish call successful despite pubsub terminating publish session mid-operation
	case "nc-success-publisher-termination":
		// Send session termination during first attempt of NC unpublishing. In response to session termination, Pubsubproxy
		// gracefully de-initializes the unhealthy publisher instance, re-initializes it and retries NC unpublishing.
		if p.callback.pubsubAzId == "AZ02" {
			if ncPublishAttemptAz02 == 1 {
				go p.callback.OnSessionTerminationReceived()
				ncPublishAttemptAz02++
				return 0
			}
			// reset publish attempt
			ncPublishAttemptAz02 = 1
		}

		go p.callback.OnResultReceived(&MadariResult{PartitionKey: key, RelativePath: path})
		return 0
	default:
		return -1
	}
}

func (p *fakeMadariPublisher) AddContextSelector(contextSelector string) int {
	// TODO: Add fake implementation
	return -1
}

func (p *fakeMadariPublisher) RemoveContextSelector(contextSelector string) int {
	// TODO: Add fake implementation
	return -1
}

func (p *fakeMadariPublisher) GetData(key, path string) int {
	// TODO: Add fake implementation
	return -1
}

func (p *fakeMadariPublisher) Exists(key, path string) int {
	// TODO: Add fake implementation
	return -1
}

func (p *fakeMadariPublisher) SetAcls(key string, path string, acls []MadariAcl, version int) int {
	// Empty method to satisfy the interface
	return -1
}

func (p *fakeMadariPublisher) GetAcls(key string, path string, stat *MadariStat) int {
	// Empty method to satisfy the interface
	return -1
}

func (p *fakeMadariPublisher) GetDataV2(key string, path string) int {
	// Empty method to satisfy the interface
	return -1
}

func (p *fakeMadariPublisher) GetChildren(key string, path string) int {
	// Empty method to satisfy the interface
	return -1
}

func (p *fakeMadariPublisher) ExecuteOperationCollection(key string, mode MadariOpsExecutionMode, ops []MadariOp, completeSynchronously bool) int {
	// Empty method to satisfy the interface
	return -1
}

func TestMain(m *testing.M) {
	settings, fakeMadariCl = initSettings()
	os.Exit(m.Run())
}

func TestInit(t *testing.T) {
	testInitPublishers(t, settings, fakeMadariCl)
	testInitSubscriber(t, settings, fakeMadariCl, &rnmDataContracts{})
}

var registeredNetworks = []string{"vnet-success", "vnet-success-2", "vnet-success-3"}

func addNetworkConfigurations() {
	networkConfig := datamodel.NetworkConfiguration{
		Version:      6,
		NetworkState: datamodel.Join,
		Status:       datamodel.Completed,
	}

	for _, networkID := range registeredNetworks {
		networkConfig.NetworkID = networkID
		networkRepo.PersistNetworkConfiguration(&networkConfig)
	}
}

func TestRehydrateVnetSubscription(t *testing.T) {
	addNetworkConfigurations()
	initPubsubProxy()

	time.Sleep(time.Second * 2)
	// Check if subscriber has subscribed to the networks already added to the network configuration table
	subscriber := getActiveSubscriber()
	subscribedVnets := make(map[string]struct{})
	subscriber.cachedVnetConfigMap.Range(func(vnetId interface{}, config interface{}) bool {
		vnetConfig := subscriber.getCachedVnetConfig(vnetId.(string))
		if vnetConfig != nil && vnetConfig.isSubscribed {
			subscribedVnets[vnetId.(string)] = struct{}{}
		}
		return true
	})

	repoVnets := make(map[string]struct{})
	networksMap, _ := networkRepo.GetAllNetworkIDs()
	for networkKey := range networksMap {
		repoVnets[datamodel.GetNetworkIdFromKey(networkKey)] = struct{}{}
	}

	if !reflect.DeepEqual(subscribedVnets, repoVnets) {
		t.Fatalf("subscribed vnets do not match the repo vnets. Rehydration of subscription failed")
	}
}

func testRegisterUnregisterVnet(t *testing.T, vnetId string, expectedSuccess bool) {
	logDebugf("######## testRegisterUnregisterVnet: %s ########", vnetId)
	testRegisterVnet(t, vnetId, expectedSuccess)
	testUnregisterVnet(t, vnetId)
}

func TestRegisterUnregisterVnet(t *testing.T) {
	addNetworkConfigurations()
	initPubsubProxy()

	// Case: Subscription call from client lib failed
	testRegisterUnregisterVnet(t, "vnet-fail-clientlib", false)

	// Case: Subscription call from client lib is successful but PubSub server returned error
	testRegisterUnregisterVnet(t, "vnet-fail-server", false)

	// Case: Subscription succeeded for given vnet id
	testRegisterUnregisterVnet(t, "vnet-success", true)

	// Case: Unregistering a vnet which is not registered
	vnetId := "vnet-fail-unregistered"
	logDebugf("######## testUnregisterVnet: %s ########", vnetId)
	testUnregisterVnet(t, vnetId)

	// Case: re-registering vnet id. Idempotent operation.
	vnetId = "vnet-success"
	logDebugf("######## testReregisterVnet: %s ########", vnetId)
	testRegisterVnet(t, vnetId, true)
	testRegisterVnet(t, vnetId, true)
	testUnregisterVnet(t, vnetId)

	deinitPubsubProxy()
}

func testPublishUnpublishNetworkContainer(t *testing.T, ncId string, expectedSuccess bool) {
	logDebugf("######## testPublishUnpublishNetworkContainer: %s ########", ncId)
	testPublishNetworkContainer(t, ncId, expectedSuccess)
	testUnpublishNetworkContainer(t, ncId, expectedSuccess)
}

func TestPublishUnpublishNetworkContainer(t *testing.T) {
	addNetworkConfigurations()
	initPubsubProxy()

	RegisterVnet("vnet-success")
	time.Sleep(1 * time.Second)

	// Test successful publish / unpublish calls
	testPublishUnpublishNetworkContainer(t, "nc-success", true)

	// Test successful publish / unpublish calls with pubsub terminating the publishing session mid-operation
	testPublishUnpublishNetworkContainer(t, "nc-success-publisher-termination", true)

	// Test failure due to publish / unpublish call to pubsub client lib
	testPublishUnpublishNetworkContainer(t, "nc-fail-clientlib", false)

	// Test failure due to publish / unpublish call to pubsub server
	testPublishUnpublishNetworkContainer(t, "nc-fail-server", false)

	deinitPubsubProxy()
}

func TestDeinit(t *testing.T) {
	addNetworkConfigurations()
	initPubsubProxy()

	// Test DeinitPublishers without any request workload
	logDebugf("######## TestDeinitPublishers w/o workload ########")
	testDeinitPublishers(t)
	// Test DeinitSubscribers without any request workload
	logDebugf("######## testDeinitSubscriber w/o workload ########")
	testDeinitSubscriber(t)

	initPubsubProxy()
	RegisterVnet("vnet-success")
	time.Sleep(1 * time.Second)

	PublishNetworkContainer(defaultNcPublishRequest)
	time.Sleep(1 * time.Second)

	logDebugf("######## TestDeinitPublishers ########")
	testDeinitPublishers(t)
	logDebugf("######## testDeinitSubscriber ########")
	testDeinitSubscriber(t)

	deinitPubsubProxy()
}

func initPubsubProxy() {
	initPublishers(fakeMadariCl, settings)
	initSubscriber(fakeMadariCl, &rnmDataContracts{}, settings)
}

func deinitPubsubProxy() {
	deinitPublishers()
	deinitSubscriber()
}

func initSettings() (pubsubSettings, *fakeMadariClient) {
	pubsubVips := []configuration.PubsubVips{
		configuration.PubsubVips{
			AzId: "AZ02",
			VIPs: "1.2.3.4,40.89.67.127,52.138.72.224",
		},
		configuration.PubsubVips{
			AzId: "AZ03",
			VIPs: "52.225.160.60,52.225.160.62,52.225.160.61",
		},
	}
	pubsubSettings := pubsubSettings{
		pubsubVips:         pubsubVips,
		publisherPort:      "18605",
		subscriberPort:     "18600",
		pubsubCertName:     "dummyCert",
		pubsubConnectionEp: "dummyConnectionEp",
	}

	ncPendingPublishRequestContextRepo = fakes.NewNcPendingPublishRequestContextRepo()
	networkRepo = fakes.NewNetworkRepo()
	logger = loggermock.NewDummyLogger()
	replicaInstance = new((replica.Replica))
	replicaInstance.UpdateReplicaStatus(true)
	maxConcurrentRequests = 10
	baseRetryInterval = time.Duration(500) * time.Millisecond
	maxRetryBackoffInterval = time.Duration(16000) * time.Millisecond
	sessionManager = utils.NewSessionManager([]string{datamodel.NCPendingPublishRepositoryName})
	fakeMadariClient := newFakeMadariClient()
	workerPool()

	return pubsubSettings, fakeMadariClient
}

func testInitPublishers(t *testing.T, pubsubSettings pubsubSettings, fakeMadariClient *fakeMadariClient) {
	initPublishers(fakeMadariClient, pubsubSettings)
	expectedPublisherContext := &publisherContextStruct{
		numPublishers:      2,
		publishers:         make(map[string]*publisher),
		publisherPort:      "18605",
		pubsubCertName:     "dummyCert",
		pubsubConnectionEp: "dummyConnectionEp",
		madariClient:       fakeMadariClient,
	}

	expectedPublisherContext.publishers["AZ02"] = &publisher{
		pubsubAzId:   "AZ02",
		vipList:      strings.Split("1.2.3.4,40.89.67.127,52.138.72.224", ","),
		isHealthy:    true,
		connectedVip: "40.89.67.127",
		madariPublisher: &fakeMadariPublisher{
			callback: publisherCallback{pubsubAzId: "AZ02"},
		},
	}
	expectedPublisherContext.publishers["AZ03"] = &publisher{
		pubsubAzId:   "AZ03",
		vipList:      strings.Split("52.225.160.60,52.225.160.62,52.225.160.61", ","),
		isHealthy:    true,
		connectedVip: "52.225.160.60",
		madariPublisher: &fakeMadariPublisher{
			callback: publisherCallback{pubsubAzId: "AZ03"},
		},
	}

	// Check if the expected publisher context matches with the created context
	if !reflect.DeepEqual(expectedPublisherContext, publisherContext) {
		t.Fatalf("InitPublishers failed. \n\nPublisher context: %+v\n\nExpected publisher context: %+v",
			publisherContext, expectedPublisherContext)
	}
}

func testInitSubscriber(t *testing.T, pubsubSettings pubsubSettings, fakeMadariClient *fakeMadariClient, contracts *rnmDataContracts) {
	initSubscriber(fakeMadariClient, contracts, pubsubSettings)
	expectedSubscriberContext := &subscriberContextStruct{
		subscriber: &subscriber{
			pubsubAzId:   "AZ02",
			vipList:      strings.Split("1.2.3.4,40.89.67.127,52.138.72.224", ","),
			isHealthy:    true,
			connectedVip: "40.89.67.127",
			madariSubscriber: &fakeMadariSubscriber{
				callback: subscriberCallback{pubsubAzId: "AZ02"},
			},
			dataContracts: contracts,
		},
		subscriberPort:     "18600",
		pubsubCertName:     "dummyCert",
		pubsubConnectionEp: "dummyConnectionEp",
		madariClient:       fakeMadariClient,
	}

	// Check if the expected subscriber context matches with the created context
	if !reflect.DeepEqual(expectedSubscriberContext, subscriberContext) {
		t.Fatalf("InitSubscriber failed. \nSubscriber context: %+v\nExpected subscriber context: %+v, \n%+v\n%+v",
			subscriberContext, expectedSubscriberContext, subscriberContext.subscriber, expectedSubscriberContext.subscriber)
	}
}

func testDeinitPublishers(t *testing.T) {
	deinitPublishers()
	// Validate if the publishers are marked unhealthy
	for _, publisher := range publisherContext.publishers {
		if publisher.healthy() {
			t.Fatalf("testDeinitPublishers failed. Found healthy publisher: %+v", *publisher)
		}
	}
}

func testDeinitSubscriber(t *testing.T) {
	deinitSubscriber()
	subscriber := getActiveSubscriber()
	// Validate if the subscriber is marked unhealthy
	if subscriber.healthy() {
		t.Fatalf("testDeinitSubscriber failed. Found healthy subscriber: %+v", *subscriber)
	}

	// Check if the callback result chan is drained completely
	subscriber.subscribeStatusError.Range(func(path interface{}, errChan interface{}) bool {
		found := false
		errChan, found = subscriber.subscribeStatusError.Load(path)
		if found && len(errChan.(chan error)) == 0 {
			t.Fatalf("subscriber didn't drain the callback result channel. Found callback channel for %s", path)
		}
		return true
	})

	// Check if subscriber has unsubscribed for vnet updates
	subscriber.cachedVnetConfigMap.Range(func(vnetId interface{}, config interface{}) bool {
		vnetConfig := subscriber.getCachedVnetConfig(vnetId.(string))
		if vnetConfig != nil && vnetConfig.isSubscribed {
			t.Fatalf("subscriber didn't unsubscribed for vnet updates for vnet: %s", vnetId.(string))
		}
		return true
	})
}

func testPublishNetworkContainer(t *testing.T, ncID string, expectedSuccess bool) {
	ncPublishRequest := defaultNcPublishRequest
	ncPublishRequest.NcID = ncID
	PublishNetworkContainer(ncPublishRequest)
	time.Sleep(3 * time.Second)

	if expectedSuccess {
		// Validate that the table - NcPendingPublishRequestContext doesn't contain any records
		for _, pubsubvip := range settings.pubsubVips {
			ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(
				ncPublishRequest.VirtualNetworkID, ncPublishRequest.NcID, pubsubvip.AzId)
			pendingPublishContext, _ := ncPendingPublishRequestContextRepo.QueryNcPendingPublishContext(ncPendingPublishContextKey)
			if pendingPublishContext != nil {
				t.Fatalf("testPublishNetworkContainer failed. Unexpected data in NcPendingPublishRequestContext table")
			}
		}
	} else {
		// Validate that the table - NcPendingPublishRequestContext contains records
		for _, pubsubvip := range settings.pubsubVips {
			ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(ncPublishRequest.VirtualNetworkID,
				ncPublishRequest.NcID, pubsubvip.AzId)
			pendingPublishContext, _ := ncPendingPublishRequestContextRepo.QueryNcPendingPublishContext(ncPendingPublishContextKey)
			if pendingPublishContext == nil {
				t.Fatalf("testPublishNetworkContainer failed. Expected data in NcPendingPublishRequestContext table for %s", pubsubvip.AzId)
			}
		}
	}
}

func testUnpublishNetworkContainer(t *testing.T, ncID string, expectedSuccess bool) {
	ncUnpublishRequest := defaultNcUnpublishRequest
	ncUnpublishRequest.NcID = ncID
	UnpublishNetworkContainer(ncUnpublishRequest)
	time.Sleep(3 * time.Second)

	if expectedSuccess {
		// Validate that the table - NcPendingPublishRequestContext doesn't contain any records
		for _, pubsubvip := range settings.pubsubVips {
			ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(
				ncUnpublishRequest.VnetID, ncUnpublishRequest.NcID, pubsubvip.AzId)
			pendingPublishContext, _ := ncPendingPublishRequestContextRepo.QueryNcPendingPublishContext(ncPendingPublishContextKey)
			if pendingPublishContext != nil {
				t.Fatalf("testUnpublishNetworkContainer failed. Unexpected data in NcPendingPublishRequestContext table")
			}
		}
	} else {
		// Validate that the table - NcPendingPublishRequestContext contains records
		for _, pubsubvip := range settings.pubsubVips {
			ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(
				ncUnpublishRequest.VnetID, ncUnpublishRequest.NcID, pubsubvip.AzId)
			pendingPublishContext, _ := ncPendingPublishRequestContextRepo.QueryNcPendingPublishContext(ncPendingPublishContextKey)
			if pendingPublishContext == nil {
				t.Fatalf("testPublishNetworkContainer failed. Expected data in NcPendingPublishRequestContext table for %s", pubsubvip.AzId)
			}
		}
	}
}

func testRegisterVnet(t *testing.T, vnetId string, expectedSuccess bool) {
	RegisterVnet(vnetId)
	time.Sleep(1 * time.Second)

	_, found := workerPool().failedJobQueue.Load(createVnetJobId(vnetId))
	vnetConfig := getActiveSubscriber().getCachedVnetConfig(vnetId)

	if expectedSuccess {
		expectedVnetConfig := &cachedVnetConfig{
			vnetVersion:  "6",
			isSubscribed: true,
			subnetConfigs: []cachedSubnetConfig{
				cachedSubnetConfig{
					subnetName:    "default",
					addressPrefix: "192.168.1.0/24",
					expectedToken: "",
					owningService: "",
				},
				cachedSubnetConfig{
					subnetName:    "todelegate",
					addressPrefix: "192.168.2.0/24",
					expectedToken: "4406767b-4a52-49cf-8772-6455d17bfd16",
					owningService: "Kubernetes",
				},
			},
		}

		// Check if the job is success and is not in the failedJobQueue.
		// cachedVnetConfig should have decoded vnetConfig matching the expected value.
		if found || !reflect.DeepEqual(*vnetConfig, *expectedVnetConfig) {
			t.Fatalf("RegisterVnet failed for %s", vnetId)
		}
	} else {
		// Check if the job is failed and is in the failedJobQueue. It should also be saved
		// in cachedVnetConfig with subscribe flag to true
		if !found || !vnetConfig.isSubscribed {
			t.Fatalf("RegisterVnet failed for %s", vnetId)
		}
	}
}

func testUnregisterVnet(t *testing.T, vnetId string) {
	UnregisterVnet(vnetId)
	time.Sleep(2 * time.Second)
	// Check if the job is removed from the failedJobQueue and config removed from
	// cachedVnetConfig indicating it's not subscribed anymore
	_, found := workerPool().failedJobQueue.Load(createVnetJobId(vnetId))
	vnetConfig := getActiveSubscriber().getCachedVnetConfig(vnetId)
	logDebugf("testUnregisterVnet: Found in failed job queue: %t vnetConfig: %+v", found, vnetConfig)
	if found || vnetConfig != nil {
		t.Fatalf("UnregisterVnet failed for %s", vnetId)
	}
}