package datamodel

import (
	"dnc/api"
	"dnc/datastore/azurestorage"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"encoding/json"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/google/uuid"
	"net/http"
)

// Properties in tables
const (
	reservationConfigProperty = "ReservationConfiguration"
	subnetKeyProperty         = "SubnetKey"
	setKeyProperty            = "ReservationSetKey"
)

type ReservationRepo struct {
	dataStore       datastoreshim.DataStoreShim
	dncPartitionKey string
	logger          loggershim.LoggerInterface
}

func NewReservationRepo(
	dataStore       datastoreshim.DataStoreShim,
	dncPartitionKey string,
	logger          loggershim.LoggerInterface) *ReservationRepo {
	return &ReservationRepo{
		dncPartitionKey: dncPartitionKey,
		dataStore:       dataStore,
		logger:          logger,
	}
}

// Todo: Move api from networkManager to reservationRepo
func (*ReservationRepo) TryQueryIPReservation(networkID, reservationID string) (ReservationRow, *api.JsonErr) {
	return TryQueryIPReservation(networkID, reservationID)
}

func (*ReservationRepo) TryQueryIPReservationByKey(key string) (ReservationRow, *api.JsonErr) {
	return TryQueryIPReservationByKey(key)
}

// deprecated: use PersistIPReservation
func (*ReservationRepo) PutIPReservationConfiguration(res *ReservationConfiguration) *api.JsonErr {
	return PutIPReservationConfiguration(res)
}

func (r *ReservationRepo) UpdateReservations(reservations []ReservationRow) *api.JsonErr {
	return UpdateIPReservations(reservations)
}

func (r *ReservationRepo) DeleteReservation(reservationKey string) *api.JsonErr {
	deleteErr := r.dataStore.Delete(ipReservationsTableName, reservationKey)
	if deleteErr != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not delete configuration for reservationKey: %s. Error: %s", reservationKey, deleteErr)
	}

	return nil
}

//todo: move all reservation repo code here
func (r *ReservationRepo) TryQueryAllReservationsInSet(
	networkID        string,
	reservationSetID string) ([]ReservationRow, *api.JsonErr) {
	reservationSetKey := []string{GetReservationSetKey(networkID, reservationSetID)}
	filter := azurestorage.BuildFilterOnProperty(dncPartitionKey, setKeyProperty, reservationSetKey)
	return tryQueryReservations(filter)
}

func (r *ReservationRepo) PersistIpReservationRequest(
	networkID string,
	reservationID string,
	allocationRequest api.AllocationRequest,
	apiVersion string) *api.JsonErr {
	return PersistIPReservationRequest(networkID, reservationID, allocationRequest, apiVersion)
}

func (r *ReservationRepo) GetUnassignedReservationsInSet(
	networkID string,
	reservationSetID string) ([]ReservationRow, *api.JsonErr) {
	reservations, err := r.TryQueryAllReservationsInSet(networkID, reservationSetID)
	if err != nil {
		return nil, err
	}

	var unassignedReservations []ReservationRow
	for _, reservation := range reservations {
		if reservation.Configuration.AssociatedNCID == "" {
			unassignedReservations = append(unassignedReservations, reservation)
		}
	}

	return unassignedReservations, nil
}

func (r *ReservationRepo) GetFailedReservationsInSet(
	networkID string,
	reservationSetID string) ([]ReservationRow, *api.JsonErr) {
	reservations, err := r.TryQueryAllReservationsInSet(networkID, reservationSetID)
	if err != nil {
		return nil, err
	}

	var failedReservations []ReservationRow
	for _, reservation := range reservations {
		if reservation.Configuration.Status == Failed {
			failedReservations = append(failedReservations, reservation)
		}
	}

	return failedReservations, nil
}

func (r *ReservationRepo) AssociateNCWithReservationInSet(
	networkID string,
	ncID string,
	reservationSetID string) (string, *api.JsonErr) {
	reservations, err := r.TryQueryAllReservationsInSet(networkID, reservationSetID)
	if err != nil {
		return "", api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve reservations in set: %s. error: %v", reservationSetID, err)
	}

	// Check if a reservation is already associated with this NC.
	for _, reservation := range reservations {
		if reservation.Configuration.AssociatedNCID == ncID {
			return reservation.Configuration.ReservationID, nil
		}
	}

	var reservationToAssociate ReservationRow
	for _, reservation := range reservations {
		// Get the reservationID which is not associated with any NC.
		if reservation.Configuration.Status == Completed && reservation.Configuration.AssociatedNCID == "" {
			reservationToAssociate = reservation
		}
	}

	if reservationToAssociate.RowKey == "" {
		return "", api.BadRequestf(api.ReservationNotFound, "could not find any free reservation in set: %s", reservationSetID)
	}

	// Update the reservation record.
	reservationToAssociate.Configuration.AssociatedNCID = ncID
	if err = r.PersistIPReservation(reservationToAssociate); err != nil {
		return "", err
	}

	return reservationToAssociate.Configuration.ReservationID, nil
}

func (r *ReservationRepo) GetNumReservationsToCreateInSet(
	networkID                string,
	reservationSetID         string,
	requestedNumReservations int) (int, *api.JsonErr) {
	existingReservations, err := r.TryQueryAllReservationsInSet(networkID, reservationSetID)
	if err != nil {
		return 0, err
	}

	numReservationsToCreate := requestedNumReservations - len(existingReservations)
	if numReservationsToCreate == 0 {
		r.logger.Debugf("reservations are already created for the reservation set: %s", reservationSetID)
	}

	return numReservationsToCreate, nil
}

// GetPendingReservationsInSet returns all the reservations in the set which are in "Process - Pending" state
func (r *ReservationRepo) GetPendingReservationsInSet(
	networkID string,
	reservationSetID string) ([]ReservationRow, *api.JsonErr) {
	reservations, err := r.TryQueryAllReservationsInSet(networkID, reservationSetID)
	if err != nil {
		r.logger.Errorf("could not retrieve reservations in the set: %s. error: %v", reservationSetID, err)
		return nil, err
	}

	var pendingReservations []ReservationRow
	for _, reservation := range reservations {
		config := reservation.Configuration
		if config != nil && config.ReservationState == Process && config.Status == Pending {
			pendingReservations = append(pendingReservations, reservation)
		}
	}

	return pendingReservations, nil
}

func (r *ReservationRepo) CreateReservationsInSet(
	networkID string,
	subnetName string,
	reservationSetID string,
	requestedNumReservations int,
	apiVersion string) *api.JsonErr {
	numReservationsToCreate, err := r.GetNumReservationsToCreateInSet(networkID, reservationSetID, requestedNumReservations)
	if err != nil {
		return err
	}

	if numReservationsToCreate <= 0 {
		return nil
	}

	reservationsToCreate := make([]ReservationRow, numReservationsToCreate)
	for i := 0; i < numReservationsToCreate; i++ {
		reservationID := uuid.New().String()
		reservationsToCreate[i] = ReservationRow{
			RowKey: GetReservationKey(networkID, reservationID),
			Configuration: &ReservationConfiguration{
				ReservationID:        reservationID,
				NetworkID:            networkID,
				ReservationState:     Process,
				Status:               Pending,
				AllocationRequest:    api.AllocationRequest{SubnetName: subnetName},
				APIVersion:           apiVersion,
			},
			SubnetKey: GetSubnetKey(networkID, subnetName),
			SetKey: GetReservationSetKey(networkID, reservationSetID),
		}
	}

	if err = r.PersistIPReservations(reservationsToCreate); err != nil {
		return err
	}

	r.logger.Debugf("successfully persisted reservation request batch of size: %d in the set: %s",
		numReservationsToCreate, reservationSetID)

	return nil
}

func (r *ReservationRow) ToStorageEntity(dncPartitionKey string) (storage.Entity, *api.JsonErr) {
	e := storage.Entity{
		Table:         &storage.Table{Name: ipReservationsTableName},
		PartitionKey:  dncPartitionKey,
		RowKey:        r.RowKey,
		//todo: Add ETag to ReservationRow
		//OdataEtag:     r.ETag,
	}

	jsonSerializedBytes, err := json.Marshal(r.Configuration)
	if err != nil {
		return storage.Entity{}, api.InternalServerErrorf(api.JsonMarshallingFailed,
			"could not serialize reservation configuration: %s. Error: %v", r.Configuration.String(), err)
	}

	e.Properties = map[string]interface{}{
		reservationConfigProperty: string(jsonSerializedBytes[:]),
		subnetKeyProperty:         r.SubnetKey,
		setKeyProperty:            r.SetKey,
	}

	return e, nil
}

// PersistIPReservations - Persist IP Reservation batch
func (r *ReservationRepo) PersistIPReservations(reservations []ReservationRow) *api.JsonErr {
	storageEntities := make([]storage.Entity, len(reservations))
	for i, reservation := range reservations {
		se, err := reservation.ToStorageEntity(r.dncPartitionKey)
		if err != nil {
			return err
		}
		storageEntities[i] = se
	}

	if err := r.dataStore.UpsertEntityBatch(ipReservationsTableName, storageEntities); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if ok {
			return &api.JsonErr{
				Code:       api.AzureStorageError,
				Text:       storageErr.Message,
				HTTPStatus: storageErr.StatusCode}
		}

		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	r.logger.Debugf("Persisting IP reservations batch succeeded")

	return nil
}

// PersistIPReservation - Persist IP Reservation
func (r *ReservationRepo) PersistIPReservation(reservation ReservationRow) *api.JsonErr {
	//todo: add implementation for InsertOrUpdateEntity and use it here
	se, err := reservation.ToStorageEntity(r.dncPartitionKey)
	if err != nil {
		return err
	}

	storageEntities := []storage.Entity{se}
	if err := r.dataStore.UpsertEntityBatch(ipReservationsTableName, storageEntities); err != nil {
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

func (r *ReservationRepo) AnyReservationConfigurationExists() (bool, *api.JsonErr) {
	exists, err := r.dataStore.AnyRowExists(ipReservationsTableName)
	if err != nil {
		return false, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to Query networkContainerTableName due to error: %+v",
			err)
	}

	return exists, nil
}

// TryQueryAllReservations returns all the reservations in the dnc partition
func (r *ReservationRepo) TryQueryAllReservations() ([]ReservationRow, *api.JsonErr) {
	var context interface{}
	var rows []ReservationRow
	for {
		entities, hasMoreData, cx, status, err := r.dataStore.QueryAllRows(ipReservationsTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return []ReservationRow{}, nil
			}

			return nil, api.InternalServerErrorf(api.StatusInternalServerError,
				"Failed to Query all ipReservationsTableName rows due to error: %+v",
				err)
		}

		for _, entity := range entities {
			var reservationConfiguration ReservationConfiguration
			if err = json.Unmarshal([]byte(entity.Properties[reservationConfigProperty].(string)), &reservationConfiguration); err != nil {
				return []ReservationRow{}, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
					"could not deserialize reservation %s. Error: %v", entity.RowKey, err)
			}

			row := ReservationRow{
				RowKey:        entity.RowKey,
				Configuration: &reservationConfiguration,
			}

			if prop, ok := entity.Properties[subnetKeyProperty]; ok {
				row.SubnetKey = prop.(string)
			}

			if prop, ok := entity.Properties[setKeyProperty]; ok {
				row.SetKey = prop.(string)
			}

			rows = append(rows, row)
		}

		if !hasMoreData {
			break
		}
		context = cx
	}

	return rows, nil
}

func (r *ReservationRepo) DeleteAllRows() *api.JsonErr {
	err := r.dataStore.WipePartition(ipReservationsTableName)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to wipe ipReservationsTableName due to error: %+v",
			err)
	}

	return nil
}
