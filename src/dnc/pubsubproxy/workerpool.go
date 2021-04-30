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

size_t SizeofSubnetConfiguration() {
    return sizeof(struct SubnetConfiguration);
}

size_t SizeofTagConfiguration() {
    return sizeof(struct TagConfiguration);
}
*/
import "C"

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/nmagentcontracts"
	"dnc/resourceupdatehandler/networkupdatehandler"
	"fmt"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

var (
	wp   *workerPoolContext
	once sync.Once
)

const (
	ncPendingRequestProcessInterval = time.Second
)

type jobType int

// Job types
const (
	NetworkContainerPublish jobType = iota
	NetworkContainerUnpublish
	VnetRegister
	VnetUnregister
	VnetUpdate
	BareMetalContainerPublish
	BareMetalContainerUnpublish
	SubnetRegister
	SubnetUnregister
	SubnetUpdate
	InvalidJobType
)

// TO DO: Consider make it as Context Selector, and
// an array of relative path + data
// this it is easy to process it as a batch, check/create context selector first,
// and then publish data. for Unpublish, process this as unit in reverse
// unpublish first, and then remove context selector.
type pubsubContext struct {
	Key  string
	Path string
	Data []byte
}

// jobContext contains the info about the job
type jobContext struct {
	id                  string
	jobType             jobType
	pubsubAzId          string
	partitionKey        string
	relativePath        string
	lastRetriedDateTime time.Time
	createdDateTime     time.Time
	retryInterval       time.Duration
	secondarys          []pubsubContext
}

// Job define job interface
type Job interface {
	process() error
	getContext() *jobContext
}

// ncJob
type ncJob struct {
	context jobContext
	ncId    string
	vnetId  string
	ncData  []byte
}

// vnetJob
type vnetJob struct {
	context  jobContext
	vnetId   string
	vnetData []byte
}

// subnetJob
type subnetJob struct {
	context    jobContext
	vnetId     string
	subnetId   string
	subnetData []byte
}

type workerPoolContext struct {
	numWorkers     int
	workerSem      chan struct{}
	failedJobQueue sync.Map
	jobTsLock      *jobTsLock
}

func (jobType jobType) String() string {
	switch jobType {

	case NetworkContainerPublish:
		return "NetworkContainerPublish"

	case NetworkContainerUnpublish:
		return "NetworkContainerUnpublish"

	case VnetRegister:
		return "VnetRegister"

	case VnetUnregister:
		return "VnetUnregister"

	case VnetUpdate:
		return "VnetUpdate"

	case BareMetalContainerPublish:
		return "BareMetalContainerPublish"

	case BareMetalContainerUnpublish:
		return "BareMetalContainerUnpublish"

	case SubnetRegister:
		return "SubnetRegister"

	case SubnetUnregister:
		return "SubnetUnregister"

	case SubnetUpdate:
		return "SubnetUpdate"

	}

	return "Unknown Type"
}

//TODO: replace with newWorkerPool
func workerPool() *workerPoolContext {
	once.Do(func() {
		wp = &workerPoolContext{
			numWorkers: maxConcurrentRequests,
			workerSem:  make(chan struct{}, maxConcurrentRequests),
			jobTsLock:  NewJobTsLock(),
		}

		for i := 0; i < maxConcurrentRequests; i++ {
			wp.workerSem <- struct{}{}
		}

		//TODO: Recover in case of go routines exit and restart.
		go wp.processNcPendingPublishRequests()
		go wp.processFailedJobs()

		logDebugf("WorkerPool successfully initialized")
	})

	return wp
}

func (wp *workerPoolContext) processNcPendingPublishRequests() {
	timerChannel := time.NewTicker(ncPendingRequestProcessInterval).C
	for {
		select {
		case <-timerChannel:
			if replicaInstance.IsReplicaPrimary() {
				processNcPendingRequests()
			}
		}
	}
}

func processNcPendingRequests() {
	pendingPublishRequests, err := ncPendingPublishRequestContextRepo.GetNcPendingPublishRequestContexts()
	if err != nil {
		logErrorf("could not get pending pubsub requests. error: %+v", err)
		return
	}

	if len(pendingPublishRequests) == 0 {
		return
	}

	for _, requestContext := range pendingPublishRequests {
		if !replicaInstance.IsReplicaPrimary() {
			return
		}

		if time.Now().UTC().Sub(requestContext.LastRetriedDateTime) > requestContext.RetryInterval {
			// Check the request type
			if !isValidPubSubJob(requestContext.Type) {
				logErrorf("Unknown type for NC pending publish context: %+v. Deleting the entry from the table.", requestContext)
				err := ncPendingPublishRequestContextRepo.RemoveNcPendingPublishContext(requestContext.NetworkID,
					requestContext.NetworkContainerID, requestContext.PubsubAzId)
				if err != nil {
					logErrorf("Failed to delete pending publish request %s for NC: %s. Error: %v",
						requestContext.Type.String(), requestContext.NetworkContainerID, err)
				}
				continue
			}

			jobId := createNcJobId(requestContext.NetworkID, requestContext.NetworkContainerID, requestContext.PubsubAzId)

			// move this out the loop
			var secondaryContext []pubsubContext

			// TO DO : Move this to a separate function
			var contextSelector, path string
			if (requestContext.Type == datamodel.BareMetalNetworkContainerPublishRequest) ||
				(requestContext.Type == datamodel.BareMetalNetworkContainerUnpublishRequest) {
				contextSelector = getAzContainerPublishingContextSelector(requestContext.PodName)
				path = getAzContainerPublishingPath()

				for _, networkInterface := range requestContext.NetworkInterfaces {
					var pubJobContext pubsubContext

					// Add config Path
					pubJobContext.Key = networkInterface.Key
					pubJobContext.Path = networkInterface.Path
					pubJobContext.Data = networkInterface.Data
					secondaryContext = append(secondaryContext, pubJobContext)
				}

			} else {
				contextSelector = generateVnetPartitionKey(requestContext.InfraNetworkID)
				path = generateVnetNcPath(requestContext.NodeInterfaceIP, requestContext.NetworkContainerID)
			}

			jobContext := jobContext{
				id:              jobId,
				pubsubAzId:      requestContext.PubsubAzId,
				partitionKey:    contextSelector,
				relativePath:    path,
				createdDateTime: requestContext.CreatedDateTime,
				retryInterval:   requestContext.RetryInterval,
				secondarys:      secondaryContext,
			}

			// No need to check error, as the type has been validated above
			pubsubJobType, _ := getJobType(requestContext.Type)

			var job Job
			jobContext.jobType = pubsubJobType
			job = &ncJob{
				context: jobContext,
				ncId:    requestContext.NetworkContainerID,
				vnetId:  requestContext.NetworkID,
				ncData:  requestContext.NcData,
			}

			workerPool().scheduleJob(job)
		}
	}
}

func (wp *workerPoolContext) isJobProcessingNeeded(jobContext *jobContext, tsl *tsLock) bool {
	latestTimestamp, isJobCompleted := tsl.ReadStatusTimestamp()
	// Do not process the job if -
	// 1. The timestamp of the job ID is prior to the latest known timestamp for that job ID.
	// 2. The job processing is already completed
	// This avoids processing stale updates and duplicate jobs
	if (jobContext.createdDateTime.Equal(latestTimestamp) && isJobCompleted) ||
		jobContext.createdDateTime.Before(latestTimestamp) {
		logDebugf("Skipping job: %s-%s. LatestTs: %v, JobTs: %v, isCompleted: %t",
			jobContext.jobType.String(), jobContext.id, latestTimestamp, jobContext.createdDateTime, isJobCompleted)
		return false
	}

	return true
}

func (wp *workerPoolContext) processJob(
	job Job) {
	var (
		isJobCompleted bool
		jobContext     = job.getContext()
	)

	tsl := wp.jobTsLock.GetTsLock(jobContext.id)
	tsl.Lock()
	defer func() {
		tsl.Unlock()
		wp.workerSem <- struct{}{}
	}()

	if !wp.isJobProcessingNeeded(jobContext, tsl) {
		return
	}

	if err := job.process(); err == nil {
		isJobCompleted = true
	} else {
		logErrorf("Failed to process the job: %s. Error: %v", jobContext.id, err)
	}

	tsl.UpdateStatusTimestamp(jobContext.createdDateTime, isJobCompleted)
}

func (wp *workerPoolContext) scheduleJob(
	job Job) {
	// Proceed if the worker (idle goroutine) is available to work on the incoming job
	<-wp.workerSem
	go wp.processJob(job)
}

func (wp *workerPoolContext) processFailedJob(
	key interface{},
	value interface{}) bool {
	jobId := key.(string)
	job := value.(Job)
	jobContext := job.getContext()

	tsl := wp.jobTsLock.GetTsLock(jobId)
	tsl.Lock()
	defer tsl.Unlock()

	if !wp.isJobProcessingNeeded(jobContext, tsl) {
		wp.failedJobQueue.Delete(jobId)
		return true
	}

	if time.Now().UTC().Sub(jobContext.lastRetriedDateTime) > jobContext.retryInterval {
		logDebugf("Picked up job %s: %s from failed job queue", jobContext.jobType.String(), jobContext.id)
		wp.failedJobQueue.Delete(jobId)
		wp.scheduleJob(job)
	}

	return true
}

func (wp *workerPoolContext) processFailedJobs() {
	ticker := time.NewTicker(
		time.Millisecond * time.Duration(1000)).C
	for {
		select {
		case <-ticker:
			wp.failedJobQueue.Range(wp.processFailedJob)
		}
	}
}

func (job *ncJob) process() error {
	var (
		err        error
		jobContext = job.getContext()
		path       = jobContext.partitionKey + jobContext.relativePath
	)

	logDebugf("Process Job-%s: %s", jobContext.jobType.String(), jobContext.id)

	if jobContext.jobType == NetworkContainerPublish {
		err = publishData(jobContext.pubsubAzId, jobContext.partitionKey, jobContext.relativePath, job.ncData)
	} else if jobContext.jobType == NetworkContainerUnpublish {
		err = unpublishData(jobContext.pubsubAzId, jobContext.partitionKey, jobContext.relativePath)
	} else if jobContext.jobType == BareMetalContainerPublish || jobContext.jobType == BareMetalContainerUnpublish {
		primary := pubsubContext{
			Key:  jobContext.partitionKey,
			Path: jobContext.relativePath,
			Data: job.ncData,
		}

		if jobContext.jobType == BareMetalContainerPublish {
			err = createContextSelectorAndPublishData(jobContext.pubsubAzId, primary, jobContext.secondarys, jobContext)
		} else {
			err = deleteContextSelectorAndUnPublishData(jobContext.pubsubAzId, primary, jobContext.secondarys, jobContext)
		}

		// set path to empty so we won't check for error below
		path = ""
	} else {
		err = fmt.Errorf("Unknown job type: %s for job: %s", jobContext.jobType, jobContext.id)
	}

	if err != nil {
		err = fmt.Errorf("%s: %s failed for NC: %s due to error: %v", jobContext.jobType.String(), jobContext.id, job.ncId, err)
	} else if path != "" {
		// This acts as an await for the async operation to finish.
		statusError := getPublishStatusError(jobContext.pubsubAzId, path)
		if statusError != nil {
			err = statusError
		}
	}

	ncPendingPublishContextKey := datamodel.GetNcPendingPublishContextKey(job.vnetId, job.ncId, jobContext.pubsubAzId)
	sessionInfo := map[string]string{datamodel.NCPendingPublishRepositoryName: ncPendingPublishContextKey}
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	pendingPublishContext, errDbOp := ncPendingPublishRequestContextRepo.QueryNcPendingPublishContext(ncPendingPublishContextKey)
	if errDbOp != nil || pendingPublishContext == nil {
		err = fmt.Errorf("%s: %s failed while querying NC pending publish context. "+
			"Context: %+v. Error: %v", jobContext.jobType.String(), jobContext.id, pendingPublishContext, errDbOp)
		// Return here since it won't be possible to update the table entry
		return err
	}

	if err != nil {
		logErrorf("Job %s: %s failed due to error: %v", jobContext.jobType.String(), jobContext.id, err)
		pendingPublishContext.LastRetriedDateTime = time.Now().UTC()
		updateRetryInterval(&pendingPublishContext.RetryInterval, maxRetryBackoffInterval)

		if errDbOp = ncPendingPublishRequestContextRepo.CreateOrUpdateNcPendingPublishRequest(pendingPublishContext); errDbOp != nil {
			logErrorf("Failed to persist pending %s request for NC: %s. Error: %v",
				jobContext.jobType.String(), pendingPublishContext.NetworkContainerID, errDbOp)
		}
	} else {
		logDebugf("Job %s: %s succeeded", jobContext.jobType.String(), jobContext.id)
		// Delete the pending context table entry only if the timestamp matches.
		// This avoids deletion of the updated entry
		if jobContext.createdDateTime.Equal(pendingPublishContext.CreatedDateTime) {
			errDbOp := ncPendingPublishRequestContextRepo.RemoveNcPendingPublishContext(job.vnetId, job.ncId, jobContext.pubsubAzId)
			if errDbOp != nil {
				err = fmt.Errorf("Failed to delete pending publish request %s for NC: %s. Error: %v",
					jobContext.jobType.String(), pendingPublishContext.NetworkContainerID, errDbOp)
			}
		}
	}

	return err
}

func (job *ncJob) getContext() *jobContext {
	return &job.context
}

// Update retry interval (by doubling it) unless it's greater than max back off interval
func updateRetryInterval(retryInterval *time.Duration, maxRetryBackoffInterval time.Duration) {
	if *retryInterval < maxRetryBackoffInterval {
		*retryInterval *= 2
	}
}

func (job *vnetJob) process() error {
	var (
		err        error
		jobContext = job.getContext()
	)

	logDebugf("Process Job %s: %s", jobContext.jobType.String(), jobContext.id)

	defer func() {
		if err != nil {
			logErrorf("Job %s: %s failed due to error: %v", jobContext.jobType.String(), jobContext.id, err)
			jobContext.lastRetriedDateTime = time.Now().UTC()
			updateRetryInterval(&jobContext.retryInterval, maxRetryBackoffInterval)
			workerPool().failedJobQueue.Store(jobContext.id, job)
		} else {
			logDebugf("Job %s: %s succeeded", jobContext.jobType.String(), jobContext.id)
		}
	}()

	// Do not process further if the vnet is deleted
	if getActiveSubscriber().getCachedVnetConfig(job.vnetId) == nil {
		logDebugf("Job %s: %s abandoned as vnet is deleted", jobContext.jobType.String(), jobContext.id)
		return err
	}

	switch jobContext.jobType {
	case VnetRegister:
		err = job.processVnetRegister()

	case VnetUnregister:
		err = job.processVnetUnregister()

	case VnetUpdate:
		err = job.processVnetUpdate()

	default:
		err = fmt.Errorf("Unknown job type: %s for job: %s", jobContext.jobType, jobContext.id)
	}

	return err
}

func (job *vnetJob) getContext() *jobContext {
	return &job.context
}

func (job *vnetJob) processVnetUpdate() error {
	var (
		err          error
		vnetConfig   C.struct_VnetConfiguration
		resultStatus C.struct_ResultStatus
	)

	vnetUpdateDataCBytes := C.CBytes(job.vnetData)
	resultStatus = C.DeserializeVnetConfiguration(vnetUpdateDataCBytes, C.size_t(len(job.vnetData)), &vnetConfig)
	if resultStatus.ResultCode != 0 {
		err = fmt.Errorf("Failed to deserialize vnet config for vnet: %s due to error: %s",
			job.vnetId, C.GoString(resultStatus.ErrorDetails))
		return err
	}

	// Update the cached vnet config
	var subnetConfigs []SubnetConfiguration
	subnetPtr := uintptr(unsafe.Pointer(vnetConfig.Subnets))
	for subnetCount := 0; subnetCount < (int)(vnetConfig.SubnetCount); subnetCount++ {
		var subnetConfigInfo SubnetConfiguration
		subnetConfig := (*C.struct_SubnetConfiguration)(unsafe.Pointer(subnetPtr))
		subnetConfigInfo.SubnetName = C.GoString(subnetConfig.SubnetName)
		subnetConfigInfo.AddressPrefix = C.GoString(subnetConfig.AddressPrefix)
		subnetConfigInfo.DefaultGateway = C.GoString(subnetConfig.DefaultGateway)
		subnetConfigInfo.ExpectedToken = C.GoString(subnetConfig.ExpectedToken)
		subnetConfigInfo.OwningService = C.GoString(subnetConfig.OwningService)

		tagPtr := uintptr(unsafe.Pointer(subnetConfig.Tags))
		for tagCount := 0; tagCount < (int)(subnetConfig.TagCount); tagCount++ {
			var tagConfigInfo TagConfiguration
			tag := (*C.struct_TagConfiguration)(unsafe.Pointer(tagPtr))
			tagConfigInfo.Name = C.GoString(tag.Name)
			tagConfigInfo.Type = C.GoString(tag.Type)

			subnetConfigInfo.Tags = append(subnetConfigInfo.Tags, tagConfigInfo)
			tagPtr += sizeofTagConfiguration()
		}

		subnetConfigs = append(subnetConfigs, subnetConfigInfo)
		subnetPtr += sizeofSubnetConfiguration()
	}

	vnetConfiguration := VnetConfiguration{
		VnetVersion:   C.GoString(vnetConfig.VnetVersion),
		VNetSpace:     C.GoString(vnetConfig.VNetSpace),
		CNetSpace:     C.GoString(vnetConfig.CNetSpace),
		DNSServers:    C.GoString(vnetConfig.DNSServers),
		SubnetConfigs: subnetConfigs,
	}

	getActiveSubscriber().updateCachedVnetConfig(job.vnetId, &vnetConfiguration)
	logDebugf("Successfully retrieved config for vnet: %s", job.vnetId)

	defer func() {
		// release C memory
		subnetPtr = uintptr(unsafe.Pointer(vnetConfig.Subnets))
		for subnetCount := 0; subnetCount < (int)(vnetConfig.SubnetCount); subnetCount++ {
			subnetConfig := (*C.struct_SubnetConfiguration)(unsafe.Pointer(subnetPtr))
			C.free((unsafe.Pointer)(subnetConfig.SubnetName))
			C.free((unsafe.Pointer)(subnetConfig.AddressPrefix))
			C.free((unsafe.Pointer)(subnetConfig.DefaultGateway))
			C.free((unsafe.Pointer)(subnetConfig.ExpectedToken))
			C.free((unsafe.Pointer)(subnetConfig.OwningService))

			tagPtr := uintptr(unsafe.Pointer(subnetConfig.Tags))
			for tagCount := 0; tagCount < (int)(subnetConfig.TagCount); tagCount++ {
				tag := (*C.struct_TagConfiguration)(unsafe.Pointer(tagPtr))
				C.free((unsafe.Pointer)(tag.Name))
				C.free((unsafe.Pointer)(tag.Type))
				tagPtr += sizeofTagConfiguration()
			}

			if subnetConfig.TagCount > 0 {
				C.free((unsafe.Pointer)(subnetConfig.Tags))
			}
			subnetPtr += sizeofSubnetConfiguration()
		}

		C.free((unsafe.Pointer)(vnetConfig.Subnets))
	}()

	// If vnet update succeeds, no further processing is needed.
	// Otherwise, return error which causes the workerpool to retry
	success, errUpdate := updateVnetConfiguration(job.vnetId, &vnetConfiguration)
	if !success || errUpdate != nil {
		logErrorf("Retriable error while processing vnetUpdateJob: %s. Error: %v", job.getContext().id, errUpdate)
		err = fmt.Errorf(fmt.Sprintf("vnetUpdateJob failed due to error: %v", errUpdate))
	}
	return err
}

func (job *vnetJob) processVnetRegister() error {
	var (
		err      error
		vnetPath = vnetPartitionKeyPrefix + job.vnetId + vnetConfigRelativePath
	)

	if err = getActiveSubscriber().subscribe(
		vnetPartitionKeyPrefix+job.vnetId, vnetConfigRelativePath); err != nil {
		return err
	}

	return getActiveSubscriber().getSubscribeStatusError(vnetPath)
}

func (job *vnetJob) processVnetUnregister() error {
	var err error
	getActiveSubscriber().deleteCachedVnetConfig(job.vnetId)

	if err = getActiveSubscriber().unsubscribe(
		vnetPartitionKeyPrefix+job.vnetId, vnetConfigRelativePath); err != nil {
		return err
	}

	return nil
}

func (job *subnetJob) process() error {
	var (
		err        error
		jobContext = job.getContext()
	)

	logDebugf("Process Job %s: %s", jobContext.jobType.String(), jobContext.id)

	defer func() {
		if err != nil {
			logErrorf("Job %s: %s failed due to error: %v", jobContext.jobType.String(), jobContext.id, err)
			jobContext.lastRetriedDateTime = time.Now().UTC()
			updateRetryInterval(&jobContext.retryInterval, maxRetryBackoffInterval)
			workerPool().failedJobQueue.Store(jobContext.id, job)
		} else {
			logDebugf("Job %s: %s succeeded", jobContext.jobType.String(), jobContext.id)
		}
	}()

	// If subnet is not in cache, it means it has been unsubscribed. Any pending tasks in queue should ignore it
	if _, getErr := getActiveSubscriber().getCachedSubnetConfig(job.vnetId, job.subnetId); getErr != nil {
		logDebugf("Job %s: %s abandoned as subnet is deleted", jobContext.jobType.String(), jobContext.id)
	    // return nil as we don't want to add this task to failedQueue in defer function above
		return nil
	}

	switch jobContext.jobType {
	case SubnetRegister:
		// Make sure to set err variable. This is used in defer function above
		err = job.processSubnetRegister()

	case SubnetUnregister:
		// Make sure to set err variable. This is used in defer function above
		err = job.processSubnetUnregister()

	case SubnetUpdate:
		// Make sure to set err variable. This is used in defer function above
		err = job.processSubnetUpdate()

	default:
		err = fmt.Errorf("Unknown job type: %s for job: %s", jobContext.jobType, jobContext.id)
	}

	return err
}

func (job *subnetJob) getContext() *jobContext {
	return &job.context
}

func (job *subnetJob) processSubnetRegister() error {
	var (
		vnetPath = getActiveSubscriber().getSubnetPartitionKey(job.vnetId, job.subnetId)
		subnetPath = getActiveSubscriber().getSubnetRelativePath(job.vnetId, job.subnetId)
		fullPath = fmt.Sprintf("%s%s", vnetPath, subnetPath)
	)

	if err := getActiveSubscriber().subscribe(vnetPath, subnetPath); err != nil {
		return err
	}

	return getActiveSubscriber().getSubscribeStatusError(fullPath)
}

func (job *subnetJob) processSubnetUnregister() error {
	var (
		vnetPath = getActiveSubscriber().getSubnetPartitionKey(job.vnetId, job.subnetId)
		subnetPath = getActiveSubscriber().getSubnetRelativePath(job.vnetId, job.subnetId)
	)

	getActiveSubscriber().deleteCachedSubnetConfig(job.vnetId, job.subnetId)

	if err := getActiveSubscriber().unsubscribe(vnetPath, subnetPath); err != nil {
		return err
	}

	return nil
}

func (job *subnetJob) processSubnetUpdate() error {
	// TODO: Implement
	return nil
}

func sizeofTagConfiguration() uintptr {
	return uintptr(C.SizeofTagConfiguration())
}

func sizeofSubnetConfiguration() uintptr {
	return uintptr(C.SizeofSubnetConfiguration())
}

func createAznmJobId(
	vnetId string,
	ncId string) string {
	return fmt.Sprintf("AZNM_%s_%s", vnetId, ncId)
}

func createNcJobId(
	vnetId string,
	ncId string,
	pubsubAzId string) string {
	return fmt.Sprintf("NC_%s_%s_%s", vnetId, ncId, pubsubAzId)
}

func createVnetJobId(
	vnetId string) string {
	return fmt.Sprintf("Vnet_%s", vnetId)
}

func createSubnetJobId(
	vnetId, subnetId string) string {
	return fmt.Sprintf("Vnet_%s_Subnet_%s", vnetId, subnetId)
}

func updateVnetConfiguration(
	vnetID string,
	vnetConfig *VnetConfiguration) (bool, *api.JsonErr) {
	logDebugf("UpdateVnetConfiguration for vnet: %s, version: %s", vnetID, vnetConfig.VnetVersion)

	networkResponse := nmagentcontracts.VirtualNetworkResponse{
		VnetVersion: vnetConfig.VnetVersion,
		CNetSpace:   vnetConfig.CNetSpace,
		DNSServers:  vnetConfig.DNSServers,
		VNetSpace:   vnetConfig.VNetSpace,
	}

	var subnets []nmagentcontracts.Subnet
	for _, s := range vnetConfig.SubnetConfigs {
		var subnet nmagentcontracts.Subnet
		subnet.SubnetName = s.SubnetName
		subnet.AddressPrefix = s.AddressPrefix
		subnet.DefaultGateway = s.DefaultGateway
		for _, t := range s.Tags {
			tag := nmagentcontracts.Tag{
				Name: t.Name,
				Type: t.Type,
			}
			subnet.Tags = append(subnet.Tags, tag)
		}
		subnets = append(subnets, subnet)
	}

	networkResponse.Subnets = subnets

	networkVersion, errParse := strconv.ParseUint(vnetConfig.VnetVersion, 10, 64)
	if errParse != nil {
		return false, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to parse network version: %s for network: %s due to error: %v", networkResponse.VnetVersion, vnetID, errParse)
	}

	repoNetworkConfiguration, err := networkRepo.TryQueryNetworkConfiguration(vnetID)
	if err != nil {
		return false, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to retrieve network config: %s due to error: %v", vnetID, err)
	}

	if repoNetworkConfiguration == nil {
		logDebugf("Configuration for Network %s not found.", vnetID)
		// Vnet update is not needed if the network is not there.
		return true, nil
	}

	if repoNetworkConfiguration.Version >= networkVersion && repoNetworkConfiguration.Status == datamodel.Completed {
		logDebugf("Vnet: %s version is the latest and processing is completed, no update necessary. "+
			"RepoVnetVersion: %d UpdateVnetVersion: %d", vnetID, repoNetworkConfiguration.Version, networkVersion)
		return true, nil
	}

	return networkupdatehandler.UpdateNetworkConfiguration(repoNetworkConfiguration, &networkResponse)
}

func createContextSelectorAndPublishData(
	pubsubAzId string, primary pubsubContext, pubsubCtx []pubsubContext, jobCtx *jobContext) error {

	var err error
	keys := make(map[string]struct{})

	// create a single slice where primary is the first element in this slice
	localPubsubCtx := []pubsubContext{primary}
	localPubsubCtx = append(localPubsubCtx, pubsubCtx...)

	for _, ctx := range localPubsubCtx {
		// Create context selector first. There could be same context selector in pubsubCtx array
		// so skip duplicates with a map
		if _, ok := keys[ctx.Key]; !ok {
			logDebugf("Creating context selector %s in AzId %s", ctx.Key, pubsubAzId)
			if err = createContextSelector(pubsubAzId, ctx.Key); err != nil {
				return err
			}

			err = getPublishStatusError(pubsubAzId, ctx.Key)
			if err != nil {
				return err
			}

			keys[ctx.Key] = struct{}{}
		} else {
			logDebugf("Context selector %s in AzId %s exists already", ctx.Key, pubsubAzId)
		}

		logDebugf("Publishing data. AzId %s, context selector %s, path %s", pubsubAzId, ctx.Key, ctx.Path)

		if err = publishData(pubsubAzId, ctx.Key, ctx.Path, ctx.Data); err != nil {
			logDebugf("Publishing data failed. AzId %s, context selector %s, path %s, err: %s",
				pubsubAzId, ctx.Key, ctx.Path, err)
			return err
		}

		path := ctx.Key + ctx.Path
		if err = getPublishStatusError(pubsubAzId, path); err != nil {
			logDebugf("Publishing data failed. AzId %s, context selector %s, path %s, publish err: %s",
				pubsubAzId, ctx.Key, ctx.Path, err)
			return err
		}
	}

	return nil
}

func deleteContextSelectorAndUnPublishData(
	pubsubAzId string, primary pubsubContext, pubsubCtx []pubsubContext, jobCtx *jobContext) error {

	// Remove secondary paths first
	if err := removePubsubPaths(pubsubAzId, pubsubCtx, jobCtx); err != nil {
		return err
	}

	// Remove primary
	primarySlice := []pubsubContext{primary}
	if err := removePubsubPaths(pubsubAzId, primarySlice, jobCtx); err != nil {
		return err
	}

	return nil
}

func removePubsubPaths(
	pubsubAzId string, pubsubCtx []pubsubContext, jobCtx *jobContext) error {

	var err error
	var exists bool
	keys := make(map[string]struct{})

	// unpublish the data first
	for _, ctx := range pubsubCtx {
		if exists, err = pathExists(pubsubAzId, ctx.Key, ctx.Path); err == nil && exists == false {
			logDebugf("Skipping unpublishing data for AzId %s since path %s%s does not exist", pubsubAzId, ctx.Key, ctx.Path)
			continue
		}

		logDebugf("unpublishing data. AzId %s, context selector %s, path %s", pubsubAzId, ctx.Key, ctx.Path)
		if err = unpublishData(pubsubAzId, ctx.Key, ctx.Path); err != nil {
			return err
		}

		path := ctx.Key + ctx.Path
		if err = getPublishStatusError(pubsubAzId, path); err != nil {
			return err
		}
	}

	// remove partition keys (or context selectors)
	for _, ctx := range pubsubCtx {

		if _, ok := keys[ctx.Key]; !ok {
			logDebugf("Removing context selector %s in AzId %s", ctx.Key, pubsubAzId)

			if err = removeContextSelector(pubsubAzId, ctx.Key); err != nil {
				return err
			}

			path := ctx.Key
			if err = getPublishStatusError(pubsubAzId, path); err != nil {
				return err
			}

			keys[ctx.Key] = struct{}{}
		} else {
			logDebugf("Context selector %s in AzId %s already removed", ctx.Key, pubsubAzId)
		}
	}

	return nil
}