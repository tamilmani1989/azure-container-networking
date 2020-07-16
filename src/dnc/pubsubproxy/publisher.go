// +build enablePubsubProxy

package pubsubproxy

import (
	. "dnc/MadariClient"
	"dnc/logger/loggershim"
	"fmt"
	"sync"
	"time"
)

const (
	AZ01 = iota + 1
	AZ02
	AZ03
)

type publisher struct {
	pubsubId  int
	isHealthy bool
	madariPublisher
	sync.RWMutex
	publishStatusError sync.Map
}

type publisherContextStruct struct {
	numPublishers int
	publishers    map[int]*publisher
}

type madariPublisher struct {
	defaultPublisher DefaultMadariPublisher
	callback         madariPublisherCallback
}

type madariPublisherCallback struct {
	pubsubId int
}

var publisherContext *publisherContextStruct

func getPublisher(
	pubsubId int) (*publisher, error) {
	if pubsubId > publisherContext.numPublishers {
		return nil, fmt.Errorf("Invalid pubsubId: %d. Number of publishers: %d",
			pubsubId, publisherContext.numPublishers)
	}

	return publisherContext.publishers[pubsubId], nil
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
			return fmt.Errorf("Publishing to pubsub: %d failed for path: %s due to error code: %d",
				p.pubsubId, publishPath, retCode)
		}
	} else {
		return fmt.Errorf("Unhealthy Publisher: %d. Publishing failed for path: %s",
			p.pubsubId, publishPath)
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
			return fmt.Errorf("Unpublishing to pubsub: %d failed for path: %s due to error code: %d",
				p.pubsubId, unpublishPath, retCode)
		}
	} else {
		return fmt.Errorf("Unhealthy Publisher: %d. Unublishing failed for path: %s",
			p.pubsubId, unpublishPath)
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
		loggershim.LS.Debugf("[PubSubProxy] Deinitializing Publisher: %d", p.pubsubId)
		p.isHealthy = false
		p.updatePublishStatusErrorChan()
		p.defaultPublisher.Close()
		DeleteDefaultMadariPublisher(p.defaultPublisher)
		loggershim.LS.Debugf("[PubSubProxy] Successfully deinitialized publisher: %d", p.pubsubId)
	}
	p.Unlock()
}

func (p *publisher) updatePublishStatusErrorChan() {
	if !p.isHealthy {
		p.publishStatusError.Range(func(publishPath interface{}, errChan interface{}) bool {
			found := false
			errChan, found = p.publishStatusError.Load(publishPath)
			if found && len(errChan.(chan error)) == 0 {
				loggershim.LS.Debugf("[PubSubProxy] Publisher: %d sending error on status channel for path: %s",
					p.pubsubId, publishPath.(string))
				errChan.(chan error) <- fmt.Errorf("Unhealthy Publisher: %d", p.pubsubId)
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
		err = fmt.Errorf("Publish status error not found for pubsubId: %d. path: %s", p.pubsubId, publishPath)
	}

	p.Lock()
	p.deletePublishStatusErrorChan(publishPath)
	p.Unlock()
	return err
}

func publishData(
	pubsubId int,
	partitionKey string,
	relativePath string,
	data []byte) error {
	if pubsubId > publisherContext.numPublishers {
		return fmt.Errorf("Invalid pubsubId: %d. Number of publishers: %d", pubsubId, publisherContext.numPublishers)
	}

	publisher, err := getPublisher(pubsubId)
	if err != nil {
		return err
	}

	return publisher.publishData(partitionKey, relativePath, data)
}

func unpublishData(
	pubsubId int,
	partitionKey string,
	relativePath string) error {
	if pubsubId > publisherContext.numPublishers {
		return fmt.Errorf("Invalid pubsubId: %d. Number of publishers: %d", pubsubId, publisherContext.numPublishers)
	}

	publisher, err := getPublisher(pubsubId)
	if err != nil {
		return err
	}

	return publisher.unpublishData(partitionKey, relativePath)
}

func getPublishStatusError(
	pubsubId int,
	publishPath string) error {
	publisher, err := getPublisher(pubsubId)
	if err != nil {
		return err
	}

	return publisher.getPublishStatusError(publishPath)
}

func createMadariPublisherCallback(
	pubsubId int) madariPublisherCallback {
	return madariPublisherCallback{
		pubsubId: pubsubId,
	}
}

func (callback madariPublisherCallback) OnNotificationReceived(
	notify *MadariNotify) {
	// Note: Empty implementation to satisfy the interface requirement
}

func (callback madariPublisherCallback) OnResultReceived(
	result *MadariResult) {
	publishPath := fmt.Sprintf("%s%s", result.PartitionKey, result.RelativePath)
	publisher, _ := getPublisher(callback.pubsubId)
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
	loggershim.LS.Errorf("[PubSubProxy] OnFailureReceived: Publisher: %d. Path: %s. Error: %v",
		callback.pubsubId, path, err)
	publisher, _ := getPublisher(callback.pubsubId)
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
	loggershim.LS.Errorf("[PubSubProxy] OnSessionTerminationReceived: Publisher PubSubId: %d",
		callback.pubsubId)

	// Deinitialize the publisher marking it unhealthy to avoid receiving
	// any further publish/unpublish requests
	publisher, _ := getPublisher(callback.pubsubId)
	if publisher.healthy() {
		publisher.deinit()
		// Re-initialize publisher since this is unexpected session termination.
		for {
			err := createOrUpdatePublisher(publisher, getVipList(publisher.pubsubId))
			if err == nil {
				break
			}

			loggershim.LS.Errorf("[PubSubProxy] OnSessionTerminationReceived: "+
				"createOrUpdatePublisher failed for pubsubId: %d due to error: %v", publisher.pubsubId, err)
			time.Sleep(time.Second * 1)
			//TODO: This calls for a sev2
		}
	}
}

func initPublishers() error {
	publisherContext = &publisherContextStruct{
		numPublishers: numPubSubs,
		publishers:    make(map[int]*publisher),
	}

	for pubsubId := AZ01; pubsubId <= numPubSubs; pubsubId++ {
		publisherContext.publishers[pubsubId] = &publisher{
			pubsubId: pubsubId,
		}

		err := createOrUpdatePublisher(publisherContext.publishers[pubsubId], getVipList(pubsubId))
		if err != nil {
			return err
		}
	}

	return nil
}

func createOrUpdatePublisher(
	publisher *publisher,
	vipList []string) error {
	publisher.Lock()
	defer publisher.Unlock()

	// Loop through the available VIP list
	for vipId := 0; vipId < len(vipList); vipId++ {
		madariPublisherCallback := createMadariPublisherCallback(publisher.pubsubId)
		defaultPublisher := CreateDefaultMadariPublisher(scheme, identifier, madariPublisherCallback)
		errorCode := defaultPublisher.Open(vipList[vipId], publisherPort, pubsubCertName, pubsubConnectionEp)
		// TODO: Replace 0 with error code enum from pubsub
		if errorCode == 0 {
			loggershim.LS.Debugf("[PubSubProxy] Successfully created Publisher with pubsub VIP: %s", vipList[vipId])
			publisher.defaultPublisher = defaultPublisher
			publisher.callback = madariPublisherCallback
			publisher.isHealthy = true
			return nil
		}

		loggershim.LS.Errorf("[PubSubProxy] Failed to connect to VIP: %s due to error code: %v", vipList[vipId], errorCode)
	}

	return fmt.Errorf("Failed to create publisher: %d", publisher.pubsubId)
}

func deinitPublishers() {
	for _, publisher := range publisherContext.publishers {
		publisher.deinit()
	}
}
