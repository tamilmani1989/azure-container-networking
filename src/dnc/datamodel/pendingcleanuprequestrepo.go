package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/storage"
	"net/http"
	"time"
)

type PendingCleanupRepo struct {
	data   datastoreshim.DataStoreShim
	logger loggershim.LoggerInterface
}

// CleanupRequestType - Type of the async cleanup request
type CleanupRequestType int

const (
	// InvalidCleanupRequest - Not Used
	InvalidCleanupRequest = iota

	// IPCleanupRequest
	IPCleanupRequest
)

func (c CleanupRequestType) String() string {
	switch c {

	case IPCleanupRequest:
		return "IPCleanupRequest"

	case InvalidCleanupRequest:
		return "InvalidCleanupRequest"
	}

	return "Unknown Status"
}

// PendingCleanupRequest - context
type CleanupRequestContext struct {
	IPCleanupMap    map[string][]string //Key:Network_Subnet: Value:List of Allocated IPs pending cleanup
	Type            CleanupRequestType
	CreatedDateTime time.Time
	UpdatedDateTime time.Time
}

// String - ToString implementation of request context.
func (c *CleanupRequestContext) String() string {
	return fmt.Sprintf("CleanupRequestContext:[IPCleanupMap:%+v, Type:%s, CreatedDateTime:%s, UpdatedDateTime:%s]",
		c.IPCleanupMap,
		c.Type,
		c.CreatedDateTime.String(),
		c.UpdatedDateTime.String())
}

type PendingCleanupRequestEntity struct {
	RowKey                string
	CleanupRequestContext CleanupRequestContext
	Etag                  string
}

// TryGetPendingCleanupRequest - Query a network container by key.
func (p *PendingCleanupRepo) TryGetPendingCleanupRequest(requestKey string) (*PendingCleanupRequestEntity, *api.JsonErr) {
	entity, status, err := p.data.Query(pendingCleanupRequestTableName, requestKey)

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
func (p *PendingCleanupRepo) GetAllPendingCleanupRequests() (map[string]PendingCleanupRequestEntity, error) {
	var context interface{} = nil
	requestsToProcess := make(map[string]PendingCleanupRequestEntity)
	for {
		requests, hasMoreData, cx, status, err := p.data.QueryAllRows(pendingCleanupRequestTableName, context)
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

// Create Or Merge entry (if it matches the etag) else retry
func (p *PendingCleanupRepo) CreateOrMergePendingCleanupRequest(
	requestKey string,
	cleanupRequestContext *CleanupRequestContext) *api.JsonErr {
	// Todo: Add a retry loop in case there is any race condition

	// Check if item exists, if yes then merge the CleanupRequestContext
	existingRequestEntity, err := p.TryGetPendingCleanupRequest(requestKey)

	if err != nil {
		return err
	}

	// if no entry exists then create one, note this function will fail if
	// entry gets created in the interim in which case caller has to retry
	if existingRequestEntity == nil {
		return p.createPendingCleanupRequest(requestKey, cleanupRequestContext)
	}

	// Merge the cleanupRequest and update the entity using the same etag
	for allocationKey, ips := range cleanupRequestContext.IPCleanupMap {
		if existingIps, found := existingRequestEntity.CleanupRequestContext.IPCleanupMap[allocationKey]; found {
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

			existingRequestEntity.CleanupRequestContext.IPCleanupMap[allocationKey] = existingIps
		} else {
			existingRequestEntity.CleanupRequestContext.IPCleanupMap[allocationKey] = ips
		}
	}

	return p.updatePendingCleanupRequest(requestKey, existingRequestEntity)

}

// Deletes the entity if etag matches
func (p *PendingCleanupRepo) DeleteCleanupRequest(
	requestKey string,
	etag string) *api.JsonErr {
	err := p.data.DeleteWithEtag(pendingCleanupRequestTableName, requestKey, etag)

	if err != nil {
		msg := fmt.Sprintf(
			"Failed to delete request associated with key %s with etag %s in table %s due to error:{%s}",
			requestKey,
			etag,
			pendingCleanupRequestTableName,
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
func (p *PendingCleanupRepo) createPendingCleanupRequest(
	requestKey string,
	cleanupRequest *CleanupRequestContext) *api.JsonErr {
	cleanupRequest.CreatedDateTime = time.Now().UTC()
	jsonSerializedBytes, err := json.Marshal(cleanupRequest)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to serialize cleanup request context. Key:%s RequestContext:%s , Error: %s",
			requestKey,
			cleanupRequest.String(),
			err.Error())
		p.logger.Errorf("%s: %s", utils.TracePrefix(2), msg)

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	_, err = p.data.Insert(pendingCleanupRequestTableName, requestKey, TableToColumnMap[pendingCleanupRequestTableName], string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to add request %s associated with key %s to pending cleanup table due to error:{%s}",
			cleanupRequest.String(),
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
func (p *PendingCleanupRepo) updatePendingCleanupRequest(
	requestKey string,
	pendingCleanupRequestEntity *PendingCleanupRequestEntity) *api.JsonErr {
	pendingCleanupRequestEntity.CleanupRequestContext.UpdatedDateTime = time.Now().UTC()
	jsonSerializedBytes, err := json.Marshal(pendingCleanupRequestEntity.CleanupRequestContext)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to serialize cleanup request context. Key:%s RequestContext:%s , Error: %s",
			requestKey,
			pendingCleanupRequestEntity.CleanupRequestContext.String(),
			err.Error())
		p.logger.Errorf("%s: %s", utils.TracePrefix(2), msg)

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	err = p.data.UpdateWithEtag(pendingCleanupRequestTableName, requestKey, pendingCleanupRequestEntity.Etag, TableToColumnMap[pendingCleanupRequestTableName], string(jsonSerializedBytes[:]))
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

func deserializePendingCleanupRequestEntity(entity *storage.Entity) (*PendingCleanupRequestEntity, *api.JsonErr) {
	prop, ok := entity.Properties[pendingCleanupRequestTag]
	if !ok { msg := fmt.Sprintf(
		"CleanupRequestContext Property for NC key:[%s] is not set",
		entity.RowKey)
		return nil, &api.JsonErr{
			Code:       api.PropertyNotSetInDb,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError} }

	serializedPendingCleanupRequest := prop.(string)
	var cleanupRequest CleanupRequestContext
	err := json.Unmarshal([]byte(serializedPendingCleanupRequest), &cleanupRequest)
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

	return &PendingCleanupRequestEntity{
		RowKey:                entity.RowKey,
		CleanupRequestContext: cleanupRequest,
		Etag:                  entity.OdataEtag}, nil
}
