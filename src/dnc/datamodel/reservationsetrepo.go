package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/storage"
	"net/http"
	"strings"
)

const (
	reservationSetTableName      = "IPReservationSetTable"
	reservationSetKey            = "Network_%s_ReservationSet_%s"
	ReservationSetRepositoryName = "ReservationSetRepository"

	// Properties in the IPReservationSetTable
	reservationSetIDProperty           = "ReservationSetID"
	networkIDProperty                  = "NetworkID"
	subnetNameProperty                 = "SubnetName"
	stateProperty                      = "State"
	statusProperty                     = "Status"
	successfulReservationCountProperty = "SuccessfulReservationCount"
	failureErrorCodeProperty           = "FailureErrorCode"
	failureErrorMsgProperty            = "FailureErrorMsg"
	failureHttpCodeProperty            = "FailureHttpCode"
	reservationSetRequestProperty      = "ReservationSetRequest"
	markedForDeleteProperty            = "MarkedForDelete"
)

type ReservationSetRepo struct {
	dncPartitionKey string
	dataStore       datastoreshim.DataStoreShim
	logger          loggershim.LoggerInterface
}

type ReservationSet struct {
	Key                        string
	ETag                       string
	ReservationSetID           string
	NetworkID                  string
	SubnetName                 string
	State                      ReservationSetState
	Status                     Status
	SuccessfulReservationCount int
	FailureDetail              api.JsonErr
	ReservationSetRequest      api.ReservationSetRequest
	MarkedForDelete            bool
}

func (r *ReservationSet) String() string {
	return fmt.Sprintf(
		"Key: %s ETag: %s ReservationSetID: %s NetworkID: %s SubnetName: %s "+
			"State: %s Status: %s SuccessfulReservationCount: %d FailureDetail: %s ReservationSetRequest: %+v MarkedForDelete: %t",
		r.Key, r.ETag, r.ReservationSetID, r.NetworkID, r.SubnetName, r.State.String(), r.Status.String(),
		r.SuccessfulReservationCount, r.FailureDetail.String(), r.ReservationSetRequest, r.MarkedForDelete)
}

// GetReservationSetKey - Helper function to generate reservation set key.
func GetReservationSetKey(networkID, reservationSetID string) string {
	return fmt.Sprintf(reservationSetKey, networkID, reservationSetID)
}

// parseRservationSetKey parses the reservation set key to return array of string values
func parseRservationSetKey(key string) ([]string, error) {
	// reservationSetKey = Network_%s_ReservationSet_%s
	// Splitting on "_" gives [1] as NetworkID and [3] as ReservationSetID
	values := strings.Split(key, "_")
	if len(values) != 4 {
		return nil, fmt.Errorf("could not extract reservation ID from key: %s", key)
	}

	if values[0] != "Network" ||  values[2] != "ReservationSet" {
		return nil, fmt.Errorf("unexpected reservation key format: %s", key)
	}

	return values, nil
}

// GetSetIDFromReservationSetKey - Helper function to get reservation set ID from key.
func GetSetIDFromReservationSetKey(key string) (string, error) {
	values, err := parseRservationSetKey(key)
	if err != nil {
		return "", err
	}

	// reservationSetKey = Network_%s_ReservationSet_%s
	// Splitting on "_" gives [1] as NetworkID and [3] as ReservationSetID
	return values[3], nil
}

// GetNetworkIDFromKey - Helper function to get reservation set ID from key.
func GetNetworkIDFromReservationSetKey(key string) (string, error) {
	values, err := parseRservationSetKey(key)
	if err != nil {
		return "", err
	}

	// reservationSetKey = Network_%s_ReservationSet_%s
	// Splitting on "_" gives [1] as NetworkID and [3] as ReservationSetID
	return values[1], nil
}

func NewReservationSetRepo(
	dataStore datastoreshim.DataStoreShim,
	dncPartitionKey string,
	logger loggershim.LoggerInterface) *ReservationSetRepo {
	return &ReservationSetRepo{
		dncPartitionKey: dncPartitionKey,
		dataStore:       dataStore,
		logger:          logger,
	}
}

func (r *ReservationSetRepo) TryQueryIPReservationSet(
	networkID string,
	reservationSetID string) (*ReservationSet, *api.JsonErr) {
	reservationSetKey := GetReservationSetKey(networkID, reservationSetID)
	entity, status, err := r.dataStore.Query(reservationSetTableName, reservationSetKey)
	if err != nil && status != http.StatusNotFound {
		return nil, api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve reservation set: %s. error: %v", reservationSetKey, err)
	}

	if entity == nil {
		return nil, nil
	}

	reservationSet, errJson := storageEntityToReservationSet(entity)
	if errJson != nil {
		r.logger.Errorf("could not convert the storage entity to reservation set. storage entity: %+v error: %s", entity, err)
		return nil, errJson
	}

	return &reservationSet, nil
}

func storageEntityToReservationSet(
	e *storage.Entity) (ReservationSet, *api.JsonErr) {
	reservationSet := ReservationSet{
		Key:              e.RowKey,
		ETag:             e.OdataEtag,
		ReservationSetID: ExtractStringProperty(e, reservationSetIDProperty),
		NetworkID:        ExtractStringProperty(e, networkIDProperty),
		SubnetName:       ExtractStringProperty(e, subnetNameProperty),
	}

	var rsRequest api.ReservationSetRequest
	if err := json.Unmarshal([]byte(e.Properties[reservationSetRequestProperty].(string)), &rsRequest); err != nil {
		return ReservationSet{}, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
			"could not deserialize reservation set request %s: %v", reservationSetKey, err)
	}

	reservationSet.ReservationSetRequest = rsRequest

	state := ExtractStringProperty(e, stateProperty)
	reservationSet.State = StringToReservationSetState(state)

	status := ExtractStringProperty(e, statusProperty)
	reservationSet.Status = StringToStatus(status)

	errorCode, err := ExtractFloat64Property(e, failureErrorCodeProperty)
	if err != nil {
		return reservationSet, api.InternalServerErrorf(api.StatusInternalServerError, err.Error())
	}
	reservationSet.FailureDetail.Code = api.ErrorCode(int(errorCode))

	reservationSet.FailureDetail.Text = ExtractStringProperty(e, failureErrorMsgProperty)

	errorCode, err = ExtractFloat64Property(e, failureHttpCodeProperty)
	if err != nil {
		return reservationSet, api.InternalServerErrorf(api.StatusInternalServerError, err.Error())
	}
	reservationSet.FailureDetail.HTTPStatus = int(errorCode)

	markedForDelete, err := ExtractBoolProperty(e, markedForDeleteProperty)
	if err != nil {
		return reservationSet, api.InternalServerErrorf(api.StatusInternalServerError, err.Error())
	}
	reservationSet.MarkedForDelete = markedForDelete

	successfulReservationCount, err := ExtractFloat64Property(e, successfulReservationCountProperty)
	if err != nil {
		return reservationSet, api.InternalServerErrorf(api.StatusInternalServerError, err.Error())
	}
	reservationSet.SuccessfulReservationCount = int(successfulReservationCount)

	return reservationSet, nil
}

func (rs *ReservationSet) ToStorageEntity(
	dncPartitionKey string) (storage.Entity, *api.JsonErr) {
	e := storage.Entity{
		Table:        &storage.Table{Name: reservationSetTableName},
		PartitionKey: dncPartitionKey,
		RowKey:       rs.Key,
		OdataEtag:    rs.ETag,
		Properties: map[string]interface{}{
			reservationSetIDProperty:           rs.ReservationSetID,
			networkIDProperty:                  rs.NetworkID,
			subnetNameProperty:                 rs.SubnetName,
			stateProperty:                      rs.State.String(),
			statusProperty:                     rs.Status.String(),
			successfulReservationCountProperty: rs.SuccessfulReservationCount,
			failureErrorCodeProperty:           rs.FailureDetail.Code,
			failureErrorMsgProperty:            rs.FailureDetail.Text,
			failureHttpCodeProperty:            rs.FailureDetail.HTTPStatus,
			markedForDeleteProperty:            rs.MarkedForDelete,
		},
	}

	jsonSerializedBytes, err := json.Marshal(rs.ReservationSetRequest)
	if err != nil {
		return storage.Entity{}, api.InternalServerErrorf(api.JsonMarshallingFailed,
			"could not marshal reservation set request: %+v. error: %v", rs.ReservationSetRequest, err)
	}

	e.Properties[reservationSetRequestProperty] = string(jsonSerializedBytes)

	return e, nil
}

func (r *ReservationSetRepo) PersistReservationSet(
	rs *ReservationSet) *api.JsonErr {
	entity, err := rs.ToStorageEntity(r.dncPartitionKey)
	if err != nil {
		r.logger.Errorf("could not translate reservation set to storage entity. reservation set: %+v error: %v", rs, err)
		return err
	}

	// todo: add InsertEntity func
	storageEntities := []storage.Entity{entity}
	if err := r.dataStore.UpsertEntityBatch(reservationSetTableName, storageEntities); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if ok {
			return &api.JsonErr{
				Code:       api.AzureStorageError,
				Text:       storageErr.Message,
				HTTPStatus: storageErr.StatusCode}
		}

		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	return nil
}

func (r *ReservationSetRepo) DeleteReservationSet(
	networkID string,
	reservationSetID string) *api.JsonErr {
	reservationSetKey := GetReservationSetKey(networkID, reservationSetID)
	if err := r.dataStore.Delete(reservationSetTableName, reservationSetKey); err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not delete reservation set: %s. Error: %s", reservationSetKey, err)
	}

	return nil
}

func (r *ReservationSetRepo) CreateReservationSet(
	networkID string,
	reservationSetID string,
	rsRequest api.ReservationSetRequest) *api.JsonErr {
	reservationSet := ReservationSet{
		Key:                   GetReservationSetKey(networkID, reservationSetID),
		ReservationSetID:      reservationSetID,
		NetworkID:             networkID,
		SubnetName:            rsRequest.SubnetName,
		State:                 Create,
		Status:                Pending,
		ReservationSetRequest: rsRequest,
		MarkedForDelete:       false,
	}

	if err := r.PersistReservationSet(&reservationSet); err != nil {
		r.logger.Errorf("Failed to persist the reservation set: %s. error: %v", reservationSetID, err)
		return err
	}

	return nil
}
