// +build enablePubsubProxy

package pubsubproxy

import (
	. "dnc/MadariClient"
	"dnc/datamodel"
	"fmt"
	"strings"
	"sync"
	"time"
)

type subscriber struct {
	pubsubAzId       string
	vipList          []string
	isHealthy        bool
	connectedVip     string
	madariSubscriber MadariSubscriber
	sync.RWMutex
	subscribeStatusError sync.Map
	cachedVnetConfigMap  sync.Map
}

type subscriberContextStruct struct {
	subscriber         *subscriber
	subscriberPort     string
	pubsubCertName     string
	pubsubConnectionEp string
	madariClient       madariClientInterface
}

// This type implements MadariClientCallback interface from MadariClient package
type subscriberCallback struct {
	pubsubAzId string
}

type cachedVnetConfig struct {
	vnetVersion   string
	isSubscribed  bool
	subnetConfigs []cachedSubnetConfig
}

type cachedSubnetConfig struct {
	subnetName    string
	addressPrefix string
	expectedToken string
	owningService string
}

var subscriberContext *subscriberContextStruct

func (s *subscriber) setupSubscribeStatusErrorChan(
	path string) {
	if _, found := s.subscribeStatusError.Load(path); !found {
		s.subscribeStatusError.Store(path, make(chan error, 1))
	}
}

func (s *subscriber) deleteSubscribeStatusErrorChan(
	path string) {
	if _, found := s.subscribeStatusError.Load(path); found {
		s.subscribeStatusError.Delete(path)
	}
}

func (s *subscriber) subscribe(
	partitionKey string,
	relativePath string) error {
	s.Lock()
	defer s.Unlock()

	path := partitionKey + relativePath
	if s.isHealthy {
		s.setupSubscribeStatusErrorChan(path)
		if retCode := s.madariSubscriber.Subscribe(partitionKey, relativePath, true); retCode != 0 {
			s.deleteSubscribeStatusErrorChan(path)
			return fmt.Errorf("Subscribing to pubsub: %s failed for path: %s due to error code: %d",
				s.pubsubAzId, path, retCode)
		}
	} else {
		return fmt.Errorf("Unhealthy Subscriber: %s. Subscribing failed for path: %s",
			s.pubsubAzId, path)
	}

	return nil
}

func (s *subscriber) unsubscribe(
	partitionKey string,
	relativePath string) error {
	s.Lock()
	defer s.Unlock()

	path := partitionKey + relativePath
	if s.isHealthy {
		if retCode := s.madariSubscriber.Unsubscribe(partitionKey, relativePath); retCode != 0 {
			return fmt.Errorf("Unsubscribing to pubsub: %s failed for path: %s due to error code: %d",
				s.pubsubAzId, path, retCode)
		}
	} else {
		return fmt.Errorf("Unhealthy Subscriber: %s. Unsubscribing failed for path: %s",
			s.pubsubAzId, path)
	}

	return nil
}

func (s *subscriber) healthy() bool {
	s.RLock()
	defer s.RUnlock()
	return s.isHealthy
}

func (s *subscriber) deinit() {
	s.Lock()
	if s.isHealthy {
		logDebugf("De-initializing Subscriber: %s", s.pubsubAzId)
		s.isHealthy = false
		s.updateSubscribeStatusErrorChan()
		s.madariSubscriber.Close()
		subscriberContext.madariClient.deleteSubscriber(s.madariSubscriber)
		s.unsubscribeCachedVnetConfigs()
		logDebugf("Successfully de-initialized Subscriber: %s", s.pubsubAzId)
	}
	s.Unlock()
}

func (s *subscriber) updateSubscribeStatusErrorChan() {
	if !s.isHealthy {
		s.subscribeStatusError.Range(func(path interface{}, errChan interface{}) bool {
			found := false
			errChan, found = s.subscribeStatusError.Load(path)
			if found && len(errChan.(chan error)) == 0 {
				logDebugf("Subscriber: %s sending error on status channel for path: %s", s.pubsubAzId, path.(string))
				errChan.(chan error) <- fmt.Errorf("Unhealthy Subscriber: %s", s.pubsubAzId)
			}
			return true
		})
	}
}

func (s *subscriber) sendSubscribeStatusError(
	path string,
	err error) {
	s.Lock()
	if s.isHealthy {
		errChan, found := s.subscribeStatusError.Load(path)
		if found && len(errChan.(chan error)) == 0 {
			errChan.(chan error) <- err
		}
	}
	s.Unlock()
}

func (s *subscriber) getSubscribeStatusError(
	path string) error {
	var err error
	errChan, found := s.subscribeStatusError.Load(path)
	if found {
		err = <-errChan.(chan error)
	} else {
		err = fmt.Errorf("Subscribe status error not found for pubsubAzId: %s. path: %s",
			s.pubsubAzId, path)
	}

	s.Lock()
	s.deleteSubscribeStatusErrorChan(path)
	s.Unlock()
	return err
}

func (s *subscriber) getCachedSubnetConfig(
	vnetId string,
	subnetName string) (cachedSubnetConfig, error) {
	vnetConfig, found := s.cachedVnetConfigMap.Load(vnetId)
	if !found {
		return cachedSubnetConfig{}, fmt.Errorf("Configuration not found for VnetId: %s", vnetId)
	}

	for _, subnetConfig := range vnetConfig.(cachedVnetConfig).subnetConfigs {
		if strings.EqualFold(subnetConfig.subnetName, subnetName) {
			return subnetConfig, nil
		}
	}

	return cachedSubnetConfig{}, fmt.Errorf("Subnet: %s not found for VnetId: %s", subnetName, vnetId)
}

func (s *subscriber) getCachedVnetConfig(
	vnetId string) *cachedVnetConfig {
	if vnetConfig, vnetConfigFound := s.cachedVnetConfigMap.Load(vnetId); vnetConfigFound {
		var cachedVnetConfig = vnetConfig.(cachedVnetConfig)
		return &cachedVnetConfig
	}

	return nil
}

func (s *subscriber) deleteCachedVnetConfig(
	vnetId string) {
	s.cachedVnetConfigMap.Delete(vnetId)
}

func (s *subscriber) subscribeCachedVnetConfig(
	vnetId string) {
	vnetConfig := s.getCachedVnetConfig(vnetId)
	if vnetConfig != nil {
		// Mark the existing vnet as subscribed
		vnetConfig.isSubscribed = true
		s.cachedVnetConfigMap.Store(vnetId, *vnetConfig)
		return
	}

	// If vnet config doesn't exist, create an entry as a subscribed vnet
	s.cachedVnetConfigMap.Store(vnetId, cachedVnetConfig{
		isSubscribed: true,
	})
}

func (s *subscriber) unsubscribeCachedVnetConfigs() {
	s.cachedVnetConfigMap.Range(func(vnetId interface{}, config interface{}) bool {
		vnetConfig := s.getCachedVnetConfig(vnetId.(string))
		if vnetConfig != nil {
			vnetConfig.isSubscribed = false
			s.cachedVnetConfigMap.Store(vnetId.(string), *vnetConfig)
		}
		return true
	})
}

func (s *subscriber) updateCachedVnetConfig(
	vnetId string,
	vnetConfig *VnetConfiguration) {
	var cachedVnetConfig cachedVnetConfig
	cachedVnetConfig.vnetVersion = vnetConfig.VnetVersion
	cachedVnetConfig.isSubscribed = true
	for _, subnetConfig := range vnetConfig.SubnetConfigs {
		cachedSubnetConfig := cachedSubnetConfig{
			subnetName:    subnetConfig.SubnetName,
			addressPrefix: subnetConfig.AddressPrefix,
			expectedToken: subnetConfig.ExpectedToken,
			owningService: subnetConfig.OwningService,
		}
		cachedVnetConfig.subnetConfigs = append(cachedVnetConfig.subnetConfigs, cachedSubnetConfig)
	}

	s.cachedVnetConfigMap.Store(vnetId, cachedVnetConfig)
}

func (s *subscriber) rehydrateSubscriptions() {
	logDebugf("Rehydrating vnet subscriptions")
	for {
		networksMap, err := networkRepo.GetAllNetworkIDs()
		if err == nil {
			for networkKey := range networksMap {
				networkID := datamodel.GetNetworkIdFromKey(networkKey)
				RegisterVnet(networkID)
			}
			// Exit after rehydrating to all vnet subscriptions
			return
		} else {
			logErrorf("Failed to rehydrate vnets. Error: %v", err)
			time.Sleep(time.Second * 1)
		}
	}
}

func (callback subscriberCallback) OnNotificationReceived(
	notification *MadariNotify) {
	switch notification.Type {
	case MADARI_NOTIFY_NODE_CHILDREN_CHANGED:
		fallthrough
	case MADARI_NOTIFY_NODE_CREATED:
		fallthrough
	case MADARI_NOTIFY_NODE_INITIAL_STATE:
		fallthrough
	case MADARI_NOTIFY_NODE_DATA_CHANGED:
		// Handle the notification for the data update
		if strings.EqualFold(vnetConfigRelativePath, notification.RelativePath) {
			// If the subscribeStatusError channel is setup, send nil error since
			// the notification is received for that path
			path := notification.PartitionKey + notification.RelativePath
			getActiveSubscriber().sendSubscribeStatusError(path, nil)

			vnetId := getVnetIdFromPartitionKey(notification.PartitionKey)
			vnetUpdate := &vnetJob{
				context: jobContext{
					id:              createVnetJobId(vnetId),
					jobType:         VnetUpdate,
					createdDateTime: time.Now().UTC(),
					retryInterval:   baseRetryInterval,
				},
				vnetId:   vnetId,
				vnetData: notification.Data,
			}
			workerPool().scheduleJob(vnetUpdate)
		}
	case MADARI_NOTIFY_NODE_DELETED:
		// Log the notification for the data deletion
		if strings.EqualFold(vnetConfigRelativePath, notification.RelativePath) {
			logErrorf("Subscriber received delete notification for vnetId: %s on path: %s",
				getVnetIdFromPartitionKey(notification.PartitionKey), notification.RelativePath)
		}
	default:
		logErrorf("Subscriber received unknown notification of type: %d for vnetId: %s on path: %s",
			notification.Type, getVnetIdFromPartitionKey(notification.PartitionKey), notification.RelativePath)
	}
}

func (callback subscriberCallback) OnResultReceived(
	result *MadariResult) {
	// Note: Empty implementation to satisfy the interface requirement
}

func (callback subscriberCallback) OnFailureReceived(
	partitionKey string,
	relativePath string,
	reason string,
	errorSource MadariErrorSrc,
	errorCode int32) {
	path := fmt.Sprintf("%s%s", partitionKey, relativePath)
	err := fmt.Errorf("FailureDetails: Code: %d, Src: %v, Cause: %s", errorCode, errorSource, reason)
	logErrorf("OnFailureReceived: Subscriber: %s. Path: %s. Error: %v", callback.pubsubAzId, path, err)
	getActiveSubscriber().sendSubscribeStatusError(path, err)
}

func (callback subscriberCallback) OnLoggingMessageReceived(
	traceLevel MadariTraceLevel,
	message string) {
	// log only when verbose mode is enabled. Verbose logging can be enabled from config.json
	// and is disabled by default as it can fill up the logs. Use it for debugging
	if verboseLogging {
		logErrorf("OnLoggingMessageReceived: Subscriber: %s. Tracelevel: %v. Message: %s", callback.pubsubAzId, traceLevel, message)
	}
}

func (callback subscriberCallback) OnHealthNotificationReceived(
	health *MadariApplicationHealth) {
	// Note: Recommendation from PubSub team is to ignore Health notification. They need to
	// establish the usage of this and may deprecate it in future.
}

func (callback subscriberCallback) OnSessionTerminationReceived() {
	logErrorf("OnSessionTerminationReceived: Subscriber PubSubId: %s", callback.pubsubAzId)

	// De-initialize the subscriber marking it unhealthy to avoid receiving
	// any further subscribe/unsubscribe requests
	subscriber := getActiveSubscriber()
	if subscriber.healthy() {
		subscriber.deinit()
		// Re-initialize subscriber since this is unexpected session termination.
		for {
			err := createOrUpdateSubscriber(subscriber)
			if err == nil {
				break
			}

			logErrorf("OnSessionTerminationReceived: createOrUpdateSubscriber failed for pubsubAzId: %s. Error: %v", subscriber.pubsubAzId, err)
			time.Sleep(time.Second * 1)
			//TODO: This calls for a sev2
		}
	}
}

// getVnetIdFromPartitionKey - Helper function to get vnet ID from key.
func getVnetIdFromPartitionKey(key string) string {
	// VnetPartitionKey = /vnets/vnetId
	// Splitting on /vnets/ gives [0] = "", [1] = vnetId
	return strings.Split(key, vnetPartitionKeyPrefix)[1]
}

func getActiveSubscriber() *subscriber {
	return subscriberContext.subscriber
}

func initSubscriber(
	mc madariClientInterface,
	settings pubsubSettings) {
	subscriberContext = &subscriberContextStruct{
		subscriberPort:     settings.subscriberPort,
		pubsubCertName:     settings.pubsubCertName,
		pubsubConnectionEp: settings.pubsubConnectionEp,
		madariClient:       mc,
	}

	subscriberContext.subscriber = &subscriber{}

	// Keep retrying until the subscriber is created successfully
	for !getActiveSubscriber().isHealthy {
		for _, pubsubAzVips := range settings.pubsubVips {
			subscriberContext.subscriber.pubsubAzId = pubsubAzVips.AzId
			subscriberContext.subscriber.vipList = strings.Split(pubsubAzVips.VIPs, ",")

			// Only one instance of the subscriber is required. As a result, return when
			// one subscriber instance is successfully created.
			if err := createOrUpdateSubscriber(subscriberContext.subscriber); err == nil {
				break
			}
		}
	}
}

func createOrUpdateSubscriber(
	subscriber *subscriber) error {
	subscriber.Lock()
	defer subscriber.Unlock()

	// Loop through the available VIP list
	for _, vip := range subscriber.vipList {
		madariSubscriber := subscriberContext.madariClient.createSubscriber(subscriber.pubsubAzId)
		errorCode := madariSubscriber.Open(vip, subscriberContext.subscriberPort,
			subscriberContext.pubsubCertName, subscriberContext.pubsubConnectionEp)
		// TODO: Replace 0 with error code enum from pubsub
		if errorCode == 0 {
			logDebugf("Successfully created subscriber %s with pubsub VIP: %s", subscriber.pubsubAzId, vip)
			subscriber.madariSubscriber = madariSubscriber
			subscriber.isHealthy = true
			subscriber.connectedVip = vip
			// Rehydrate the subscriptions for the networks which are already joined
			subscriber.rehydrateSubscriptions()

			return nil
		}

		logErrorf("Subscriber %s failed to connect to VIP: %s due to error code: %v", subscriber.pubsubAzId, vip, errorCode)
	}

	return fmt.Errorf("Failed to create subscriber: %s", subscriber.pubsubAzId)
}

func deinitSubscriber() {
	getActiveSubscriber().deinit()
}
