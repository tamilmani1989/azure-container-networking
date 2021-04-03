// +build enablePubsubProxy

package pubsubproxy

import (
	. "dnc/MadariClient"
	sdn "dnc/microsoft/azure/sdncontracts"
	"fmt"
	"google.golang.org/protobuf/proto"
	"strings"
	"sync"
	"time"
)

const (
	pubsubConnectionRetryInterval = time.Second * 2
)

type publisher struct {
	pubsubAzId      string
	vipList         []string
	isHealthy       bool
	connectedVip    string
	madariPublisher MadariPublisher
	sync.RWMutex
	publishStatusError sync.Map
	publishResults     sync.Map
}

type publisherContextStruct struct {
	numPublishers      int
	publishers         map[string]*publisher
	publisherPort      string
	pubsubCertName     string
	pubsubConnectionEp string
	madariClient       madariClientInterface
}

type madariClientInterface interface {
	createPublisher(scheme string, identifier string, pubsubAzId string) MadariPublisher
	deletePublisher(madariPublisher MadariPublisher)
	createSubscriber(pubsubAzId string) MadariSubscriber
	deleteSubscriber(madariSubscriber MadariSubscriber)
}

type madariClient struct {
}

func newMadariClient() madariClientInterface {
	return &madariClient{}
}

func (mc *madariClient) createPublisher(
	scheme string,
	identifier string,
	pubsubAzId string) MadariPublisher {
	madariPublisherCallback := publisherCallback{
		pubsubAzId: pubsubAzId,
	}

	defaultPublisher := CreateDefaultMadariPublisher(scheme, identifier, madariPublisherCallback)
	return defaultPublisher
}

func (mc *madariClient) deletePublisher(
	mp MadariPublisher) {
	DeleteDefaultMadariPublisher(mp.(DefaultMadariPublisher))
}

func (mc *madariClient) createSubscriber(
	pubsubAzId string) MadariSubscriber {
	madariSubscriberCallback := subscriberCallback{
		pubsubAzId: pubsubAzId,
	}

	defaultSubscriber := CreateDefaultMadariSubscriber(madariSubscriberCallback)
	return defaultSubscriber
}

func (mc *madariClient) deleteSubscriber(
	ms MadariSubscriber) {
	DeleteDefaultMadariSubscriber(ms.(DefaultMadariSubscriber))
}

// This type implements MadariClientCallback interface from MadariClient package
type publisherCallback struct {
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

// use this method if you want to create a channel for a given path to get ONLY error from Madari (pubsub)
func (p *publisher) setupPublishStatusErrorChan(
	path string) {
	if _, found := p.publishStatusError.Load(path); !found {
		p.publishStatusError.Store(path, make(chan error, 1))
	}
}

// method to delete error channel for a given path
func (p *publisher) deletePublishStatusErrorChan(
	path string) {
	if _, found := p.publishStatusError.Load(path); found {
		p.publishStatusError.Delete(path)
	}
}

// use this method if you want to create a channel for a given path to get results from Madari (pubsub)
func (p *publisher) setupPublishResultsChan(
	path string) {
	if _, found := p.publishResults.Load(path); !found {
		p.publishResults.Store(path, make(chan MadariResult, 1))
	}
}

// method to delete result channel for a given path
func (p *publisher) deletePublishResultsChan(
	path string) {
	p.publishResults.Delete(path)
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
		if retCode := p.madariPublisher.UpdateNode(partitionKey, relativePath,
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
		if retCode := p.madariPublisher.RemoveNode(partitionKey, relativePath); retCode != 0 {
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
		if retCode := p.madariPublisher.AddContextSelector(partitionKey); retCode != 0 {
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
		if retCode := p.madariPublisher.RemoveContextSelector(partitionKey); retCode != 0 {
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

func (p *publisher) pathExists(
	partitionKey string,
	relativePath string) error {
	p.Lock()
	defer p.Unlock()

	publishPath := fmt.Sprintf("%s%s", partitionKey, relativePath)
	if p.isHealthy {
		p.setupPublishResultsChan(publishPath)
		if retCode := p.madariPublisher.Exists(partitionKey, relativePath); retCode != 0 {
			p.deletePublishResultsChan(publishPath)
			return fmt.Errorf("pathExists check for %s failed for path: %s due to error code: %d",
				p.pubsubAzId, publishPath, retCode)
		}
	} else {
		return fmt.Errorf("Unhealthy Publisher: %s. pathExists check failed for path: %s",
			p.pubsubAzId, publishPath)
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
		logDebugf("De-initializing publisher: %s", p.pubsubAzId)
		p.isHealthy = false
		p.updatePublishStatusErrorChan()
		p.madariPublisher.Close()
		publisherContext.madariClient.deletePublisher(p.madariPublisher)
		logDebugf("Successfully de-initialized publisher: %s", p.pubsubAzId)
	}
	p.Unlock()
}

func (p *publisher) updatePublishStatusErrorChan() {
	if !p.isHealthy {
		p.publishStatusError.Range(func(publishPath interface{}, errChan interface{}) bool {
			found := false
			errChan, found = p.publishStatusError.Load(publishPath)
			if found && len(errChan.(chan error)) == 0 {
				logDebugf("Publisher: %s sending error on status channel for path: %s", p.pubsubAzId, publishPath.(string))
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

func (p *publisher) sendPublishResults(
	publishPath string,
	result MadariResult) {
	p.Lock()
	defer p.Unlock()
	if p.isHealthy {
		resultChan, found := p.publishResults.Load(publishPath)
		if found && len(resultChan.(chan MadariResult)) == 0 {
			resultChan.(chan MadariResult) <- result
		}
	}
}

func (p *publisher) getPublishResults(
	publishPath string) (MadariResult, error) {
	var result MadariResult
	var err error
	errChan, found := p.publishResults.Load(publishPath)
	if found {
		result = <-errChan.(chan MadariResult)
		err = nil
	} else {
		result = MadariResult{}
		err = fmt.Errorf("Publish result not found for pubsubAzId: %s. path: %s", p.pubsubAzId, publishPath)
	}

	p.Lock()
	defer p.Unlock()
	p.deletePublishStatusErrorChan(publishPath)
	return result, err
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

func pathExists(
	pubsubAzId string,
	partitionKey string,
	relativePath string) (bool, error) {

	if _, found := publisherContext.publishers[pubsubAzId]; !found {
		return false, fmt.Errorf("Invalid pubsubAzId: %s. Number of publishers: %d", pubsubAzId, publisherContext.numPublishers)
	}

	publisher, err := getPublisher(pubsubAzId)
	if err != nil {
		return false, err
	}

	if err = publisher.pathExists(partitionKey, relativePath); err != nil {
		return false, err
	}

	path := fmt.Sprintf("%s%s", partitionKey, relativePath)
	result, err := getPublishResults(pubsubAzId, path)
	if err != nil {
		return false, err
	}

	return result.StatIsValid, err
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

func getPublishResults(
	pubsubAzId string,
	publishPath string) (MadariResult, error) {
	publisher, err := getPublisher(pubsubAzId)
	if err != nil {
		return MadariResult{}, err
	}

	return publisher.getPublishResults(publishPath)
}

func createMadariPublisherCallback(
	pubsubAzId string) publisherCallback {
	return publisherCallback{
		pubsubAzId: pubsubAzId,
	}
}

func (callback publisherCallback) OnNotificationReceived(
	notify *MadariNotify) {
	// Note: Empty implementation to satisfy the interface requirement
}

func (callback publisherCallback) OnResultReceived(
	result *MadariResult) {
	publishPath := fmt.Sprintf("%s%s", result.PartitionKey, result.RelativePath)
	publisher, _ := getPublisher(callback.pubsubAzId)
	//TODO: Consolidate these 2 into just Result channel
	publisher.sendPublishStatusError(publishPath, nil)
	publisher.sendPublishResults(publishPath, *result)

	if verboseLogging {
		displayMadariResult(result, callback.pubsubAzId)
	}
}

func (callback publisherCallback) OnFailureReceived(
	partitionKey string,
	relativePath string,
	reason string,
	errorSource MadariErrorSrc,
	errorCode int32) {
	path := fmt.Sprintf("%s%s", partitionKey, relativePath)
	err := fmt.Errorf("FailureDetails: Code: %d, Src: %v, Cause: %s", errorCode, errorSource, reason)
	logErrorf("OnFailureReceived: Publisher: %s. Path: %s. Error: %v", callback.pubsubAzId, path, err)
	publisher, _ := getPublisher(callback.pubsubAzId)
	publisher.sendPublishStatusError(path, err)
}

func (callback publisherCallback) OnLoggingMessageReceived(
	traceLevel MadariTraceLevel,
	message string) {
	// log only when verbose mode is enabled. Verbose logging can be enabled from config.json
	// and is disabled by default as it can fill up the logs. Use it for debugging
	if verboseLogging {
		logDebugf("OnLoggingMessageReceived: Publisher: %s. Tracelevel: %v. Message: %s", callback.pubsubAzId, traceLevel, message)
	}
}

func (callback publisherCallback) OnHealthNotificationReceived(
	health *MadariApplicationHealth) {
	// Note: Recommendation from PubSub team is to ignore Health notification. They need to
	// establish the usage of this and may deprecate it in future.
}

func (callback publisherCallback) OnSessionTerminationReceived() {
	logErrorf("OnSessionTerminationReceived: Publisher PubSubId: %s", callback.pubsubAzId)

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

			logErrorf("OnSessionTerminationReceived: createOrUpdatePublisher failed for pubsubAzId: %s. Error: %v", publisher.pubsubAzId, err)
			time.Sleep(time.Second * 1)
			//TODO: This calls for a sev2
		}
	}
}

func initPublishers(
	mc madariClientInterface,
	settings pubsubSettings) {
	publisherContext = &publisherContextStruct{
		numPublishers:      len(settings.pubsubVips),
		publishers:         make(map[string]*publisher),
		publisherPort:      settings.publisherPort,
		pubsubCertName:     settings.pubsubCertName,
		pubsubConnectionEp: settings.pubsubConnectionEp,
		madariClient:       mc,
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
					logErrorf("Failed to create publisher: %s. Retriable Error: %v", pubsubAzId, err)
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
		madariPublisher := publisherContext.madariClient.createPublisher(scheme, identifier, publisher.pubsubAzId)
		errorCode := madariPublisher.Open(vip, publisherContext.publisherPort,
			publisherContext.pubsubCertName, publisherContext.pubsubConnectionEp)
		// TODO: Replace 0 with error code enum from pubsub
		if errorCode == 0 {
			logDebugf("Successfully created Publisher %s with pubsub VIP: %s", publisher.pubsubAzId, vip)
			publisher.madariPublisher = madariPublisher
			publisher.isHealthy = true
			publisher.connectedVip = vip
			return nil
		}

		logErrorf("Publisher %s failed to connect to VIP: %s due to error code: %v", publisher.pubsubAzId, vip, errorCode)
	}

	return fmt.Errorf("Failed to create publisher: %s", publisher.pubsubAzId)
}

func deinitPublishers() {
	for _, publisher := range publisherContext.publishers {
		publisher.deinit()
	}
}

func displayMadariResult(result *MadariResult, pubsubAzId string) {
	if strings.HasPrefix(result.PartitionKey, "/containers") &&
		strings.HasPrefix(result.RelativePath, "/config") {

		bmContainer := &sdn.Container{}
		err := proto.Unmarshal(result.Data, bmContainer)
		if err != nil {
			logErrorf("OnResultReceived GetData: Publisher: %s. PartitionKey: %s. Path: %s. Error: %v",
				pubsubAzId, result.PartitionKey, result.RelativePath, err)
			return
		}

		logDebugf("OnResultReceived GetData: Publisher: %s. PartitionKey: %s. Path: %s. Data: %v",
			pubsubAzId, result.PartitionKey, result.RelativePath, bmContainer)
	}

	if strings.HasPrefix(result.PartitionKey, "/networkinterfaces") &&
		strings.HasPrefix(result.RelativePath, "/config") {

		nwInterface := &sdn.NetworkInterface{}
		err := proto.Unmarshal(result.Data, nwInterface)
		if err != nil {
			logErrorf("OnResultReceived GetData: Publisher: %s. PartitionKey: %s. Path: %s. Error: %v",
				pubsubAzId, result.PartitionKey, result.RelativePath, err)
			return
		}

		logDebugf("OnResultReceived GetData: Publisher: %s. PartitionKey: %s. Path: %s. Data: %v",
			pubsubAzId, result.PartitionKey, result.RelativePath, nwInterface)
	}
}
