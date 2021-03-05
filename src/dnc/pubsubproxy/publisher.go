// +build enablePubsubProxy

package pubsubproxy

import (
	. "dnc/MadariClient"
	"dnc/logger/loggershim"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	pubsubConnectionRetryInterval = time.Second * 2
)

type publisher struct {
	pubsubAzId   string
	vipList      []string
	isHealthy    bool
	connectedVip string
	madariPublisher
	sync.RWMutex
	publishStatusError sync.Map
}

type publisherContextStruct struct {
	numPublishers      int
	publishers         map[string]*publisher
	publisherPort      string
	pubsubCertName     string
	pubsubConnectionEp string
}

type madariPublisher struct {
	defaultPublisher DefaultMadariPublisher
	callback         madariPublisherCallback
}

type madariPublisherCallback struct {
	pubsubAzId string
}

var publisherContext *publisherContextStruct

func getPublisher(
	pubsubAzId string) (*publisher, error) {
	if _, found := publisherContext.publishers[pubsubAzId]; !found {
		return nil, fmt.Errorf("Invalid pubsubAzId: %s. Number of publishers: %d",
			pubsubAzId, publisherContext.numPublishers)
	}

	return publisherContext.publishers[pubsubAzId], nil
}

func getAllPublishers() map[string]*publisher {
	return publisherContext.publishers
}

func (p *publisher) setupPublishStatusErrorChan(
	path string) {
	if _, found := p.publishStatusError.Load(path); !found {
		p.publishStatusError.Store(path, make(chan error, 1))
	}
}

func (p *publisher) deletePublishStatusErrorChan(
	path string) {
	if _, found := p.publishStatusError.Load(path); found {
		p.publishStatusError.Delete(path)
	}
}

func (p *publisher) publishData(
	partitionKey string,
	relativePath string,
	data []byte) error {
	p.Lock()
	defer p.Unlock()

	publishPath := partitionKey + relativePath
	if p.isHealthy {
		p.setupPublishStatusErrorChan(publishPath)
		if retCode := p.defaultPublisher.UpdateNode(partitionKey, relativePath,
			data, nil, MADARICREATEACTION_CREATEIFNOTEXISTS); retCode != 0 {
			p.deletePublishStatusErrorChan(publishPath)
			return fmt.Errorf("Publishing to pubsub: %s failed for path: %s due to error code: %d",
				p.pubsubAzId, publishPath, retCode)
		}
	} else {
		return fmt.Errorf("Unhealthy Publisher: %s. Publishing failed for path: %s",
			p.pubsubAzId, publishPath)
	}

	return nil
}

func (p *publisher) unpublishData(
	partitionKey string,
	relativePath string) error {
	p.Lock()
	defer p.Unlock()

	unpublishPath := partitionKey + relativePath
	if p.isHealthy {
		p.setupPublishStatusErrorChan(unpublishPath)
		if retCode := p.defaultPublisher.RemoveNode(partitionKey, relativePath); retCode != 0 {
			p.deletePublishStatusErrorChan(unpublishPath)
			return fmt.Errorf("Unpublishing to pubsub: %s failed for path: %s due to error code: %d",
				p.pubsubAzId, unpublishPath, retCode)
		}
	} else {
		return fmt.Errorf("Unhealthy Publisher: %s. Unublishing failed for path: %s",
			p.pubsubAzId, unpublishPath)
	}

	return nil
}

func (p *publisher) createContextSelector(
	partitionKey string) error {
	p.Lock()
	defer p.Unlock()

	if p.isHealthy {
		p.setupPublishStatusErrorChan(partitionKey)
		if retCode := p.defaultPublisher.AddContextSelector(partitionKey); retCode != 0 {
			p.deletePublishStatusErrorChan(partitionKey)
			return fmt.Errorf("Failed to create context selector %s in pubsub %s due to error code %d",
				partitionKey, p.pubsubAzId, retCode)
		}
	} else {
		return fmt.Errorf("Unhealthy Publisher: %s. Failed to create context selector %s",
			p.pubsubAzId, partitionKey)
	}

	return nil
}

func (p *publisher) removeContextSelector(
	partitionKey string) error {
	p.Lock()
	defer p.Unlock()

	if p.isHealthy {
		p.setupPublishStatusErrorChan(partitionKey)
		if retCode := p.defaultPublisher.RemoveContextSelector(partitionKey); retCode != 0 {
			p.deletePublishStatusErrorChan(partitionKey)
			return fmt.Errorf("Failed to delete context selector %s in pubsub %s due to error code %d",
				partitionKey, p.pubsubAzId, retCode)
		}
	} else {
		return fmt.Errorf("Unhealthy Publisher: %s. Failed to delete context selector %s",
			p.pubsubAzId, partitionKey)
	}

	return nil
}

func (p *publisher) healthy() bool {
	p.RLock()
	defer p.RUnlock()
	return p.isHealthy
}

func (p *publisher) deinit() {
	p.Lock()
	if p.isHealthy {
		loggershim.LS.Debugf("[PubSubProxy] Deinitializing publisher: %s", p.pubsubAzId)
		p.isHealthy = false
		p.updatePublishStatusErrorChan()
		p.defaultPublisher.Close()
		DeleteDefaultMadariPublisher(p.defaultPublisher)
		loggershim.LS.Debugf("[PubSubProxy] Successfully deinitialized publisher: %s", p.pubsubAzId)
	}
	p.Unlock()
}

func (p *publisher) updatePublishStatusErrorChan() {
	if !p.isHealthy {
		p.publishStatusError.Range(func(publishPath interface{}, errChan interface{}) bool {
			found := false
			errChan, found = p.publishStatusError.Load(publishPath)
			if found && len(errChan.(chan error)) == 0 {
				loggershim.LS.Debugf("[PubSubProxy] Publisher: %s sending error on status channel for path: %s",
					p.pubsubAzId, publishPath.(string))
				errChan.(chan error) <- fmt.Errorf("Unhealthy Publisher: %s", p.pubsubAzId)
			}
			return true
		})
	}
}

func (p *publisher) sendPublishStatusError(
	publishPath string,
	err error) {
	p.Lock()
	if p.isHealthy {
		errChan, found := p.publishStatusError.Load(publishPath)
		if found && len(errChan.(chan error)) == 0 {
			errChan.(chan error) <- err
		}
	}
	p.Unlock()
}

func (p *publisher) getPublishStatusError(
	publishPath string) error {
	var err error
	errChan, found := p.publishStatusError.Load(publishPath)
	if found {
		err = <-errChan.(chan error)
	} else {
		err = fmt.Errorf("Publish status error not found for pubsubAzId: %s. path: %s", p.pubsubAzId, publishPath)
	}

	p.Lock()
	p.deletePublishStatusErrorChan(publishPath)
	p.Unlock()
	return err
}

func publishData(
	pubsubAzId string,
	partitionKey string,
	relativePath string,
	data []byte) error {
	if _, found := publisherContext.publishers[pubsubAzId]; !found {
		return fmt.Errorf("Invalid pubsubAzId: %s. Number of publishers: %d", pubsubAzId, publisherContext.numPublishers)
	}

	publisher, err := getPublisher(pubsubAzId)
	if err != nil {
		return err
	}

	return publisher.publishData(partitionKey, relativePath, data)
}

func unpublishData(
	pubsubAzId string,
	partitionKey string,
	relativePath string) error {
	if _, found := publisherContext.publishers[pubsubAzId]; !found {
		return fmt.Errorf("Invalid pubsubAzId: %s. Number of publishers: %d", pubsubAzId, publisherContext.numPublishers)
	}

	publisher, err := getPublisher(pubsubAzId)
	if err != nil {
		return err
	}

	return publisher.unpublishData(partitionKey, relativePath)
}

func createContextSelector(
	pubsubAzId string,
	partitionKey string) error {
	if _, found := publisherContext.publishers[pubsubAzId]; !found {
		return fmt.Errorf("Invalid pubsubAzId: %s. Number of publishers: %d", pubsubAzId, publisherContext.numPublishers)
	}

	publisher, err := getPublisher(pubsubAzId)
	if err != nil {
		return err
	}

	return publisher.createContextSelector(partitionKey)
}

func removeContextSelector(
	pubsubAzId string,
	partitionKey string) error {
	if _, found := publisherContext.publishers[pubsubAzId]; !found {
		return fmt.Errorf("Invalid pubsubAzId: %s. Number of publishers: %d", pubsubAzId, publisherContext.numPublishers)
	}

	publisher, err := getPublisher(pubsubAzId)
	if err != nil {
		return err
	}

	return publisher.removeContextSelector(partitionKey)
}

func getPublishStatusError(
	pubsubAzId string,
	publishPath string) error {
	publisher, err := getPublisher(pubsubAzId)
	if err != nil {
		return err
	}

	return publisher.getPublishStatusError(publishPath)
}

func createMadariPublisherCallback(
	pubsubAzId string) madariPublisherCallback {
	return madariPublisherCallback{
		pubsubAzId: pubsubAzId,
	}
}

func (callback madariPublisherCallback) OnNotificationReceived(
	notify *MadariNotify) {
	// Note: Empty implementation to satisfy the interface requirement
}

func (callback madariPublisherCallback) OnResultReceived(
	result *MadariResult) {
	publishPath := fmt.Sprintf("%s%s", result.PartitionKey, result.RelativePath)
	publisher, _ := getPublisher(callback.pubsubAzId)
	publisher.sendPublishStatusError(publishPath, nil)
}

func (callback madariPublisherCallback) OnFailureReceived(
	partitionKey string,
	relativePath string,
	reason string,
	errorSource MadariErrorSrc,
	errorCode int32) {
	path := fmt.Sprintf("%s%s", partitionKey, relativePath)
	err := fmt.Errorf("FailureDetails: Code: %d, Src: %v, Cause: %s", errorCode, errorSource, reason)
	loggershim.LS.Errorf("[PubSubProxy] OnFailureReceived: Publisher: %s. Path: %s. Error: %v",
		callback.pubsubAzId, path, err)
	publisher, _ := getPublisher(callback.pubsubAzId)
	publisher.sendPublishStatusError(path, err)
}

func (callback madariPublisherCallback) OnLoggingMessageReceived(
	traceLevel MadariTraceLevel,
	message string) {
	// Note: Empty implementation to satisfy the interface requirement
}

func (callback madariPublisherCallback) OnHealthNotificationReceived(
	health *MadariApplicationHealth) {
	// Note: Recommendation from PubSub team is to ignore Health notification. They need to
	// establish the usage of this and may deprecate it in future.
}

func (callback madariPublisherCallback) OnSessionTerminationReceived() {
	loggershim.LS.Errorf("[PubSubProxy] OnSessionTerminationReceived: Publisher PubSubId: %s",
		callback.pubsubAzId)

	// Deinitialize the publisher marking it unhealthy to avoid receiving
	// any further publish/unpublish requests
	publisher, _ := getPublisher(callback.pubsubAzId)
	if publisher.healthy() {
		publisher.deinit()
		// Re-initialize publisher since this is unexpected session termination.
		for {
			err := createOrUpdatePublisher(publisher)
			if err == nil {
				break
			}

			loggershim.LS.Errorf("[PubSubProxy] OnSessionTerminationReceived: "+
				"createOrUpdatePublisher failed for pubsubAzId: %s due to error: %v", publisher.pubsubAzId, err)
			time.Sleep(time.Second * 1)
			//TODO: This calls for a sev2
		}
	}
}

func initPublishers(
	settings pubsubSettings) {
	publisherContext = &publisherContextStruct{
		numPublishers:      len(settings.pubsubVips),
		publishers:         make(map[string]*publisher),
		publisherPort:      settings.publisherPort,
		pubsubCertName:     settings.pubsubCertName,
		pubsubConnectionEp: settings.pubsubConnectionEp,
	}

	// Initialize the publishers
	for _, pubsubVips := range settings.pubsubVips {
		publisherContext.publishers[pubsubVips.AzId] = &publisher{
			pubsubAzId: pubsubVips.AzId,
			vipList:    strings.Split(pubsubVips.VIPs, ","),
		}
	}

	// Create all the publishers
	// Keep retrying until the publisher is created successfully
	createdPublishers := make(map[string]struct{})
	for len(createdPublishers) < publisherContext.numPublishers {
		for pubsubAzId, _ := range publisherContext.publishers {
			if _, created := createdPublishers[pubsubAzId]; !created {
				err := createOrUpdatePublisher(publisherContext.publishers[pubsubAzId])
				if err == nil {
					createdPublishers[pubsubAzId] = struct{}{}
				} else {
					loggershim.LS.Errorf("[PubSubProxy] Retriable error: %v", err)
					time.Sleep(pubsubConnectionRetryInterval)
				}
			}
		}
	}
}

func createOrUpdatePublisher(
	publisher *publisher) error {
	publisher.Lock()
	defer publisher.Unlock()

	// Loop through the available VIP list
	for _, vip := range publisher.vipList {
		madariPublisherCallback := createMadariPublisherCallback(publisher.pubsubAzId)
		defaultPublisher := CreateDefaultMadariPublisher(scheme, identifier, madariPublisherCallback)
		errorCode := defaultPublisher.Open(vip, publisherContext.publisherPort,
			publisherContext.pubsubCertName, publisherContext.pubsubConnectionEp)
		// TODO: Replace 0 with error code enum from pubsub
		if errorCode == 0 {
			loggershim.LS.Debugf("[PubSubProxy] Successfully created Publisher %s with pubsub VIP: %s", publisher.pubsubAzId, vip)
			publisher.defaultPublisher = defaultPublisher
			publisher.callback = madariPublisherCallback
			publisher.isHealthy = true
			publisher.connectedVip = vip
			return nil
		}

		loggershim.LS.Errorf("[PubSubProxy] Publisher %s failed to connect to VIP: %s due to error code: %v",
			publisher.pubsubAzId, vip, errorCode)
	}

	return fmt.Errorf("Failed to create publisher: %s", publisher.pubsubAzId)
}

func deinitPublishers() {
	for _, publisher := range publisherContext.publishers {
		publisher.deinit()
	}
}