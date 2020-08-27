// +build enablePubsubProxy

package pubsubproxy

import (
	. "dnc/MadariClient"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	rehydrateSubscriptionInterval = time.Second
)

type subscriber struct {
	pubsubAzId      string
	vipList         []string
	isHealthy       bool
	stopRehydration chan bool
	connectedVip string
	madariSubscriber
	sync.RWMutex
	subscribeStatusError sync.Map
	cachedVnetConfigMap  sync.Map
}

type subscriberContextStruct struct {
	subscriber         *subscriber
	subscriberPort     string
	pubsubCertName     string
	pubsubConnectionEp string
}

type madariSubscriber struct {
	defaultSubscriber DefaultMadariSubscriber
	callback          subscriberCallback
}

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

func createSubscriberCallback(
	pubsubAzId string) subscriberCallback {
	return subscriberCallback{
		pubsubAzId: pubsubAzId,
	}
}

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
		if retCode := s.defaultSubscriber.Subscribe(partitionKey, relativePath, true); retCode != 0 {
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
		if retCode := s.defaultSubscriber.Unsubscribe(partitionKey, relativePath); retCode != 0 {
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
		loggershim.LS.Debugf("[PubSubProxy] Deinitializing Subscriber: %s", s.pubsubAzId)
		s.isHealthy = false
		s.updateSubscribeStatusErrorChan()
		s.defaultSubscriber.Close()
		DeleteDefaultMadariSubscriber(s.defaultSubscriber)
		s.stopRehydration <- true
		s.unsubscribeCachedVnetConfigs()
		loggershim.LS.Debugf("[PubSubProxy] Successfully deinitialized Subscriber: %s", s.pubsubAzId)
	}
	s.Unlock()
}

func (s *subscriber) updateSubscribeStatusErrorChan() {
	if !s.isHealthy {
		s.subscribeStatusError.Range(func(path interface{}, errChan interface{}) bool {
			found := false
			errChan, found = s.subscribeStatusError.Load(path)
			if found && len(errChan.(chan error)) == 0 {
				loggershim.LS.Debugf("[PubSubProxy] Subscriber: %s sending error on status channel for path: %s",
					s.pubsubAzId, path.(string))
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
	loggershim.LS.Debugf("[PubSubProxy] Rehydrating subscriptions")
	timerChannel := time.NewTicker(rehydrateSubscriptionInterval).C
	for {
		select {
		case <-s.stopRehydration:
			loggershim.LS.Debugf("[PubSubProxy] Stopping rehydration of subscriptions")
			return
		case <-timerChannel:
			networksMap, err := datamodel.GetAllNetworkIDs()
			if err == nil {
				for networkKey := range networksMap {
					networkID := datamodel.GetNetworkIdFromKey(networkKey)
					RegisterVnet(networkID)
				}
			} else {
				loggershim.LS.Errorf("[PubSubProxy] Failed to rehydrate vnets due to error %v", err)
			}
		}
	}
}

func (callback subscriberCallback) OnNotificationReceived(
	notification *MadariNotify) {
	switch notification.Type {
	case MADARI_NOTIFY_NODE_INITIAL_STATE:
		fallthrough
	case MADARI_NOTIFY_NODE_DATA_CHANGED:
		// Handle the notification for the data update
		if strings.EqualFold(subnetConfigRelativePath, notification.RelativePath) ||
			strings.EqualFold(vnetConfigRelativePath, notification.RelativePath) {
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
		if strings.EqualFold(subnetConfigRelativePath, notification.RelativePath) ||
			strings.EqualFold(vnetConfigRelativePath, notification.RelativePath) {
			loggershim.LS.Errorf("[PubSubProxy] Subscriber received delete notification for vnetId: %s on path: %s",
				getVnetIdFromPartitionKey(notification.PartitionKey), notification.RelativePath)
		}
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
	loggershim.LS.Errorf("[PubSubProxy] OnFailureReceived: Subscriber: %s. Path: %s. Error: %v",
		callback.pubsubAzId, path, err)
	getActiveSubscriber().sendSubscribeStatusError(path, err)
}

func (callback subscriberCallback) OnLoggingMessageReceived(
	traceLevel MadariTraceLevel,
	message string) {
	// Note: Empty implementation to satisfy the interface requirement
}

func (callback subscriberCallback) OnHealthNotificationReceived(
	health *MadariApplicationHealth) {
	// Note: Recommendation from PubSub team is to ignore Health notification. They need to
	// establish the usage of this and may deprecate it in future.
}

func (callback subscriberCallback) OnSessionTerminationReceived() {
	loggershim.LS.Errorf("[PubSubProxy] OnSessionTerminationReceived: Subscriber PubSubId: %s",
		callback.pubsubAzId)

	// Deinitialize the subscriber marking it unhealthy to avoid receiving
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

			loggershim.LS.Errorf("[PubSubProxy] OnSessionTerminationReceived: "+
				"createOrUpdateSubscriber failed for pubsubAzId: %s due to error: %v", subscriber.pubsubAzId, err)
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
	settings pubsubSettings) {
	subscriberContext = &subscriberContextStruct{
		subscriberPort:     settings.subscriberPort,
		pubsubCertName:     settings.pubsubCertName,
		pubsubConnectionEp: settings.pubsubConnectionEp,
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
		subscriberCallback := createSubscriberCallback(subscriber.pubsubAzId)
		defaultSubscriber := CreateDefaultMadariSubscriber(subscriberCallback)
		errorCode := defaultSubscriber.Open(vip, subscriberContext.subscriberPort,
			subscriberContext.pubsubCertName, subscriberContext.pubsubConnectionEp)
		// TODO: Replace 0 with error code enum from pubsub
		if errorCode == 0 {
			loggershim.LS.Debugf("[PubSubProxy] Successfully created subscriber %s with pubsub VIP: %s", subscriber.pubsubAzId, vip)
			subscriber.defaultSubscriber = defaultSubscriber
			subscriber.callback = subscriberCallback
			subscriber.isHealthy = true
			subscriber.stopRehydration = make(chan bool)
			subscriber.connectedVip = vip
			// Start the subscription rehydration loop
			go subscriber.rehydrateSubscriptions()

			return nil
		}

		loggershim.LS.Errorf("[PubSubProxy] Subscriber %s failed to connect to VIP: %s due to error code: %v",
			subscriber.pubsubAzId, vip, errorCode)
	}

	return fmt.Errorf("Failed to create subscriber: %s", subscriber.pubsubAzId)
}

func deinitSubscriber() {
	getActiveSubscriber().deinit()
}
