package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-container-networking/log"

	"github.com/Azure/azure-sdk-for-go/storage"
)

type PendingResourceCleanupRepo struct {
	data   datastoreshim.DataStoreShim
	logger loggershim.LoggerInterface
}

// ResourceCleanupType - Type of the async cleanup request
type ResourceCleanupType int

const (
	// InvalidResourceCleanupType - Not Used
	InvalidResourceCleanupType = iota

	// IP
	IPCleanup
)

const (
	pendingResourceCleanupKey = "Type_%s_Network_%s_NC_%s"
)

func (c ResourceCleanupType) String() string {
	switch c {

	case IPCleanup:
		return "IPCleanup"

	case InvalidResourceCleanupType:
		return "InvalidResourceCleanupType"
	}

	return "Unknown Status"
}

// CleanupContext
type CleanupContext struct {
	Type                     ResourceCleanupType
	ToBeDeletedIpAllocations map[string][]string //Key:Network_Subnet: Value:List of Allocated IPs pending cleanup
	CreatedDateTime          time.Time
	UpdatedDateTime          time.Time
}

// String - ToString implementation of request context.
func (c *CleanupContext) String() string {
	return fmt.Sprintf("CleanupContext:[Type:%s, ToBeDeletedIpAllocations:%+v, CreatedDateTime:%s, UpdatedDateTime:%s]",
		c.Type,
		c.ToBeDeletedIpAllocations,
		c.CreatedDateTime.String(),
		c.UpdatedDateTime.String())
}

func GetPendingResourceCleanupKey(cleanupType ResourceCleanupType, NetworkId string, NCID string) string {
	return fmt.Sprintf(pendingResourceCleanupKey, cleanupType.String(), NetworkId, NCID)
}

func GetNetworkContainerIDFromRowKey(rowKey string) string {
	return strings.Split(rowKey, "_")[1]
}

type PendingResourceCleanupEntity struct {
	RowKey         string
	CleanupContext CleanupContext
	Etag           string
}

func NewPendingResourceCleanupRepo(dataStore *datastoreshim.DataStoreShim, logger loggershim.LoggerInterface) *PendingResourceCleanupRepo {
	return &PendingResourceCleanupRepo{
		data:   dataStoreShim,
		logger: logger,
	}
}

// TryGetPendingCleanupRequest - Query a network container by key.
func (p *PendingResourceCleanupRepo) TryGetPendingCleanupRequest(requestKey string) (*PendingResourceCleanupEntity, *api.JsonErr) {
	entity, status, err := p.data.Query(pendingResourceCleanupTableName, requestKey)

	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve PendingCleanupRequest for NC key:[%s] due to error:[%s]",
			requestKey,
			err.Error())
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if entity == nil {
		return nil, nil
	}

	return deserializePendingCleanupRequestEntity(entity)
}

// GetAllPendingCleanupRequests - Retrieve all request pending cleanup requests for a given partition
func (p *PendingResourceCleanupRepo) GetAllPendingResourceCleanups() (map[string]PendingResourceCleanupEntity, error) {
	var context interface{} = nil
	requestsToProcess := make(map[string]PendingResourceCleanupEntity)
	for {
		requests, hasMoreData, cx, status, err := p.data.QueryAllRows(pendingResourceCleanupTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return nil, nil
			}
			return nil, err
		}

		for _, entity := range requests {
			requestEntity, err := deserializePendingCleanupRequestEntity(entity)
			if err != nil {
				return nil, err
			}
			requestsToProcess[entity.RowKey] = *requestEntity
		}

		if !hasMoreData {
			break
		}
		context = cx
	}

	return requestsToProcess, nil
}

// Create or Merge the PendingResourceCleanup Job
func (p *PendingResourceCleanupRepo) CreateOrMergeIPResourceCleanupRequest(resConfig *ReservationConfiguration) *api.JsonErr {
	// construct IPcleanup from the NCConfiguration and Reservation request
	cleanupRowKey := GetPendingResourceCleanupKey(IPCleanup, resConfig.NetworkID, resConfig.AssociatedNCID)

	var ipCleanup = make(map[string][]string)
	ipAllocPartitionKey := GetSubnetKey(resConfig.NetworkID, resConfig.AllocationRequest.SubnetName)
	if resConfig.AllocatedSubnet != "" {
		ipAddressKey := EncodeIPAllocationKey(resConfig.AllocatedSubnet)
		ipCleanup[ipAllocPartitionKey] = append(ipCleanup[ipAllocPartitionKey], ipAddressKey)
	}

	for _, allocatedIp := range resConfig.SecondaryAllocations {
		if allocatedIp != "" {
			ipAddressKey := EncodeIPAllocationKey(allocatedIp)
			ipCleanup[ipAllocPartitionKey] = append(ipCleanup[ipAllocPartitionKey], ipAddressKey)
		}
	}

	if len(ipCleanup) == 0 {
		log.Debugf("CreateOrMergeIPResourceCleanupRequest: Skipping creating"+
			" PendingResourceCleanupJob as there are no ips associated to be cleaned up, cleanupRowKey %s, reservation %+v ",
			cleanupRowKey, resConfig)
		return nil
	}

	// Created the CleanupContext
	cleanupRequest := &CleanupContext{
		ToBeDeletedIpAllocations: ipCleanup,
		Type:                     IPCleanup}

	return p.CreateOrMergePendingCleanupRequest(cleanupRowKey, cleanupRequest)
}

// Create or Merge the PendingResourceCleanup Job from IpAllocations Entities
func (p *PendingResourceCleanupRepo) CreateOrMergeIPResourceCleanupRequestFromIPAllocations(networkId string, subnetName string, ncId string, ipAllocationsEntities []IPAllocationEntity) *api.JsonErr {
	cleanupRowKey := GetPendingResourceCleanupKey(IPCleanup, networkId, ncId)

	if len(ipAllocationsEntities) == 0 {
		log.Debugf("CreateOrMergeIPResourceCleanupRequestFromIPAllocations: Skipping creating"+
			" PendingResourceCleanupJob as ipAllocationsEntities is empty, cleanupRowKey %s, networkId %s, subnetName %s ",
			cleanupRowKey, networkId, subnetName)
		return nil
	}

	// construct IPcleanup from list of ipAllocationEntities
	var ipCleanup = make(map[string][]string)
	ipAllocPartitionKey := GetSubnetKey(networkId, subnetName)
	for _, ipAllocationEntity := range ipAllocationsEntities {
		ipAddressKey := ipAllocationEntity.RowKey
		ipCleanup[ipAllocPartitionKey] = append(ipCleanup[ipAllocPartitionKey], ipAddressKey)
	}

	// Created the CleanupContext
	cleanupRequest := &CleanupContext{
		ToBeDeletedIpAllocations: ipCleanup,
		Type:                     IPCleanup}

	return p.CreateOrMergePendingCleanupRequest(cleanupRowKey, cleanupRequest)
}

// Create Or Merge entry (if it matches the etag) else retry
func (p *PendingResourceCleanupRepo) CreateOrMergePendingCleanupRequest(
	requestKey string,
	cleanupContext *CleanupContext) *api.JsonErr {
	// Todo: Add a retry loop in case there is any race condition

	// Check if item exists, if yes then merge the CleanupRequestContext
	existingRequestEntity, err := p.TryGetPendingCleanupRequest(requestKey)

	if err != nil {
		return err
	}

	// if no entry exists then create one, note this function will fail if
	// entry gets created in the interim in which case caller has to retry
	if existingRequestEntity == nil {
		return p.createPendingCleanupRequest(requestKey, cleanupContext)
	}

	// Merge the cleanupRequest and update the entity using the same etag
	// TODO: For now, Data is only IPCleanupMap but this can change later
	for allocationKey, ips := range cleanupContext.ToBeDeletedIpAllocations {
		if existingIps, found := existingRequestEntity.CleanupContext.ToBeDeletedIpAllocations[allocationKey]; found {
			for _, newIp := range ips {
				var skipAppend bool
				for _, existingIp := range existingIps {
					if existingIp == newIp {
						skipAppend = true
						break
					}
				}

				if !skipAppend {
					existingIps = append(existingIps, newIp)
				}
			}

			existingRequestEntity.CleanupContext.ToBeDeletedIpAllocations[allocationKey] = existingIps
		} else {
			existingRequestEntity.CleanupContext.ToBeDeletedIpAllocations[allocationKey] = ips
		}
	}

	return p.updatePendingCleanupRequest(requestKey, existingRequestEntity)

}

// Deletes the entity if etag matches
func (p *PendingResourceCleanupRepo) DeleteCleanupRequest(
	requestKey string,
	etag string) *api.JsonErr {
	err := p.data.DeleteWithEtag(pendingResourceCleanupTableName, requestKey, etag)

	if err != nil {
		msg := fmt.Sprintf(
			"Failed to delete request associated with key %s with etag %s in table %s due to error:{%s}",
			requestKey,
			etag,
			pendingResourceCleanupTableName,
			err.Error())
		p.logger.Errorf("%s: %s", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

// Create entry (if entry exists, it will return failure)
// For updates, caller is expected to call UpdatePendingCleanupRequest with etag
func (p *PendingResourceCleanupRepo) createPendingCleanupRequest(
	requestKey string,
	cleanupContext *CleanupContext) *api.JsonErr {
	cleanupContext.CreatedDateTime = time.Now().UTC()
	jsonSerializedBytes, err := json.Marshal(cleanupContext)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to serialize cleanup request context. Key:%s RequestContext:%s , Error: %s",
			requestKey,
			cleanupContext.String(),
			err.Error())
		p.logger.Errorf("%s: %s", utils.TracePrefix(2), msg)

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	_, err = p.data.Insert(pendingResourceCleanupTableName, requestKey, TableToColumnMap[pendingResourceCleanupTableName], string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to add request %s associated with key %s to pending cleanup table due to error:{%s}",
			cleanupContext.String(),
			requestKey,
			err.Error())
		p.logger.Errorf("%s: %s", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

// Updates entry if it matches the etag
// For updates, caller is expected to call UpdatePendingCleanupRequest with etag
func (p *PendingResourceCleanupRepo) updatePendingCleanupRequest(
	requestKey string,
	pendingCleanupRequestEntity *PendingResourceCleanupEntity) *api.JsonErr {
	pendingCleanupRequestEntity.CleanupContext.UpdatedDateTime = time.Now().UTC()
	jsonSerializedBytes, err := json.Marshal(pendingCleanupRequestEntity.CleanupContext)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to serialize cleanup request context. Key:%s RequestContext:%s , Error: %s",
			requestKey,
			pendingCleanupRequestEntity.CleanupContext.String(),
			err.Error())
		p.logger.Errorf("%s: %s", utils.TracePrefix(2), msg)

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	err = p.data.UpdateWithEtag(pendingResourceCleanupTableName, requestKey, pendingCleanupRequestEntity.Etag, TableToColumnMap[pendingResourceCleanupTableName], string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to add request %+v associated with key %s to pending cleanup table due to error:{%s}",
			pendingCleanupRequestEntity,
			requestKey,
			err.Error())
		p.logger.Errorf("%s: %s", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

func deserializePendingCleanupRequestEntity(entity *storage.Entity) (*PendingResourceCleanupEntity, *api.JsonErr) {
	prop, ok := entity.Properties[pendingResourceCleanupTag]
	if !ok {
		msg := fmt.Sprintf(
			"CleanupRequestContext Property for NC key:[%s] is not set",
			entity.RowKey)
		return nil, &api.JsonErr{
			Code:       api.PropertyNotSetInDb,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	serializedPendingCleanupRequest := prop.(string)
	var cleanupContext CleanupContext
	err := json.Unmarshal([]byte(serializedPendingCleanupRequest), &cleanupContext)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to unmarshal PendingCleanupRequest for NC key:[%s] due to error:[%s]",
			entity.RowKey,
			err)
		return nil, &api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return &PendingResourceCleanupEntity{
		RowKey:         entity.RowKey,
		CleanupContext: cleanupContext,
		Etag:           entity.OdataEtag}, nil
}
