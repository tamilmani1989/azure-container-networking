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
	"dnc/logger/loggershim"
	"dnc/nmagentcontracts"
	"dnc/resourceupdatehandler/networkupdatehandler"
	"fmt"
	"net/http"
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
	InvalidJobType // do not change the position
)

// TO DO: Consider make it as Context Selector, and
// an array of relative path + data
// this it is easy to process it as a batch, check/create context selector first,
// and then publish data. for Unpublish, process this as unit in reverse
// unpublish first, and then remove context selector.
type PublishJobContext struct {
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
	secondarys          []PublishJobContext
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
	}

	return "Unknown Type"
}

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

		go wp.processNcPendingPublishRequests()
		go wp.processFailedJobs()
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
	pendingPublishRequests, _ := datamodel.GetNcPendingPublishRequestContexts()
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
				loggershim.LS.Errorf("[PubSubProxy] Unknown type for NC pending publish context: %+v. "+
					"Deleting the entry from the table.", requestContext)
				datamodel.RemoveNcPendingPublishContext(requestContext.NetworkID,
					requestContext.NetworkContainerID, requestContext.PubsubAzId)
				continue
			}

			jobId := createNcJobId(requestContext.NetworkID,
				requestContext.NetworkContainerID, requestContext.PubsubAzId)

			// move this out the loop
			var secondaryContext []PublishJobContext

			// TO DO : Move this to a separate function
			var contextSelector, path string
			if (requestContext.Type == datamodel.BareMetalNetworkContainerPublishRequest) ||
				(requestContext.Type == datamodel.BareMetalNetworkContainerUnpublishRequest) {
				contextSelector = getAzContainerPublishingContextSelector(requestContext.PodName)
				path = getAzContainerPublishingPath()

				for _, networkInterface := range requestContext.NetworkInterfaces {
					var pubJobContext PublishJobContext

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

			loggershim.LS.Errorf("[PubSubProxy] Setting the context selector: Path : %s.",
				contextSelector,
				path)
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

func (wp *workerPoolContext) isJobProcessingNeeded(
	jobContext *jobContext) bool {
	latestTimestamp, isJobCompleted := wp.jobTsLock.ReadStatusTimestamp(jobContext.id)
	// Do not process the job if -
	// 1. The timestamp of the job ID is prior to the latest known timestamp for that job ID.
	// 2. The job processing is already completed
	// This avoids processing stale updates and duplicate jobs
	if (jobContext.createdDateTime.Equal(latestTimestamp) && isJobCompleted) ||
		jobContext.createdDateTime.Before(latestTimestamp) {
		loggershim.LS.Debugf("[PubSubProxy] Skipping job: %s-%s. LatestTs: %v, JobTs: %v, isCompleted: %t",
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

	wp.jobTsLock.LockAcquire(jobContext.id)
	defer func() {
		wp.jobTsLock.LockRelease(jobContext.id)
		wp.workerSem <- struct{}{}
	}()

	if !wp.isJobProcessingNeeded(jobContext) {
		return
	}

	if err := job.process(); err == nil {
		isJobCompleted = true
	}

	wp.jobTsLock.UpdateStatusTimestamp(jobContext.id, jobContext.createdDateTime, isJobCompleted)
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

	wp.jobTsLock.LockAcquire(jobId)
	defer wp.jobTsLock.LockRelease(jobId)

	if !wp.isJobProcessingNeeded(jobContext) {
		wp.failedJobQueue.Delete(jobId)
		return true
	}

	if time.Now().UTC().Sub(jobContext.lastRetriedDateTime) > jobContext.retryInterval {
		loggershim.LS.Debugf("[PubSubProxy] Picked up job %s: %s from failed job queue",
			jobContext.jobType.String(), jobContext.id)
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

	loggershim.LS.Debugf("[PubSubProxy] Process Job-%s: %s", jobContext.jobType.String(), jobContext.id)

	if jobContext.jobType == NetworkContainerPublish {
		err = publishData(jobContext.pubsubAzId, jobContext.partitionKey, jobContext.relativePath, job.ncData)
	} else if jobContext.jobType == NetworkContainerUnpublish {
		err = unpublishData(jobContext.pubsubAzId, jobContext.partitionKey, jobContext.relativePath)
	} else if jobContext.jobType == BareMetalContainerPublish {

		loggershim.LS.Debugf("[PubSubProxy] JobId: %s. Creating context selector: %s",
			jobContext.id, jobContext.partitionKey)

		if err = createContextSelector(jobContext.pubsubAzId, jobContext.partitionKey); err == nil {
			// wait for the context selector to complete
			statusError := getPublishStatusError(jobContext.pubsubAzId, jobContext.partitionKey)
			if statusError != nil {
				err = statusError
			} else {
				loggershim.LS.Debugf("[PubSubProxy] JobId: %s. Publishing to context selector: %s, path: %s",
					jobContext.id, jobContext.partitionKey, jobContext.relativePath)

				err = publishData(jobContext.pubsubAzId, jobContext.partitionKey, jobContext.relativePath, job.ncData)
			}

			statusError = getPublishStatusError(jobContext.pubsubAzId, path)
			if statusError != nil {
				err = statusError
			} else {
				keys := make(map[string]struct{})
				// reset path
				path = ""

				for _, secondaryJob := range jobContext.secondarys {
					// wait for publish job from the previous loop to finish (if paplicable)
					if path != "" {
						statusError = getPublishStatusError(jobContext.pubsubAzId, path)
						if statusError != nil {
							err = statusError
							break
						}
					}

					if _, ok := keys[secondaryJob.Key]; !ok {

						loggershim.LS.Debugf("[PubSubProxy] JobId: %s. Creating context selector: %s",
							jobContext.id, secondaryJob.Key)

						if err = createContextSelector(jobContext.pubsubAzId, secondaryJob.Key); err == nil {
							// wait for the context selector to complete
							statusError = getPublishStatusError(jobContext.pubsubAzId, secondaryJob.Key)
							if statusError != nil {
								err = statusError
								break
							}

							// put it in the context selector map
							keys[secondaryJob.Key] = struct{}{}
						} else {
							break
						}
					}

					loggershim.LS.Debugf("[PubSubProxy] JobId: %s. Publishing to context selector: %s, path: %s",
						jobContext.id, secondaryJob.Key, secondaryJob.Path)

					err = publishData(jobContext.pubsubAzId, secondaryJob.Key, secondaryJob.Path, secondaryJob.Data)
					if err != nil {
						break
					}

					path = secondaryJob.Key + secondaryJob.Path
				}
			}
		}
	} else if jobContext.jobType == BareMetalContainerUnpublish {

		keys := make(map[string]struct{})
		// reset path
		path = ""

		for _, secondaryJob := range jobContext.secondarys {
			// wait for publish job from the previous loop to finish (if paplicable)
			if path != "" {
				statusError := getPublishStatusError(jobContext.pubsubAzId, path)
				if statusError != nil {
					err = statusError
					break
				}
			}

			loggershim.LS.Debugf("[PubSubProxy] JobId: %s. Unpublishing to context selector: %s, path: %s",
				jobContext.id, secondaryJob.Key, secondaryJob.Path)

			err = unpublishData(jobContext.pubsubAzId, secondaryJob.Key, secondaryJob.Path)
			if err != nil {
				break
			}

			keys[secondaryJob.Key] = struct{}{}
			path = secondaryJob.Key + secondaryJob.Path
		}

		for partitionKey, _ := range keys {
			// wait for publish job from the previous loop to finish (if paplicable)
			if path != "" {
				statusError := getPublishStatusError(jobContext.pubsubAzId, path)
				if statusError != nil {
					err = statusError
					break
				}
			}

			loggershim.LS.Debugf("[PubSubProxy] JobId: %s. Removing context selector: %s",
				jobContext.id, partitionKey)

			err = removeContextSelector(jobContext.pubsubAzId, partitionKey)
			if err != nil {
				break
			}

			path = partitionKey
		}

		statusError := getPublishStatusError(jobContext.pubsubAzId, path)
		if statusError != nil {
			err = statusError
		} else {
			// remove relative path first.
			loggershim.LS.Debugf("[PubSubProxy] JobId: %s. Unpublishing to context selector: %s, path: %s",
				jobContext.id, jobContext.partitionKey, jobContext.relativePath)

			if err = unpublishData(jobContext.pubsubAzId, jobContext.partitionKey, jobContext.relativePath); err == nil {
				// wait for the context selector to complete
				statusError := getPublishStatusError(jobContext.pubsubAzId, jobContext.partitionKey+jobContext.relativePath)
				if statusError != nil {
					err = statusError
				} else {
					// Need to reset the path, because by default we will look for partition key+ relative path
					// but here there is no relative path, as we are
					loggershim.LS.Debugf("[PubSubProxy] JobId: %s. Removing context selector: %s",
						jobContext.id, jobContext.partitionKey)

					path = jobContext.partitionKey
					err = removeContextSelector(jobContext.pubsubAzId, jobContext.partitionKey)
				}
			}
		}
	} else {
		err = fmt.Errorf("Unknown job type: %s for job: %s", jobContext.jobType, jobContext.id)
	}

	if err != nil {
		err = fmt.Errorf("%s: %s failed for NC: %s due to error: %v",
			jobContext.jobType.String(), jobContext.id, job.ncId, err)
	} else {
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

	pendingPublishContext, errDbOp := datamodel.QueryNcPendingPublishContext(ncPendingPublishContextKey)
	if errDbOp != nil || pendingPublishContext == nil {
		err = fmt.Errorf("%s: %s failed while querying NC pending publish context. "+
			"Context: %+v. Error: %v", jobContext.jobType.String(), jobContext.id, pendingPublishContext, errDbOp)
		loggershim.LS.Errorf("[PubSubProxy] %v", err)
		// Return here since it won't be possible to update the table entry
		return err
	}

	if err != nil {
		loggershim.LS.Errorf("[PubSubProxy] %s: %s failed due to error: %v", jobContext.jobType.String(), jobContext.id, err)
		pendingPublishContext.LastRetriedDateTime = time.Now().UTC()
		if pendingPublishContext.RetryInterval < maxRetryBackoffInterval {
			pendingPublishContext.RetryInterval *= 2
		}

		if errDbOp = datamodel.CreateOrUpdateNcPendingPublishRequest(pendingPublishContext); errDbOp != nil {
			loggershim.LS.Errorf("Failed to persist pending %s request for NC: %s. Error: %v",
				jobContext.jobType.String(), pendingPublishContext.NetworkContainerID, errDbOp)
		}
	} else {
		loggershim.LS.Debugf("[PubSubProxy] %s: %s succeeded", jobContext.jobType.String(), jobContext.id)
		// Delete the pending context table entry only if the timestamp matches.
		// This avoids deletion of the updated entry
		if jobContext.createdDateTime.Equal(pendingPublishContext.CreatedDateTime) {
			datamodel.RemoveNcPendingPublishContext(job.vnetId, job.ncId, jobContext.pubsubAzId)
		}
	}

	return err
}

func (job *ncJob) getContext() *jobContext {
	return &job.context
}

func (job *vnetJob) process() error {
	var (
		err        error
		jobContext = job.getContext()
	)

	loggershim.LS.Debugf("[PubSubProxy] Process Job-%s: %s", jobContext.jobType.String(), jobContext.id)

	defer func() {
		if err != nil {
			loggershim.LS.Errorf("[PubSubProxy] %s: %s failed due to error: %v",
				jobContext.jobType.String(), jobContext.id, err)
			jobContext.lastRetriedDateTime = time.Now().UTC()
			if jobContext.retryInterval < maxRetryBackoffInterval {
				jobContext.retryInterval *= 2
			}

			workerPool().failedJobQueue.Store(jobContext.id, job)
		} else {
			loggershim.LS.Debugf("[PubSubProxy] %s: %s succeeded", jobContext.jobType.String(), jobContext.id)
		}
	}()

	// Do not process further if the vnet is deleted
	if getActiveSubscriber().getCachedVnetConfig(job.vnetId) == nil {
		loggershim.LS.Debugf("[PubSubProxy] %s: %s abandoned as vnet is deleted", jobContext.jobType.String(), job.vnetId)
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
		jobContext   = job.getContext()
	)

	vnetUpdateDataCBytes := C.CBytes(job.vnetData)
	resultStatus = C.DeserializeVnetConfiguration(vnetUpdateDataCBytes, C.size_t(len(job.vnetData)), &vnetConfig)
	if resultStatus.ResultCode != 0 {
		err = fmt.Errorf("Failed to deserialize vnet config for vnet: %s due to error: %s",
			job.vnetId, C.GoString(resultStatus.ErrorDetails))
		loggershim.LS.Debugf("[PubSubProxy] %v", err)
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
	loggershim.LS.Debugf("[PubSubProxy] Successfully retrieved config for vnet: %s", job.vnetId)

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
		loggershim.LS.Errorf("[PubSubProxy] Retriable error while processing vnetUpdateJob: %s. Error: %v",
			jobContext.id, errUpdate)
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

	if err = getActiveSubscriber().subscribe(
		vnetPartitionKeyPrefix+job.vnetId, subnetConfigRelativePath); err != nil {
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

	if err = getActiveSubscriber().unsubscribe(
		vnetPartitionKeyPrefix+job.vnetId, subnetConfigRelativePath); err != nil {
		return err
	}

	return nil
}

func sizeofTagConfiguration() uintptr {
	return uintptr(C.SizeofTagConfiguration())
}

func sizeofSubnetConfiguration() uintptr {
	return uintptr(C.SizeofSubnetConfiguration())
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

func updateVnetConfiguration(
	vnetID string,
	vnetConfig *VnetConfiguration) (bool, *api.JsonErr) {
	loggershim.LS.Debugf("[PubSubProxy] UpdateVnetConfiguration for vnet: %s, version: %s",
		vnetID, vnetConfig.VnetVersion)

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

	networkVersion, errParsing := strconv.ParseUint(vnetConfig.VnetVersion, 10, 64)
	if errParsing != nil {
		errMsg := fmt.Sprintf("Failed to parse network version %s for network %s due to error %s",
			networkResponse.VnetVersion,
			vnetID,
			errParsing.Error())
		loggershim.LS.Errorf("[PubSubProxy] %s", errMsg)
		return false, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       errMsg,
			HTTPStatus: http.StatusInternalServerError}
	}

	networkKey := datamodel.GetNetworkKey(vnetID)
	repoNetworkConfiguration, err := datamodel.TryQueryNetworkConfigurationByKey(networkKey)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to retrieve network config: %s due to error: %v", vnetID, err)
		loggershim.LS.Errorf("[PubSubProxy] %s", errMsg)
		return false, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       errMsg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if repoNetworkConfiguration == nil {
		loggershim.LS.Debugf("[PubSubProxy] Configuration for Network %s not found.", vnetID)
		// Vnet update is not needed if the network is not there.
		return true, nil
	}

	if repoNetworkConfiguration.Version >= networkVersion && repoNetworkConfiguration.Status == datamodel.Completed {
		loggershim.LS.Debugf("[PubSubProxy] Vnet: %s version is the latest and processing is completed, no update necessary. "+
			"RepoVnetVersion: %d UpdateVnetVersion: %d", vnetID, repoNetworkConfiguration.Version, networkVersion)
		return true, nil
	}

	return networkupdatehandler.UpdateNetworkConfiguration(repoNetworkConfiguration, &networkResponse)
}
